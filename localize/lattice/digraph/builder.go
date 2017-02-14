package digraph

import (
	"github.com/timtadh/data-structures/errors"
)

type Builder struct {
	V            Vertices
	E            Edges
	Adj          [][]int
	VertexColors map[int]int
	EdgeColors   map[int]int
	Graphs       int
}

func Build(V, E int) *Builder {
	if V < 10 {
		V = 100
	}
	if E < 10 {
		E = 100
	}
	return &Builder{
		V: make(Vertices, 0, V),
		E: make(Edges, 0, E),
		Adj: make([][]int, 0, V),
		VertexColors: make(map[int]int, V),
		EdgeColors: make(map[int]int, E),
	}
}

func (b *Builder) Build(indexVertex func(*Vertex), indexEdge func(*Edge)) *Digraph {
	g := &Digraph{
		V: make(Vertices, len(b.V)),
		E: make(Edges, len(b.E)),
		Adj: make([][]int, len(b.V)),
		Kids: make([][]int, len(b.V)),
		Parents: make([][]int, len(b.V)),
		Graphs: b.Graphs,
	}
	for i := range b.V {
		g.V[i].Idx = b.V[i].Idx
		g.V[i].Color = b.V[i].Color
		g.Adj[i] = make([]int, len(b.Adj[i]))
		kids := 0
		parents := 0
		for j, e := range b.Adj[i] {
			g.Adj[i][j] = e
			if b.E[e].Src == i {
				kids++
			} else if b.E[e].Targ == i {
				parents++
			} else {
				panic("edge on neither source or target")
			}
		}
		g.Kids[i] = make([]int, 0, kids)
		g.Parents[i] = make([]int, 0, parents)
		for _, e := range b.Adj[i] {
			if b.E[e].Src == i {
				g.Kids[i] = append(g.Kids[i], e)
			} else if b.E[e].Targ == i {
				g.Parents[i] = append(g.Parents[i], e)
			} else {
				panic("edge on neither source or target")
			}
		}
		if indexVertex != nil {
			indexVertex(&g.V[i])
		}
	}
	errors.Logf("DEBUG", "Built vertex indices about to build edge indices")
	for i := range b.E {
		g.E[i].Src = b.E[i].Src
		g.E[i].Targ = b.E[i].Targ
		g.E[i].Color = b.E[i].Color
		if indexEdge != nil {
			indexEdge(&g.E[i])
		}
	}
	return g
}

func (b *Builder) AddVertex(color int) *Vertex {
	if b == nil {
		panic("b was nil")
	}
	idx := len(b.V)
	if idx < cap(b.V) && idx < cap(b.Adj) {
		b.V = b.V[:idx+1]
		b.V[idx].Idx = idx
		b.V[idx].Color = color
		b.Adj = b.Adj[:idx+1]
		b.Adj[idx] = make([]int, 0, 5)
	} else {
		b.V = append(b.V, Vertex{
			Idx: idx,
			Color: color,
		})
		b.Adj = append(b.Adj, make([]int, 0, 5))
	}
	b.VertexColors[color]++
	return &b.V[idx]
}

func (b *Builder) AddEdge(u, v *Vertex, color int) *Edge {
	idx := len(b.E)
	if idx < cap(b.E) {
		b.E = b.E[:idx+1]
		b.E[idx].Src = u.Idx
		b.E[idx].Targ = v.Idx
		b.E[idx].Color = color
	} else {
		b.E = append(b.E, Edge{
			Src: u.Idx,
			Targ: v.Idx,
			Color: color,
		})
	}
	e := &b.E[idx]
	b.Adj[e.Src] = append(b.Adj[e.Src], idx)
	b.Adj[e.Targ] = append(b.Adj[e.Targ], idx)
	b.EdgeColors[color]++
	return e
}
