package flow

import (
	"reflect"
	// "strings"

	"github.com/golang/glog"
)

// Gadget keeps track of internal details about a gadget.
type Gadget struct {
	circuitry Circuitry    // pointer to self as Circuitry object
	name      string       // name of this gadget in the circuit
	admin     chan Message // for communication with owning circuit
	regType   string       // type, as listed in the registry
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

func (g *Gadget) initPins(channels wiring) {
	gv := reflect.ValueOf(g.circuitry).Elem()
	for i := 0; i < gv.NumField(); i++ {
		ft := gv.Type().Field(i)
		ch := channels[g.name+"."+ft.Name]
		if ch != nil {
			glog.Errorln("g-hup", g.name, ft.Name, ch, cap(ch))
		}
		fv := gv.Field(i)
		switch fv.Type().String() {
		case "flow.Input":
			if ch == nil {
				ch = nullChan
			}
			setPin(fv, ch)
		case "flow.Output":
			if ch == nil {
				ch = channels[""] // special admin channel, used as sink
			}
			setPin(fv, ch)
		}
	}
}

func (g *Gadget) pinValue(pin string) reflect.Value {
	fv := reflect.ValueOf(g.circuitry).Elem().FieldByName(pin)
	if !fv.IsValid() {
		glog.Fatalln("pin not found:", pin)
	}
	return fv
}
