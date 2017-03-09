package subgraph

import (
	"github.com/timtadh/data-structures/types"
)

func (sg *SubGraph) Equals(o types.Equatable) bool {
	switch b := o.(type) {
	case *SubGraph:
		return sg.equals(b)
	default:
		return false
	}
}

func (a *SubGraph) equals(b *SubGraph) bool {
	if len(a.V) != len(b.V) {
		return false
	}
	if len(a.E) != len(b.E) {
		return false
	}
	for i := range a.V {
		if a.V[i].Color != b.V[i].Color {
			return false
		}
	}
	for i := range a.E {
		if a.E[i].Src != b.E[i].Src {
			return false
		}
		if a.E[i].Targ != b.E[i].Targ {
			return false
		}
		if a.E[i].Color != b.E[i].Color {
			return false
		}
	}
	return true
}

func (sg *SubGraph) Less(o types.Sortable) bool {
	a := types.ByteSlice(sg.Label())
	switch b := o.(type) {
	case *SubGraph:
		return a.Less(types.ByteSlice(b.Label()))
	default:
		return false
	}
}

func (sg *SubGraph) Hash() int {
	return types.ByteSlice(sg.Label()).Hash()
}

