package flow

import (
	"strings"
	"sync"

	"github.com/golang/glog"
)

// Initialise a new circuit.
func NewCircuit() *Circuit {
	c := Circuit{
		gadgets: map[string]*Gadget{},
		wires:   map[string]int{},
		feeds:   map[string][]Message{},
		labels:  map[string]string{},
		null:    make(chan Message),
	}
	close(c.null)
	return &c
}

// A circuit is a collection of inter-connected gadgets.
type Circuit struct {
	Gadget

	gadgets map[string]*Gadget   // gadgets added to this circuit
	wires   map[string]int       // all wire definitions
	feeds   map[string][]Message // all message feeds
	labels  map[string]string    // pin label lookup map

	null chan Message // used for dangling inputs
	sink chan Message // used for dangling outputs

	wait sync.WaitGroup // tracks number of running gadgets
}

func (c *Circuit) initPins() {
	// fill c.inputs[]
	// fill c.outputs[]
}

// Add a named gadget to the circuit with a unique name.
func (c *Circuit) Add(name, gadget string) {
	constructor := Registry[gadget]
	if constructor == nil {
		glog.Warningln("not found:", gadget)
		return
	}
	g := c.AddCircuitry(name, constructor())
	g.regType = gadget
}

// Add a gadget or circuit to the circuit with a unique name.
func (c *Circuit) AddCircuitry(name string, g Circuitry) *Gadget {
	r := g.initGadget(g, name, c)
	c.gadgets[name] = r
	return r
}

// func (c *Circuit) gadgetOf(s string) *Gadget {
// 	// TODO: migth be useful for extending an existing circuit
// 	// if gadgetPart(s) == "" && c.labels[s] != "" {
// 	// 	s = c.labels[s] // unnamed gadgets can use the circuit's pin map
// 	// }
// 	g, ok := c.gadgets[gadgetPart(s)]
// 	if !ok {
// 		glog.Fatalln("gadget not found for:", s)
// 	}
// 	return g
// }

// Connect an output pin with an input pin.
func (c *Circuit) Connect(from, to string, capacity int) {
	// c.wires = append(c.wires, wireDef{from, to, capacity})
	// w := c.gadgetOf(to).getInput(pinPart(to), capacity)
	// c.gadgetOf(from).setOutput(pinPart(from), w)
	c.wires[from+"/"+to] = capacity
}

// Set up a message to feed to a gadget on startup.
func (c *Circuit) Feed(pin string, m Message) {
	c.feeds[pin] = append(c.feeds[pin], m)
}

// Label an external pin to map it to an internal one.
func (c *Circuit) Label(external, internal string) {
	if strings.Contains(external, ".") {
		glog.Fatalln("external pin should not include a dot:", external)
	}
	c.labels[external] = internal
}

// Start up the circuit, and return when it is finished.
func (c *Circuit) Run() {
	c.sink = make(chan Message)
	go func() {
		for m := range c.sink {
			glog.Infoln("lost:", m)
		}
	}()
	for w, c := range c.wires {
		_, _ = w, c // ...
	}
	for _, g := range c.gadgets {
		g.launch()
	}
	c.wait.Wait()
	close(c.sink)
}
