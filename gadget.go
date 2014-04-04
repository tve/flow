package flow

import (
	"reflect"
	// "strings"

	"github.com/golang/glog"
)

// Gadget keeps track of internal details about a gadget.
type Gadget struct {
	circuitry Circuitry                // pointer to self as Circuitry object
	name      string                   // name of this gadget in the circuit
	owner     *Circuit                 // owning circuit
	regType   string                   // type, as listed in the registry
	inputs    map[string]reflect.Value // input pins
	outputs   map[string]reflect.Value // output pins
}

// Disconnect an output channel, closing it when all refs are gone.
func (g *Gadget) Disconnect(c Output) {
	glog.Errorln("disconnect")
}

func (g *Gadget) initGadget(c Circuitry, n string, o *Circuit) *Gadget {
	if g.owner != nil {
		glog.Fatalln("gadget is already in use:", n)
	}
	g.circuitry = c
	g.name = n
	g.owner = o
	g.inputs = map[string]reflect.Value{}
	g.outputs = map[string]reflect.Value{}
	return g
}

func (g *Gadget) initPins() {
	gv := reflect.ValueOf(g.circuitry).Elem()
	for i := 0; i < gv.NumField(); i++ {
		ft := gv.Type().Field(i)
		fv := gv.Field(i)
		switch fv.Type().String() {
		case "flow.Input":
			g.inputs[ft.Name] = fv
		case "flow.Output":
			g.outputs[ft.Name] = fv
		}
	}
	glog.Errorln("inputs", g.inputs)
	glog.Errorln("outputs", g.outputs)
}

// func (g *Gadget) gadgetValue() reflect.Value {
// 	return reflect.ValueOf(g.circuitry).Elem()
// }

// func (g *Gadget) pinValue(pin string) reflect.Value {
// 	pp := pinPart(pin)
// 	// if it's a circuit, look up mapped pins
// 	if g, ok := g.circuitry.(*Circuit); ok {
// 		p := g.labels[pp]
// 		return g.gadgetOf(p).circuitry.pinValue(p) // recursive
// 	}
// 	fv := g.gadgetValue().FieldByName(pp)
// 	if !fv.IsValid() {
// 		glog.Fatalln("pin not found:", pin)
// 	}
// 	return fv
// }

// func (g *Gadget) getInput(pin string, capacity int) *wire {
// 	c := g.inputs[pin]
// 	if c == nil {
// 		c = &wire{channel: make(chan Message, capacity), dest: g}
// 		g.inputs[pin] = c
// 	}
// 	if capacity > c.capacity {
// 		c.capacity = capacity
// 	}
// 	return c
// }

// func (g *Gadget) setOutput(pin string, c *wire) {
// 	ppfv := strings.Split(pin, ":")
// 	fp := g.circuitry.pinValue(ppfv[0])
// 	if len(ppfv) == 1 {
// 		if !fp.IsNil() {
// 			glog.Fatalf("output already connected: %s.%s", g.name, pin)
// 		}
// 		setValue(fp, c)
// 	} else { // it's not an Output, so it must be a map[string]Output
// 		if fp.IsNil() {
// 			setValue(fp, map[string]Output{})
// 		}
// 		outputs := fp.Interface().(map[string]Output)
// 		if _, ok := outputs[ppfv[1]]; ok {
// 			glog.Fatalf("output already connected: %s.%s", g.name, pin)
// 		}
// 		outputs[ppfv[1]] = c.channel
// 	}
// 	c.senders++
// 	g.outputs[pin] = c
// }

func (g *Gadget) setupChannels() {
	// // make sure all the feed wires have also been set up
	// for dest, msgs := range g.owner.feeds {
	// 	if gadgetPart(dest) == g.name {
	// 		g.getInput(dest, len(msgs)) // will add wire to the inputs map
	// 	}
	// }

	// // set up and pre-fill all the input pins
	// for pin, wire := range g.inputs {
	// 	// create a channel with the proper capacity
	// 	wire.channel = make(chan Message, wire.capacity)
	// 	setValue(g.circuitry.pinValue(pin), wire.channel)
	// 	// fill it with messages from the feed inbox, if any
	// 	for _, msg := range g.owner.feeds[pin] {
	// 		wire.channel <- msg
	// 	}
	// 	// close the channel if there is no other feed
	// 	if wire.senders == 0 {
	// 		close(wire.channel)
	// 	}
	// }

	// // set dangling inputs to null input and dangling outputs to fake sink
	// gadget := g.gadgetValue()
	// for i := 0; i < gadget.NumField(); i++ {
	// 	field := gadget.Field(i)
	// 	switch field.Type().String() {
	// 	case "flow.Input":
	// 		if field.IsNil() {
	// 			null := make(chan Message)
	// 			close(null)
	// 			setValue(field, null)
	// 		}
	// 	case "flow.Output":
	// 		if field.IsNil() {
	// 			// FIXME: setValue(field, &fakeSink{})
	// 		}
	// 	}
	// }
}

func (g *Gadget) closeChannels() {
	// for _, wire := range g.outputs {
	// 	close(wire.channel)
	// }

	// for _, wire := range g.inputs {
	// 	// close channel if not nil and not already closed
	// 	if wire.channel != nil {
	// 		select {
	// 		case _, ok := <-wire.channel:
	// 			if ok {
	// 				close(wire.channel)
	// 			}
	// 		default:
	// 		}
	// 	}
	// 	// setValue(g.circuitry.pinValue(pin), wire.channel)
	// }
}

func (g *Gadget) launch() {
	g.owner.wait.Add(1)
	g.setupChannels()

	go func() {
		defer DontPanic()
		defer g.owner.wait.Done()
		defer g.closeChannels()

		g.circuitry.Run()
	}()
}

func setValue(value reflect.Value, any interface{}) {
	value.Set(reflect.ValueOf(any))
}
