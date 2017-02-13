package digraph

import (
	"fmt"
)

import (
	"github.com/timtadh/data-structures/errors"
)

import (
	"github.com/timtadh/sfp/lattice"
	"github.com/timtadh/dynagrok/localize/digraph/subgraph"
)

type Node struct {
	dt *Digraph
	SubGraph *subgraph.SubGraph
	Embeddings subgraph.Embeddings
	unsupportedExts map[subgraph.Extension]bool
	kids []lattice.Node
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

func (n *Node) AsNode() lattice.Node {
	return n
}

func (n *Node) Pattern() lattice.Pattern {
	if n.SubGraph == nil {
		return &Pattern{}
	}
	return &Pattern{*n.SubGraph}
}

func (n *Node) String() string {
	if n.SubGraph == nil {
		return "<Node {0:0}>"
	}
	return fmt.Sprintf("<Node %v>", n.SubGraph.Pretty(n.dt.Labels))
}

func (n *Node) Parents() ([]lattice.Node, error) {
	return nil, errors.Errorf("not supported yet")
}

func (n *Node) Children() (nodes []lattice.Node, err error) {
	n.kids, err = n.findChildren(nil)
	return n.kids, err
}

func (n *Node) CanonKids() (nodes []lattice.Node, err error) {
	return n.findChildren(func(ext *subgraph.SubGraph) (bool, error) {
		return isCanonicalExtension(n.SubGraph, ext)
	})
}

func (n *Node) AdjacentCount() (int, error) {
	pc, err := n.ParentCount()
	if err != nil {
		return 0, err
	}
	cc, err := n.ChildCount()
	if err != nil {
		return 0, err
	}
	return pc + cc, nil
}

func (n *Node) ParentCount() (int, error) {
	return 0, errors.Errorf("not supported yet")
}

func (n *Node) ChildCount() (int, error) {
	if n.kids == nil {
		_, err := n.Children()
		if err != nil {
			return 0, err
		}
	}
	return len(n.kids), nil
}

func (n *Node) Maximal() (bool, error) {
	cc, err := n.ChildCount()
	if err != nil {
		return false, err
	}
	return cc == 0, nil
}

func (n *Node) Lattice() (*lattice.Lattice, error) {
	return nil, &lattice.NoLattice{}
}
