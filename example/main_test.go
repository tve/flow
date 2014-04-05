package main

import "github.com/jcw/flow-dev"

func Example() {
	g := flow.NewCircuit()
	g.Add("clock", "Clock")
	g.Add("pipe", "Pipe")
	g.Add("printer", "Printer")
	g.Add("repeater", "Repeater")
	g.Add("sink", "Sink")
	g.Add("timer", "Timer")
	g.Run()
	// Output:
}
