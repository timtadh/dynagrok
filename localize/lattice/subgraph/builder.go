package subgraph

import (
	"github.com/timtadh/data-structures/errors"
	"github.com/timtadh/goiso/bliss"
)

import (
	"github.com/timtadh/dynagrok/localize/lattice/digraph"
)

type Builder struct {
	V Vertices
	E Edges
}

func Build(V, E int) *Builder {
	return &Builder{
		V: make([]Vertex, 0, V),
		E: make([]Edge, 0, E),
	}
}

// Mutates the current builder and returns it
func (b *Builder) From(sg *SubGraph) *Builder {
	if len(b.V) != 0 || len(b.E) != 0 {
		panic("builder must be empty to use From")
	}
	for i := range sg.V {
		b.AddVertex(sg.V[i].Color)
	}
	for i := range sg.E {
		b.AddEdge(&b.V[sg.E[i].Src], &b.V[sg.E[i].Targ], sg.E[i].Color)
	}
	return b
}

func FromGraph(g *digraph.Digraph) *Builder {
	if g == nil {
		return &Builder{
			V:   make([]Vertex, 0),
			E:   make([]Edge, 0),
		}
	}
	b := &Builder{
		V:   make([]Vertex, len(g.V)),
		E:   make([]Edge, len(g.E)),
	}
	for i := range g.V {
		b.V[i].Idx = i
		b.V[i].Color = g.V[i].Color
	}
	for i := range g.E {
		b.E[i].Src = g.E[i].Src
		b.E[i].Targ = g.E[i].Targ
		b.E[i].Color = g.E[i].Color
	}
	return b
}

func (b *Builder) FromVertex(color int) *Builder {
	b.AddVertex(color)
	return b
}

func (b *Builder) Copy() *Builder {
	V := make([]Vertex, len(b.V), cap(b.V))
	E := make([]Edge, len(b.E), cap(b.E))
	copy(V, b.V)
	copy(E, b.E)
	return &Builder{
		V: V,
		E: E,
	}
}

func (b *Builder) Ctx(do func(*Builder)) *Builder {
	do(b)
	return b
}

func (b *Builder) Do(do func(*Builder) error) (*Builder, error) {
	err := do(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (b *Builder) AddVertex(color int) *Vertex {
	b.V = append(b.V, Vertex{
		Idx:   len(b.V),
		Color: color,
	})
	return &b.V[len(b.V)-1]
}

func (b *Builder) AddEdge(src, targ *Vertex, color int) *Edge {
	b.E = append(b.E, Edge{
		Src:   src.Idx,
		Targ:  targ.Idx,
		Color: color,
	})
	return &b.E[len(b.E)-1]
}

func (b *Builder) RemoveEdge(edgeIdx int) error {
	dropVertex, vertexIdx, err := b.droppedVertexOnEdgeRm(edgeIdx)
	if err != nil {
		return err
	}
	b.V, b.E = b.removeVertexAndEdge(dropVertex, vertexIdx, edgeIdx)
	return nil
}

func (b *Builder) droppedVertexOnEdgeRm(edgeIdx int) (drop bool, idx int, err error) {
	edge := &b.E[edgeIdx]
	rmSrc := true
	rmTarg := true
	for i := range b.E {
		e := &b.E[i]
		if e == edge {
			continue
		}
		if edge.Src == e.Src || edge.Src == e.Targ {
			// a kid edge
			rmSrc = false
		}
		if edge.Targ == e.Src || edge.Targ == e.Targ {
			// a parent edge
			rmTarg = false
		}
	}
	if rmSrc && rmTarg {
		return false, 0, errors.Errorf("would have removed both source and target %v %v", rmSrc, rmTarg)
	}
	rmV := rmSrc || rmTarg
	var rmVidx int
	if rmSrc {
		rmVidx = edge.Src
	}
	if rmTarg {
		rmVidx = edge.Targ
	}
	return rmV, rmVidx, nil
}

func (b *Builder) removeVertexAndEdge(dropVertex bool, vertexIdx, edgeIdx int) (Vertices, Edges) {
	adjustIdx := func(idx int) int {
		if dropVertex && idx > vertexIdx {
			return idx - 1
		}
		return idx
	}
	V := make([]Vertex, 0, len(b.V))
	for idx := range b.V {
		if dropVertex && vertexIdx == idx {
			continue
		}
		V = append(V, Vertex{Idx: adjustIdx(idx), Color: b.V[idx].Color})
	}
	E := make([]Edge, 0, len(b.E)-1)
	for idx := range b.E {
		if idx == edgeIdx {
			continue
		}
		E = append(E, Edge{
			Src:   adjustIdx(b.E[idx].Src),
			Targ:  adjustIdx(b.E[idx].Targ),
			Color: b.E[idx].Color,
		})
	}
	return V, E
}

func (b *Builder) Extend(e *Extension) (newe *Edge, newv *Vertex, err error) {
	if e.Source.Idx > len(b.V) {
		return nil, nil, errors.Errorf("Source.Idx %v outside of |V| %v", e.Source.Idx, len(b.V))
	} else if e.Target.Idx > len(b.V) {
		return nil, nil, errors.Errorf("Target.Idx %v outside of |V| %v", e.Target.Idx, len(b.V))
	} else if e.Source.Idx == len(b.V) && e.Target.Idx == len(b.V) {
		return nil, nil, errors.Errorf("Only one new vertice allowed (Extension would create a disconnnected graph)")
	}
	var src *Vertex = &e.Source
	var targ *Vertex = &e.Target
	if e.Source.Idx == len(b.V) {
		src = b.AddVertex(e.Source.Color)
		newv = src
	} else if e.Target.Idx == len(b.V) {
		targ = b.AddVertex(e.Target.Color)
		newv = targ
	}
	newe = b.AddEdge(src, targ, e.Color)
	return newe, newv, nil
}

func (b *Builder) Kids() [][]*Edge {
	kids := make([][]*Edge, 0, len(b.V))
	for _ = range b.V {
		kids = append(kids, make([]*Edge, 0, 5))
	}
	for i := range b.E {
		e := &b.E[i]
		kids[e.Src] = append(kids[e.Src], e)
	}
	return kids
}

func (b *Builder) Parents() [][]*Edge {
	parents := make([][]*Edge, 0, len(b.V))
	for _ = range b.V {
		parents = append(parents, make([]*Edge, 0, 5))
	}
	for i := range b.E {
		e := &b.E[i]
		parents[e.Targ] = append(parents[e.Targ], e)
	}
	return parents
}

func (b *Builder) Connected() bool {
	kids := b.Kids()
	parents := b.Parents()
	pop := func(stack []int) (int, []int) {
		idx := stack[len(stack)-1]
		stack = stack[0 : len(stack)-1]
		return idx, stack
	}
	visit := func(idx int, stack []int, processed map[int]bool) []int {
		processed[idx] = true
		for _, kid := range kids[idx] {
			if _, has := processed[kid.Targ]; !has {
				stack = append(stack, kid.Targ)
			}
		}
		for _, parent := range parents[idx] {
			if _, has := processed[parent.Src]; !has {
				stack = append(stack, parent.Src)
			}
		}
		return stack
	}
	processed := make(map[int]bool, len(b.V))
	stack := make([]int, 0, len(b.V))
	stack = append(stack, 0)
	for len(stack) > 0 {
		var v int
		v, stack = pop(stack)
		stack = visit(v, stack, processed)
	}
	return len(processed) == len(b.V)
}

func (b *Builder) Build() *SubGraph {
	return b.BuildFromPermutation(b.CanonicalPermutation())
}

func (b *Builder) BuildFromPermutation(vord, eord []int) *SubGraph {
	pat := &SubGraph{
		V:   make([]Vertex, len(b.V)),
		E:   make([]Edge, len(b.E)),
		Adj: make([][]int, len(b.V)),
		InDeg: make([]int, len(b.V)),
		OutDeg: make([]int, len(b.V)),
	}
	for i, j := range vord {
		pat.V[j].Idx = j
		pat.V[j].Color = b.V[i].Color
		pat.Adj[j] = make([]int, 0, 5)
	}
	for i, j := range eord {
		pat.E[j].Src = vord[b.E[i].Src]
		pat.E[j].Targ = vord[b.E[i].Targ]
		pat.E[j].Color = b.E[i].Color
		pat.Adj[pat.E[j].Src] = append(pat.Adj[pat.E[j].Src], j)
		pat.Adj[pat.E[j].Targ] = append(pat.Adj[pat.E[j].Targ], j)
		pat.OutDeg[pat.E[j].Src]++
		pat.InDeg[pat.E[j].Targ]++
	}
	return pat
}

func (b *Builder) CanonicalPermutation() (vord, eord []int) {
	bMap := bliss.NewMap(len(b.V), len(b.E), b.V.Iterate(), b.E.Iterate())
	vord, eord, _ = bMap.CanonicalPermutation()
	return vord, eord
}
