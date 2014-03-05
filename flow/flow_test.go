package flow_test

import (
	"strings"

	"github.com/jcw/flow/flow"
	_ "github.com/jcw/flow/workers"
)

func ExampleNewGroup() {
	g := flow.NewGroup()
	g.Run()
}

func ExampleTransformer() {
	upper := flow.Transformer(func(m flow.Memo) flow.Memo {
		return strings.ToUpper(m.(string))
	})

	g := flow.NewGroup()
	g.AddWorker("u", upper)
	g.Set("u.In", "abc")
	g.Run()
	// Output:
	// Lost string: ABC
}

func ExampleGroup_Map() {
	// new workgroup to repeat each incoming memo three times
	wg := flow.NewGroup()
	wg.Add("r", "Repeater")
	wg.Set("r.Num", 3)
	wg.Map("MyIn", "r.In")
	wg.Map("MyOut", "r.Out")

	g := flow.NewGroup()
	g.AddWorker("wg", wg)
	g.Set("wg.MyIn", "abc")

	g.Run()
	// Output:
	// Lost string: abc
	// Lost string: abc
	// Lost string: abc
}
