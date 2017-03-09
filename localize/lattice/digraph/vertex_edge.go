package digraph

import (
	"github.com/timtadh/goiso/bliss"
)

type Vertex struct {
	Idx, Color int
}

type Edge struct {
	Src, Targ, Color int
}

type Vertices []Vertex
type Edges []Edge

func (V Vertices) Iterate() (vi bliss.VertexIterator) {
	i := 0
	vi = func() (color int, _ bliss.VertexIterator) {
		if i >= len(V) {
			return 0, nil
		}
		color = V[i].Color
		i++
		return color, vi
	}
	return vi
}

func (E Edges) Iterate() (ei bliss.EdgeIterator) {
	i := 0
	ei = func() (src, targ, color int, _ bliss.EdgeIterator) {
		if i >= len(E) {
			return 0, 0, 0, nil
		}
		src = E[i].Src
		targ = E[i].Targ
		color = E[i].Color
		i++
		return src, targ, color, ei
	}
	return ei
}
