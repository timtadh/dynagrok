package main

import (
	"fmt"
	"io/ioutil"
	"os"
)

import (
	"github.com/timtadh/combos"
	"github.com/timtadh/dot"
)

func main() {
	bytes, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}
	p := newParser()
	err = dot.StreamParse(bytes, p)
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
