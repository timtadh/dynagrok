package subgraph

import ()

import (
	"github.com/timtadh/dynagrok/localize/digraph/digraph"
)


func (sg *SubGraph) AsIndices() *digraph.Indices {
	b := digraph.Build(len(sg.V),len(sg.E))
	for vidx := range sg.V {
		u := &sg.V[vidx]
		b.AddVertex(u.Color)
	}
	for eidx := range sg.E {
		src := &b.V[sg.E[eidx].Src]
		targ := &b.V[sg.E[eidx].Targ]
		color := sg.E[eidx].Color
		b.AddEdge(src, targ, color)
	}
	return digraph.NewIndices(b, 1)
}
