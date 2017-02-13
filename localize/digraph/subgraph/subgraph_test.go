package subgraph

import "testing"
import "github.com/stretchr/testify/assert"

import ()

import (
	"github.com/timtadh/goiso"
)

import ()

func graph(t *testing.T) (*goiso.Graph, *goiso.SubGraph, *SubGraph, *Indices) {
	Graph := goiso.NewGraph(10, 10)
	G := &Graph
	n1 := G.AddVertex(1, "black")
	n2 := G.AddVertex(2, "black")
	n3 := G.AddVertex(3, "red")
	n4 := G.AddVertex(4, "red")
	n5 := G.AddVertex(5, "red")
	n6 := G.AddVertex(6, "red")
	G.AddEdge(n1, n3, "")
	G.AddEdge(n1, n4, "")
	G.AddEdge(n2, n5, "")
	G.AddEdge(n2, n6, "")
	G.AddEdge(n5, n3, "")
	G.AddEdge(n4, n6, "")
	sg, _ := G.SubGraph([]int{n1.Idx, n2.Idx, n3.Idx, n4.Idx, n5.Idx, n6.Idx}, nil)

	indices := &Indices{
		G:          G,
		ColorIndex: make(map[int][]int),
		SrcIndex:   make(map[IdColorColor][]int),
		TargIndex:  make(map[IdColorColor][]int),
		EdgeIndex:  make(map[Edge]*goiso.Edge),
		EdgeCounts: make(map[Colors]int),
	}

	indices.InitEdgeIndices(G)
	indices.InitColorMap(G)

	return G, sg, FromEmbedding(sg), indices
}

func TestEdgeChain(t *testing.T) {
	_, _, sg, indices := graph(t)
	t.Log(sg)
	chain := sg.edgeChain(indices, nil, 0)
	for _, e := range chain {
		t.Log(e)
	}
	b := BuildEmbedding(len(sg.V), len(sg.E)).Fillable()
	t.Log(b.Builder, b.Ids)
	id := 0
	for _, e := range chain {
		if b.V[e.Src].Idx == -1 {
			b.SetVertex(e.Src, sg.V[e.Src].Color, id)
			id++
		}
		if b.V[e.Targ].Idx == -1 {
			b.SetVertex(e.Targ, sg.V[e.Targ].Color, id)
			id++
		}
		b.AddEdge(&b.V[e.Src], &b.V[e.Targ], e.Color)
		t.Log(b.Builder, b.Ids)
	}
	emb := b.Build()
	t.Log(emb)
	t.Log(emb.SG)
	t.Log(sg)
	if !sg.Equals(emb.SG) {
		t.Fatal("emb != sg")
	}
}

func TestEdgeChain2(t *testing.T) {
	sg := Build(3, 2).Ctx(func(b *Builder) {
		x := b.AddVertex(0)
		y := b.AddVertex(1)
		z := b.AddVertex(1)
		b.AddEdge(x, y, 2)
		b.AddEdge(z, y, 2)
	}).Build()
	t.Log(sg)
	t.Skip("skip test because borked without indices for edgeChain")
	chain := sg.edgeChain(nil, nil, 0)
	for _, e := range chain {
		t.Log(e)
	}
	b := BuildEmbedding(len(sg.V), len(sg.E)).Fillable()
	t.Log(b.Builder, b.Ids)
	id := 0
	for _, e := range chain {
		if b.V[e.Src].Idx == -1 {
			b.SetVertex(e.Src, sg.V[e.Src].Color, id)
			id++
		}
		if b.V[e.Targ].Idx == -1 {
			b.SetVertex(e.Targ, sg.V[e.Targ].Color, id)
			id++
		}
		b.AddEdge(&b.V[e.Src], &b.V[e.Targ], e.Color)
		t.Log(b.Builder, b.Ids)
	}
	emb := b.Build()
	t.Log(emb)
	t.Log(emb.SG)
	t.Log(sg)
	if !sg.Equals(emb.SG) {
		t.Fatal("emb != sg")
	}
}

func TestEmbeddings(t *testing.T) {
	x := assert.New(t)
	t.Logf("%T %v", x, x)
	G, _, sg, indices := graph(t)
	t.Log(sg.Pretty(G.Colors))

	embs, err := sg.Embeddings(indices)
	if err != nil {
		t.Fatal(err)
	}
	for _, emb := range embs {
		t.Log(emb.SG.Pretty(G.Colors))
	}
	for _, emb := range embs {
		t.Log(emb.Pretty(G.Colors))
	}
	x.Equal(len(embs), 2, "embs should have 2 embeddings")
}

func TestEmbeddings2(t *testing.T) {
	G, _, _, indices := graph(t)
	sg := Build(3, 2).Ctx(func(b *Builder) {
		x := b.AddVertex(0)
		y := b.AddVertex(1)
		z := b.AddVertex(1)
		b.AddEdge(x, y, 2)
		b.AddEdge(z, y, 2)
	}).Build()
	t.Log(sg)
	embs, err := sg.Embeddings(indices)
	if err != nil {
		t.Fatal(err)
	}
	for _, emb := range embs {
		t.Log(emb.SG.Pretty(G.Colors))
	}
	for _, emb := range embs {
		t.Log(emb.Pretty(G.Colors))
	}
	if len(embs) != 2 {
		t.Error("embs should have 2 embeddings")
	}
}

func TestNewBuilder(t *testing.T) {
	x := assert.New(t)
	t.Logf("%T %v", x, x)
	_, _, expected, _ := graph(t)
	b := Build(6, 6)
	n1 := b.AddVertex(0)
	n2 := b.AddVertex(0)
	n3 := b.AddVertex(1)
	n4 := b.AddVertex(1)
	n5 := b.AddVertex(1)
	n6 := b.AddVertex(1)
	b.AddEdge(n1, n3, 2)
	b.AddEdge(n1, n4, 2)
	b.AddEdge(n2, n5, 2)
	b.AddEdge(n5, n3, 2)
	b.AddEdge(n4, n6, 2)
	b.AddEdge(n2, n6, 2)
	sg := b.Build()
	t.Log(sg)
	t.Log(sg.Adj)
	t.Log(expected)
	t.Log(expected.Adj)
	x.Equal(sg.String(), expected.String())

	/*
		embs, err := sg.Embeddings(indices, extender)
		if err != nil { t.Fatal(err) }
		for _, emb := range embs {
			t.Log(emb.Label())
		}
		for _, emb := range embs {
			t.Log(emb)
		}
		x.Equal(len(embs), 2, "embs should have 2 embeddings")
	*/
}

func TestFromBuilder(t *testing.T) {
	x := assert.New(t)
	t.Logf("%T %v", x, x)
	_, _, expected, _ := graph(t)
	b := Build(6, 6)
	n1 := b.AddVertex(0)
	n2 := b.AddVertex(0)
	n3 := b.AddVertex(1)
	n4 := b.AddVertex(1)
	n5 := b.AddVertex(1)
	n6 := b.AddVertex(1)
	b.AddEdge(n1, n3, 2)
	b.AddEdge(n1, n4, 2)
	b.AddEdge(n2, n5, 2)
	b.AddEdge(n2, n6, 2)
	b.AddEdge(n4, n6, 2)
	sg1 := b.Build()
	t.Log(sg1)
	t.Log(sg1.Adj)
	b2 := sg1.Builder()
	sg := b2.Copy().Ctx(func(b *Builder) {
		t.Log(b)
		b.AddEdge(&b.V[2], &b.V[3], 2)
	}).Build()
	t.Log(b2.Build())
	t.Log(sg)
	t.Log(expected)
	t.Log(sg.Adj)
	t.Log(expected.Adj)
	x.Equal(sg.String(), expected.String())
}

func TestFromExtension(t *testing.T) {
	x := assert.New(t)
	t.Logf("%T %v", x, x)
	_, _, expected, _ := graph(t)
	b := Build(6, 6)
	n1 := b.AddVertex(0)
	n2 := b.AddVertex(0)
	n3 := b.AddVertex(1)
	n4 := b.AddVertex(1)
	n5 := b.AddVertex(1)
	b.AddEdge(n1, n3, 2)
	b.AddEdge(n1, n4, 2)
	b.AddEdge(n2, n5, 2)
	_, n6, _ := b.Extend(NewExt(*n2, Vertex{Idx: 5, Color: 1}, 2))
	b.Extend(NewExt(*n4, *n6, 2))
	b.Extend(NewExt(*n5, *n3, 2))
	sg := b.Build()
	t.Log(sg)
	t.Log(expected)
	t.Log(sg.Adj)
	t.Log(expected.Adj)
	x.Equal(sg.String(), expected.String())
}

func TestBuilderRemoveEdge(t *testing.T) {
	x := assert.New(t)
	t.Logf("%T %v", x, x)
	_, _, expected, _ := graph(t)
	b := Build(6, 6)
	n1 := b.AddVertex(0)
	n2 := b.AddVertex(0)
	n3 := b.AddVertex(1)
	n4 := b.AddVertex(1)
	n5 := b.AddVertex(1)
	n6 := b.AddVertex(1)
	b.AddEdge(n1, n3, 2)
	b.AddEdge(n1, n4, 2)
	b.AddEdge(n2, n5, 2)
	b.AddEdge(n5, n3, 2)
	b.AddEdge(n4, n6, 2)
	b.AddEdge(n2, n6, 2)
	b.AddEdge(n3, n6, 2)
	wrong := b.Build()
	t.Log(wrong)
	t.Log(expected)
	x.NotEqual(wrong.String(), expected.String())
	b = Build(6, 6).From(wrong)
	err := b.RemoveEdge(6)
	if err != nil {
		t.Fatal(err)
	}
	right := b.Build()
	t.Log(right)
	t.Log(expected)
	x.Equal(right.String(), expected.String())
}

func TestBuilderConnected(t *testing.T) {
	x := assert.New(t)
	b := Build(5, 5)
	n1 := b.AddVertex(0)
	n2 := b.AddVertex(0)
	n3 := b.AddVertex(1)
	n4 := b.AddVertex(1)
	n5 := b.AddVertex(1)
	n6 := b.AddVertex(1)
	b.AddEdge(n1, n3, 2)
	b.AddEdge(n1, n4, 2)
	b.AddEdge(n2, n5, 2)
	b.AddEdge(n5, n3, 2)
	b.AddEdge(n4, n6, 2)
	b.AddEdge(n2, n6, 2)
	x.True(b.Connected())
	_ = b.AddVertex(2)
	x.False(b.Connected())
}
