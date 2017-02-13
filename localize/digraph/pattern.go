package digraph

import (
	"github.com/timtadh/data-structures/types"
)

import (
	"github.com/timtadh/sfp/lattice"
	"github.com/timtadh/dynagrok/localize/digraph/subgraph"
)

type Pattern struct {
	subgraph.SubGraph
}

func (p *Pattern) Level() int {
	return len(p.SubGraph.E) + 1
}

func (p *Pattern) Distance(x lattice.Pattern) float64 {
	o := x.(*Pattern)
	return p.Metric(&o.SubGraph)
}

type Labeled interface {
	Label() []byte
}

func (p *Pattern) Equals(o types.Equatable) bool {
	a := types.ByteSlice(p.Label())
	switch b := o.(type) {
	case Labeled:
		return a.Equals(types.ByteSlice(b.Label()))
	default:
		return false
	}
}

func (p *Pattern) Less(o types.Sortable) bool {
	a := types.ByteSlice(p.Label())
	switch b := o.(type) {
	case Labeled:
		return a.Less(types.ByteSlice(b.Label()))
	default:
		return false
	}
}

func (p *Pattern) Hash() int {
	return types.ByteSlice(p.Label()).Hash()
}
