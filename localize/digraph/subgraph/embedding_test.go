package subgraph

import "testing"

import (
	"bytes"
)

func embedding(t *testing.T) *Embedding {
	b := BuildEmbedding(6, 6)
	n1 := b.AddVertex(0, 1)
	n2 := b.AddVertex(0, 2)
	n3 := b.AddVertex(1, 3)
	n4 := b.AddVertex(1, 4)
	n5 := b.AddVertex(1, 5)
	n6 := b.AddVertex(1, 6)
	b.AddEdge(n1, n3, 2)
	b.AddEdge(n1, n4, 2)
	b.AddEdge(n2, n5, 2)
	b.AddEdge(n2, n6, 2)
	b.AddEdge(n5, n3, 2)
	b.AddEdge(n4, n6, 2)
	emb := b.Build()
	return emb
}

func TestBuildEmb(t *testing.T) {
	emb := embedding(t)
	_, _, sg, _ := graph(t)
	t.Log(emb)
	t.Log(emb.SG)
	t.Log(sg)
	if !bytes.Equal(emb.Label(), sg.Label()) {
		t.Errorf("\n emb %v !=\n  sg %v", emb.SG, sg)
	}
}

func TestSerializeEmb(t *testing.T) {
	emb1 := embedding(t)
	emb2 := embedding(t)
	_, _, sg, _ := graph(t)
	if !bytes.Equal(emb1.Label(), sg.Label()) {
		t.Errorf("\n emb1 %v !=\n   sg %v", emb1.SG, sg)
	}
	if !bytes.Equal(emb2.Label(), sg.Label()) {
		t.Errorf("\n emb2 %v !=\n   sg %v", emb2.SG, sg)
	}
	emb3, err := LoadEmbedding(emb1.Serialize())
	if err != nil {
		t.Fatal(err)
	}
	if emb3 == emb1 {
		t.Errorf("emb3 and emb1 were the same pointer")
	}
	if !bytes.Equal(emb3.Label(), sg.Label()) {
		t.Errorf("\n emb3 %v !=\n   sg %v", emb3.SG, sg)
	}
	if !emb2.Equals(emb3) {
		t.Errorf("emb2 != emb3")
	}
}

func TestEmbBuildRmEdge(t *testing.T) {
	emb1 := embedding(t)
	emb2 := BuildEmbedding(6, 5).Ctx(func(b *EmbeddingBuilder) {
		n1 := b.AddVertex(0, 1)
		n2 := b.AddVertex(0, 2)
		n3 := b.AddVertex(1, 3)
		n4 := b.AddVertex(1, 4)
		n5 := b.AddVertex(1, 5)
		n6 := b.AddVertex(1, 6)
		b.AddEdge(n1, n3, 2)
		b.AddEdge(n1, n4, 2)
		b.AddEdge(n2, n5, 2)
		b.AddEdge(n5, n3, 2)
		b.AddEdge(n4, n6, 2)
	}).Build()
	emb3 := BuildEmbedding(5, 4).Ctx(func(b *EmbeddingBuilder) {
		n1 := b.AddVertex(0, 1)
		n2 := b.AddVertex(0, 2)
		n3 := b.AddVertex(1, 3)
		n4 := b.AddVertex(1, 4)
		n5 := b.AddVertex(1, 5)
		b.AddEdge(n1, n3, 2)
		b.AddEdge(n1, n4, 2)
		b.AddEdge(n2, n5, 2)
		b.AddEdge(n5, n3, 2)
	}).Build()
	t.Log(emb1)
	t.Log(emb2)
	t.Log(emb3)
	b2, err := emb1.Builder().Do(func(b *EmbeddingBuilder) error {
		return b.RemoveEdge(4)
	})
	if err != nil {
		t.Fatal(err)
	}
	b3, err := b2.Copy().Do(func(b *EmbeddingBuilder) error {
		return b.RemoveEdge(0)
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log(b2.Build())
	t.Log(b3.Build())
	if !b2.Build().Equals(emb2) {
		t.Errorf("emb2 != b2 \n   b2 %v %v\n emb2 %v", b2.Builder, b2.Ids, emb2)
	}
	if !b3.Build().Equals(emb3) {
		t.Errorf("emb3 != b3 \n   b3 %v %v\n emb3 %v", b3.Builder, b2.Ids, emb3)
	}
}
