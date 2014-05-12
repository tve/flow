//
// Represent a packet as a map[string]interface{} so every gadget can pull the info it wants
// and add additional info to the packet for downstream gadgets.

package flow

import (
        "math"
        "time"
        "github.com/golang/glog"
)

type PacketMap map[string]interface{}

// Wrapper function to extract string value from PacketMap
func (pm PacketMap) String(key string) (string) {
        if v, ok := pm[key].(string); ok {
                return v
        } else {
                glog.Errorf("%s is not a string in PacketMap{%+v}", key, pm)
                return ""
        }
}

// Wrapper function to extract time value from PacketMap
func (pm PacketMap) Time(key string) (time.Time) {
        if v, ok := pm[key].(time.Time); ok {
                return v
        } else {
                glog.Errorf("%s is not a time in PacketMap{%+v}", key, pm)
                return time.Time{}
        }
}

// Wrapper function to extract int value from PacketMap
func (pm PacketMap) Int(key string) (int) {
        switch v := pm[key].(type) {
        case int:
                return v
        case float32:
                return int(v)
        case float64:
                return int(v)
        case byte:
                return int(v)
        default:
                glog.Errorf("%s is not an int in PacketMap{%+v}", key, pm)
                return 0
        }
}

// Wrapper function to extract float64 value from PacketMap
func (pm PacketMap) Float64(key string) (float64) {
        switch v := pm[key].(type) {
        case int:
                return float64(v)
        case float32:
                return float64(v)
        case float64:
                return float64(v)
        case byte:
                return float64(v)
        default:
                glog.Errorf("%s is not a float in PacketMap{%+v}", key, pm)
                return math.NaN()
        }
}

// Wrapper function to extract []byte value from PacketMap
func (pm PacketMap) Bytes(key string) ([]byte) {
        switch v := pm[key].(type) {
        case []byte:
                return v
        case string:
                return []byte(v)
        default:
                glog.Errorf("%s is not a float in PacketMap{%+v}", key, pm)
                return nil
        }
}

//===== PacketMapDispatcher =====

func init() {
	Registry["PacketMapDispatcher"] = func() Circuitry {
		c := NewCircuit()
		c.AddCircuitry("head", &pmDispatchHead{})
		c.AddCircuitry("tail", &pmDispatchTail{})
		c.Connect("head.Feeds:", "tail.In", 0)  // keeps tail alive
		c.Label("In", "head.In")
		c.Label("Prefix", "head.Prefix")
		c.Label("Field", "head.Field")
		c.Label("Rej", "head.Rej")
		c.Label("Out", "tail.Out")
		return c
	}
}

// Dispatch to a gadget based on a field in incoming PacketMaps
// Registers as "PacketMapDispatcher".
type PacketMapDispatcher Circuit

type pmDispatchHead struct {
	Gadget
	Prefix  Input         // Expects string with decoder gadget prefix
	Field   Input         // Expects string with field to dispatch on
	In      Input         // Expects PacketMaps with [field]:string
	Rej     Output        // Outputs rejected gadget names
        Feeds   map[string]Output  // Output leading to all the decoders
}

type pmDispatchTail struct {
        Gadget
	In      Input         // Expects PacketMaps coming back from decoders
	Out     Output        // Produces final output
}

func (g *pmDispatchTail) Run() {
        for m := range g.In {
                glog.V(2).Infof("Tail: %+v", m)
                g.Out.Send(m)
        }
        glog.Warningln("Input of pmDispatchTail %s was closed", g.Name())
}

// Dispatch incoming PacketMaps
func (g *pmDispatchHead) Run() {
	prefix := ""
	if p, ok := <-g.Prefix; ok {
		prefix = p.(string)
	}

	field := ""
	if p, ok := <-g.Field; ok {
		field = p.(string)
	}

	for m := range g.In {
                glog.V(4).Infof("In: %+v", m)
                glog.V(6).Infof("Feeds: %+v", g.Feeds)
                if v, ok := m.(PacketMap); ok {
                        if gadget := v.String(field); gadget != "" {
                                if _, ok := g.Feeds[gadget]; !ok {
                                        g.addGadget(prefix, gadget)
                                }
                                if feed, ok := g.Feeds[gadget]; ok && feed != nil {
                                        glog.V(1).Infof("Dispatch to %s", gadget)
                                        glog.V(4).Infof("Feed: %+v", m)
                                        v["decoder"] = gadget
                                        feed.Send(m)
                                        continue
                                }
                        }
                }
                glog.V(2).Infof("Out: %+v", m)
                g.Feeds[""].Send(m)
        }
}

func (g *pmDispatchHead) addGadget(prefix, key string) {
        pm := prefix + key
        if Registry[pm] == nil {
                glog.Warningf("gadget %s not found for dispatch", pm)
                g.Rej.Send(key) // report that no such gadget was found
                g.Feeds[key] = nil
        } else { // create, hook up, and launch the new gadget
                glog.Infof("hooking up %s for dispatch", pm)
                c := g.Owner()
                c.Add(pm, pm)
                c.Connect("head.Feeds:"+key, pm+".In", 0)
                c.Connect(pm+".Out", "tail.In", 0)
                c.RunGadget(pm)
                //glog.V(4).Infoln(self+".Feeds:"+key, "->", pm+".In")
                //glog.V(4).Infoln(pm+".Out", "->", dest)
        }
}
