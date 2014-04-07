package flow

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/golang/glog"
)

var nullChan = make(chan Message)

func init() {
	// set up a null channel which always returns eof
	close(nullChan)
}

// Initialise a new circuit.
func NewCircuit() *Circuit {
	return &Circuit{
		gadgets: map[string]*Gadget{},
		wires:   map[string]int{},
		feeds:   map[string][]Message{},
		labels:  map[string]string{},
	}
}

// A circuit is a collection of inter-connected gadgets.
type Circuit struct {
	Gadget

	gadgets map[string]*Gadget   // gadgets added to this circuit
	wires   map[string]int       // all wire definitions
	feeds   map[string][]Message // all message feeds
	labels  map[string]string    // pin label lookup map
}

func (c *Circuit) initPins() map[string]interface{} {
	glog.Infoln("c-initpins", c.name)
	gmap := map[string]chan Message{}

	// make sure there is enough room to store all feed messages
	for in, feed := range c.feeds {
		gmap[in] = make(chan Message, len(feed))
	}
	// first pass assigns a channel to each input pin
	for wpair, wcap := range c.wires {
		v := strings.Split(wpair, "/")
		in := v[1]
		// increase wire capacity if needed
		if gmap[in] == nil || wcap > cap(gmap[in]) {
			gmap[in] = make(chan Message, wcap)
		}
	}
	// second pass assigns the proper channel to all output pins
	for wpair := range c.wires {
		v := strings.Split(wpair, "/")
		out := v[0]
		if gmap[out] != nil {
			glog.Fatalln("output already connected", c.name, out)
		}
		in := v[1]
		gmap[out] = gmap[in]
	}
	glog.Errorln("gmap", gmap)

	// look up the label definitions
	pins := map[string]interface{}{}
	for k, v := range c.labels {
		if gmap[v] == nil {
			gmap[v] = make(chan Message)
		}
		pins[k] = v
	}
	glog.Errorln("pins", pins)
	return pins
}

// Add a named gadget to the circuit with a unique name.
func (c *Circuit) Add(name, gadget string) {
	constructor := Registry[gadget]
	if constructor == nil {
		glog.Errorln("not found", gadget)
		return
	}
	g := c.AddCircuitry(name, constructor())
	g.regType = gadget
}

// Add a gadget or circuit to the circuit with a unique name.
func (c *Circuit) AddCircuitry(name string, child Circuitry) *Gadget {
	g := child.initGadget(child, name)
	c.gadgets[name] = g
	return g
}

// Connect an output pin with an input pin.
func (c *Circuit) Connect(from, to string, capacity int) {
	c.wires[from+"/"+to] = capacity
}

// Set up a message to feed to a gadget on startup.
func (c *Circuit) Feed(pin string, m Message) {
	c.feeds[pin] = append(c.feeds[pin], m)
}

// Label an external pin to map it to an internal one.
// TODO: get rid of this, use wires with undotted in or out pins instead
func (c *Circuit) Label(external, internal string) {
	if strings.Contains(external, ".") {
		glog.Fatalln("external pin should not include a dot:", external)
	}
	c.labels[external] = internal
}

type group struct {
	fanIn    int          // number of attached output pins
	fanOut   int          // number of attached input pins
	capacity int          // channel buffering capacity
	channel  chan Message // actual channel for this group
}

type adminMsg struct {
	g *Gadget
	o Output
}

// Start up the circuit, and return when it is finished.
func (c *Circuit) Run() {
	inbound := map[string]*group{}
	outbound := map[string]*group{}

	// start by creating channels large enough to contain the feed data
	glog.Infoln("c-init", c.name, len(c.feeds))
	for k, v := range c.feeds {
		inbound[k] = &group{channel: make(chan Message, len(v))}
	}

	// collect all wire endpoints, increasing wire capacities as needed
	glog.Infoln("wires", c.name, len(c.wires))
	for wpair, wcap := range c.wires {
		v := strings.Split(wpair, "/")
		from := v[0]
		to := v[1]
		glog.Infoln("wire", wpair, wcap)
		if _, ok := inbound[to]; !ok {
			inbound[to] = &group{}
		}
		in := inbound[to]
		if cap(in.channel) < wcap {
			in.channel = make(chan Message, wcap) // replace with larger one
		}
		outbound[from] = in
	}

	glog.Infoln("inbound", inbound)
	glog.Infoln("outbound", outbound)

	// push the feed dato into the channels
	glog.Infoln("feeds", c.name, len(c.feeds))
	for k, v := range c.feeds {
		glog.Infoln("feed", k, v)
		for _, f := range v {
			inbound[k].channel <- f
		}
	}

	// set up an admin channel for communication from gadgets to this circuit
	glog.Infoln("admin", c.name)
	admin := make(chan Message)
	count := 0

	// set up all the gadgets and start them up
	glog.Infoln("gadgets", c.name, len(c.gadgets))
	for _, g := range c.gadgets {
		count++
		g.admin = admin

		// set pins to a valid channel, or source from null, or sink to admin
		pins := g.circuitry.initPins()
		glog.Infoln("g-pins", g.name, len(pins))
		for k, p := range pins {
			if s, ok := p.(string); ok { // aliased label into the child circuit
				// gc := g.circuitry.(*Circuit)
				// vs := v.Interface().(string)
				// if in, ok := gc.inbound[vs]; ok {
				// 	t = "flow.Input"
				// 	v := reflect.ValueOf(in.channel)
				// }
				// if out, ok := gc.outbound[vs]; ok {
				// 	t = "flow.Output"
				// 	v := reflect.ValueOf(out.channel)
				// }
				glog.Errorln("label", s, k)
				continue
			}
			v := p.(reflect.Value)
			t := v.Type().String()
			switch t {
			case "flow.Input":
				if in, ok := inbound[g.name+"."+k]; ok {
					glog.Infoln("inpin", k)
					setPin(v, in.channel)
				} else {
					glog.Infoln("null", k)
					setPin(v, nullChan) // feed eof to unconnected inputs
				}
			case "flow.Output":
				if out, ok := outbound[g.name+"."+k]; ok {
					out.fanIn++
					glog.Infoln("outpin", k, out.fanIn)
					setPin(v, out.channel)
				} else {
					glog.Infoln("sink", k)
					setPin(v, admin) // ignore data from unconnected outputs
				}
			}
		}

		// close all channels for this gadget which have no outputs feeding in
		glog.Infoln("g-close", g.name)
		for k, in := range inbound {
			if strings.HasPrefix(k, g.name+".") && in.fanIn == 0 {
				glog.Infoln("in-close", k)
				close(in.channel)
			}
		}

		// start the gadget as goroutine
		glog.Infoln("g-go", g.name)
		go func(g *Gadget) {
			defer func() {
				admin <- adminMsg{g: g}
			}()
			defer DontPanic()

			glog.Infoln("g-run", g.name)
			g.circuitry.Run()
			glog.Infoln("g-end", g.name)
		}(g)
	}

	// listen for incoming admin requests until all gadgets have finished
	for count > 0 {
		m := <-admin
		glog.Infoln("g-admin", m)
		if a, ok := m.(adminMsg); ok {
			// also use for output pin releases and live circuit rewiring
			glog.Infoln("g-finish", a.g.name)
			// teardown pins
			count-- // will eventually terminate the loop
		} else {
			// all other messages are from unconnected output pins
			glog.Warningln("lost:", c.name, m)
			fmt.Printf("Lost %T: %v\n", m, m)
		}
	}

	// close(admin)
	glog.Infoln("g-done", c.name)
}

func setPin(v reflect.Value, c chan Message) {
	v.Set(reflect.ValueOf(c))
}
