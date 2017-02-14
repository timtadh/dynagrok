package lattice

import (
	"fmt"
)

import ()

import (
	"github.com/timtadh/dynagrok/localize/lattice/subgraph"
)

type Node struct {
	l *Lattice
	SubGraph *subgraph.SubGraph
	Embeddings subgraph.Embeddings
}

func NewNode(l *Lattice, sg *subgraph.SubGraph, embs subgraph.Embeddings) *Node {
	return &Node{
		l: l,
		SubGraph: sg,
		Embeddings: embs,
	}
}

func (n *Node) String() string {
	if n.SubGraph == nil {
		return "<Node {0:0}>"
	}
	return fmt.Sprintf("<Node %v %v>", len(n.Embeddings), n.SubGraph.Pretty(n.l.Labels))
}

func (n *Node) Children() (nodes []*Node, err error) {
	return n.findChildren(nil)
}

func (n *Node) CanonKids() (nodes []*Node, err error) {
	return n.findChildren(func(ext *subgraph.SubGraph) (bool, error) {
		return isCanonicalExtension(n.SubGraph, ext)
	})
}
