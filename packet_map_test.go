package flow

import (
        "bytes"
        "math"
        "testing"
)


var pm = PacketMap{"s": "ok", "i": 2, "f": 123.1, "f1": float32(99.9),
                "f2": float64(-4.4), "b": []byte{1,2,3}}

func TestPacketMapString(t *testing.T) {
        tests := map[string]string{"s":"ok", "xx":"", "i":""}

        for k, v := range tests {
                r := pm.String(k) 
                if r != v {
                        t.Errorf("getting '%s' failed: got '%s' instead of '%s'", k, r, v)
                }
        }
}

func TestPacketMapInt(t *testing.T) {
        tests := map[string]int{"s":0, "xx":0, "i":2, "f":123, "f1":99, "f2":-4}

        for k, v := range tests {
                r := pm.Int(k) 
                if r != v {
                        t.Errorf("getting '%s' failed: got '%v' instead of '%v'", k, r, v)
                }
        }
}

func TestPacketMapFloat64(t *testing.T) {
        tests := map[string]float64{"s":math.NaN(), "xx":math.NaN(), "i":2.0, "f":123.1, "f1":99.9, "f2":-4.4}

        for k, v := range tests {
                r := pm.Float64(k) 
                if math.Abs(r-v) > 0.001 && !(math.IsNaN(r) && math.IsNaN(v)) {
                        t.Errorf("getting '%s' failed: got '%v' instead of '%v'", k, r, v)
                }
        }
}


func TestPacketMapBytes(t *testing.T) {
        tests := map[string][]byte{"s": []byte{'o', 'k'},
                "xx":nil, "i":nil, "f":nil, "f1":nil, "f2":nil,
                "b": []byte{1, 2, 3} }

        for k, v := range tests {
                r := pm.Bytes(k) 
                if !bytes.Equal(r, v) {
                        t.Errorf("getting '%s' failed: got '%v' instead of '%v'", k, r, v)
                }
        }
}

