package main

import "log"

type Node struct {
	Op   int
	Left *Node
	Type *Type
}

type Type struct{}

const (
	OXDOT = iota
	ODOT
	OPAREN
	OCONVNOP
	OINDEX
)

func (t *Type) IsArray() bool {
	return false
}

// what's the outer value that a write to n affects?
// outer value means containing struct or array.
func outervalue(n *Node) *Node {
	for {
		switch n.Op {
		case OXDOT:
			log.Fatalf("OXDOT in walk")
		case ODOT, OPAREN, OCONVNOP:
			n = n.Left
			continue
		case OINDEX:
			if n.Left.Type != nil && n.Left.Type.IsArray() {
				n = n.Left
				continue
			}
		}
		return n
	}
}

func main() {
	n := &Node{
		Op: 12,
	}
	outervalue(n)
	log.Fatalf("Hello")
}
