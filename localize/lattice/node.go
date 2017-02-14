package lattice

import (
	"fmt"
)

import ()

import (
	"github.com/timtadh/dynagrok/localize/lattice/subgraph"
)

type Node struct {
	dt *Digraph
	SubGraph *subgraph.SubGraph
	Embeddings subgraph.Embeddings
	unsupportedExts map[subgraph.Extension]bool
}

func NewNode(dt *Digraph, sg *subgraph.SubGraph, embs subgraph.Embeddings) *Node {
	return &Node{
		dt: dt,
		SubGraph: sg,
		Embeddings: embs,
		unsupportedExts: make(map[subgraph.Extension]bool),
	}
}

func (n *Node) addUnsupportedExts(unsup map[subgraph.Extension]bool, V int, vord []int) {
	for u := range unsup {
		n.unsupportedExts[*u.Translate(V, vord)] = true
	}
}

func (n *Node) String() string {
	if n.SubGraph == nil {
		return "<Node {0:0}>"
	}
	return fmt.Sprintf("<Node %v>", n.SubGraph.Pretty(n.dt.Labels))
}

func (n *Node) Children() (nodes []*Node, err error) {
	return n.findChildren(nil)
}

func (n *Node) CanonKids() (nodes []*Node, err error) {
	return n.findChildren(func(ext *subgraph.SubGraph) (bool, error) {
		return isCanonicalExtension(n.SubGraph, ext)
	})
}
