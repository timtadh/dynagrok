package main

import (
	"fmt"
)

import (
	"github.com/timtadh/dot"
	"github.com/timtadh/combos"
)

var text = `
digraph g {
	a [label="wizard"]
	a -> b [weight=.23];
}
`

func main() {
	p := newParser()
	err := dot.StreamParse([]byte(text), p)
	if err != nil {
		panic(err)
	}
}

type dotParser struct{}

func newParser() *dotParser {
	return &dotParser{}
}

func (p *dotParser) Enter(name string, n *combos.Node) error {
	fmt.Println("enter", name, n)
	return nil
}

func (p *dotParser) Stmt(n *combos.Node) error {
	fmt.Println("stmt", n)
	return nil
}

func (p *dotParser) Exit(name string) error {
	fmt.Println("exit", name)
	return nil
}
