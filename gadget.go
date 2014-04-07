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
	admin     chan Message             // for communication with owning circuit
	regType   string                   // type, as listed in the registry
}

// Release an output channel, closing it when all refs are gone.
func (g *Gadget) Release(c Output) {
	if g.admin != nil {
		glog.Infoln("release", g.name)
		g.admin <- adminMsg{g: g, o: c}
	} else {
		glog.Infoln("close", g.name)
		close(c) // not inside cicuit, assume we should just close the channel
	}
}

func (g *Gadget) initGadget(c Circuitry, n string) *Gadget {
	g.circuitry = c
	g.name = n
	return g
}

func (g *Gadget) initPins() map[string]interface{} {
	pins := map[string]interface{}{} // input and output pins
	gv := reflect.ValueOf(g.circuitry).Elem()
	for i := 0; i < gv.NumField(); i++ {
		ft := gv.Type().Field(i)
		fv := gv.Field(i)
		glog.Infoln("pin", g.name, ft.Name, fv.CanSet())
		switch fv.Type().String() {
		case "flow.Input", "flow.Output":
			pins[ft.Name] = fv
		}
	}
	glog.Infoln("pins", pins)
	return pins
}

func (g *Gadget) pinValue(pin string) reflect.Value {
	fv := reflect.ValueOf(g.circuitry).Elem().FieldByName(pin)
	if !fv.IsValid() {
		glog.Fatalln("pin not found:", pin)
	}
	return fv
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
