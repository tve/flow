package flow

import (
	"strings"
	"sync"

	"github.com/golang/glog"
)

// Initialise a new circuit.
func NewCircuit() *Circuit {
	return &Circuit{
		gadgets: map[string]*Gadget{},
		feeds:   map[string][]Message{},
		labels:  map[string]string{},
                abort:   make(chan struct{}),
                abortOnce: new(sync.Once),
	}
}

// A circuit is a collection of inter-connected gadgets.
type Circuit struct {
	Gadget

	gnames  []gadgetDef          // gadgets added by name from the registry
	gadgets map[string]*Gadget   // gadgets added to this circuit
	wires   []wireDef            // list of all connections
	feeds   map[string][]Message // message feeds
	labels  map[string]string    // pin label lookup map

        abort   chan struct{}        // closing this channel aborts the circuit
        abortOnce *sync.Once         // ensure we only close the abort channel once
	wait      sync.WaitGroup     // tracks number of running gadgets
}

// definition of one named gadget
type gadgetDef struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// definition of one connection
type wireDef struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Capacity int    `json:"capacity"`
}

// Add a named gadget to the circuit with a unique name.
func (c *Circuit) Add(name, gadget string) {
	constructor := Registry[gadget]
	if constructor == nil {
		glog.Warningln("not found:", gadget)
		return
	}
	c.gnames = append(c.gnames, gadgetDef{name, gadget})
	c.AddCircuitry(name, constructor())
}

// Add a gadget or circuit to the circuit with a unique name.
func (c *Circuit) AddCircuitry(name string, g Circuitry) {
	c.gadgets[name] = g.initGadget(g, name, c)
        // if we're adding a sub-circuit then inherit the abort channel
        if cc, ok := g.(*Circuit); ok {
                cc.abort = c.abort
                cc.abortOnce = c.abortOnce
        }
}

func (c *Circuit) gadgetOf(s string) *Gadget {
	// TODO: migth be useful for extending an existing circuit
	// if gadgetPart(s) == "" && c.labels[s] != "" {
	// 	s = c.labels[s] // unnamed gadgets can use the circuit's pin map
	// }
	g, ok := c.gadgets[gadgetPart(s)]
	if !ok {
		glog.Fatalln("gadget not found for:", s)
	}
	return g
}

// Return the "to" part of a wire given a "from" part
func (c *Circuit) DestOf(from string) string {
        for _, w := range c.wires {
                glog.V(4).Info(w.From, "==", from)
                if w.From == from {
                        return w.To
                }
        }
        glog.Warningln("wire not found:", from)
        return ""
}

// Connect an output pin with an input pin.
func (c *Circuit) Connect(from, to string, capacity int) {
	c.wires = append(c.wires, wireDef{from, to, capacity})
	w := c.gadgetOf(to).getInput(pinPart(to), capacity)
	c.gadgetOf(from).setOutput(pinPart(from), w)
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
	for _, g := range c.gadgets {
		g.launch()
	}
	c.wait.Wait()
}

// Start up one gadget in the circuit, useful after dynamically ading a gadget
func (c *Circuit) RunGadget(name string) {
        c.gadgets[name].launch()
}

// Abort the operation of a circuit
func (c *Circuit) Abort() {
        // signal the abort to all gadgets
        c.abortOnce.Do(func() {
                close(c.abort)
                glog.Warningf("Aborting circuit %s", c.name)
        })
}

// Return a description of this circuit in serialisable form.
func (c *Circuit) Describe() interface{} {
	desc := map[string]interface{}{}
	if len(c.gnames) > 0 {
		desc["gadgets"] = c.gnames
	}
	if len(c.gadgets) > len(c.gnames) {
		named := map[string]bool{}
		for _, n := range c.gnames {
			named[n.Name] = true
		}
		unreg := []string{}
		for k := range c.gadgets {
			if !named[k] {
				unreg = append(unreg, k)
			}
		}
		desc["unregistered"] = unreg
	}
	if len(c.wires) > 0 {
		desc["wires"] = c.wires
	}
	if len(c.feeds) > 0 {
		expanded := []map[string]Message{}
		for pin, feeds := range c.feeds {
			for _, m := range feeds {
				one := map[string]Message{}
				if t, ok := m.(Tag); ok {
					one["tag"] = t.Tag
					one["data"] = t.Msg
				} else {
					one["data"] = m
				}
				one["to"] = pin
				expanded = append(expanded, one)
			}
		}
		desc["feeds"] = expanded
	}
	if len(c.labels) > 0 {
		desc["labels"] = c.labels
	}
	return desc
}
