package flow

import (
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
	admin   chan Message		 // admin channel while running
}

func (c *Circuit) initPins() {
	// fill c.inputs[]
	// fill c.outputs[]
	glog.Errorln("c-initpins", c.name)
}

// Add a named gadget to the circuit with a unique name.
func (c *Circuit) Add(name, gadget string) {
	constructor := Registry[gadget]
	if constructor == nil {
		glog.Errorln("not found:", gadget)
		return
	}
	g := c.AddCircuitry(name, constructor())
	g.regType = gadget
}

// Add a gadget or circuit to the circuit with a unique name.
func (c *Circuit) AddCircuitry(name string, circ Circuitry) *Gadget {
	g := circ.initGadget(circ, name)
	c.gadgets[name] = g
	circ.initPins()
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

type wire struct {
	fanIn   int
	channel chan Message
}

type adminMsg struct {
	g *Gadget
	o Output
}

// Start up the circuit, and return when it is finished.
func (c *Circuit) Run() {
	inbound := map[string]*wire{}
	outbound := map[string]*wire{}

	// start by creating channels large enough to contain the feed data
	glog.Errorln("c-init", c.name, len(c.feeds))
	for k, v := range c.feeds {
		inbound[k] = &wire{channel: make(chan Message, len(v))}
	}

	// collect all wire endpoints, increasing wire capacities as needed
	glog.Errorln("wires", c.name, len(c.wires))
	for wpair, wcap := range c.wires {
		v := strings.Split(wpair, "/")
		from := v[0]
		to := v[1]
		glog.Errorln("wire", wpair, wcap)
		if _, ok := inbound[to]; !ok {
			inbound[to] = &wire{}
		}
		in := inbound[to]
		if cap(in.channel) < wcap {
			in.channel = make(chan Message, wcap) // replace with larger one
		}
		outbound[from] = in
	}

	// push the feed dato into the channels
	glog.Errorln("feeds", c.name, len(c.feeds))
	for k, v := range c.feeds {
		glog.Errorln("feed", k, v)
		for _, f := range v {
			inbound[k].channel <- f
		}
	}

	// set up an admin channel for communication from gadgets to this circuit
	glog.Errorln("admin", c.name)
	c.admin = make(chan Message)
	count := len(c.gadgets)
	
	// set up all the gadgets and start them up
	glog.Errorln("gadgets", c.name, len(c.gadgets))
	for _, g := range c.gadgets {
		g.admin = c.admin
		
		glog.Errorln("g-in", g.name, len(g.inputs))
		for k, v := range g.inputs {
			if in, ok := inbound[k]; ok {
				setPin(v, in.channel)
			} else {
				setPin(v, nullChan) // feed eof to unconnected inputs
			}
		}

		glog.Errorln("g-out", g.name, len(g.outputs))
		for k, v := range g.outputs {
			if out, ok := outbound[k]; ok {
				out.fanIn++
				setPin(v, out.channel)
			} else {
				setPin(v, c.admin) // ignore data from unconnected outputs
			}
		}

		glog.Errorln("g-close", g.name)
		for _, in := range inbound {
			if in.fanIn == 0 {
				close(in.channel)
			}
		}

		glog.Errorln("g-go", g.name)
		go func() {
			defer DontPanic()
			defer func() {
				c.admin <- adminMsg{g: g}
			}()

			glog.Errorln("g-run", g.name)
			g.circuitry.Run()
			glog.Errorln("g-end", g.name)
		}()
	}
	
	// listen for incoming admin requests until all gadgets have finished
	if count > 0 { // TODO: this check can probably move up
		for m := range c.admin {
			glog.Errorln("g-admin", m)
			if a, ok := m.(adminMsg); ok {
				// also use for disconnects and live circuit rewiring?
				glog.Errorln("g-finish", a.g.name)
				// teardown pins
				count--
				if count == 0 {
					close(c.admin) // will terminate the loop
				}
			} else {
				// all other messages are from unconnected output pins
				glog.Errorln("lost:", c.name, m)
			}
		}
	}
	
	glog.Errorln("g-done", c.name)
	c.admin = nil // this also marks the circuit as not running
}

func setPin(v reflect.Value, c chan Message) {
	v.Set(reflect.ValueOf(c))
}
