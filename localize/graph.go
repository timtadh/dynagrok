package localize

import (
	"fmt"
	"io"
	"io/ioutil"
)

import (
	"github.com/timtadh/data-structures/errors"
	"github.com/timtadh/dot"
	"github.com/timtadh/combos"
)

import (
	"github.com/timtadh/sfp/types/digraph/digraph"
)

type VertexAttrs map[int]map[string]interface{}

type DotLoader struct {
	Builder *digraph.Builder
	Labels *digraph.Labels
	Attrs  map[int]map[string]interface{}
	vidxs  map[int]int
}

func LoadDot(labels *digraph.Labels, input io.Reader) (*digraph.Digraph, VertexAttrs, error) {
	l := &DotLoader{
		Builder: digraph.Build(100, 1000),
		Labels: labels,
		Attrs: make(VertexAttrs),
		vidxs: make(map[int]int),
	}
	return l.load(input)
}

func (l *DotLoader) load(input io.Reader) (*digraph.Digraph, VertexAttrs, error) {
	text, err := ioutil.ReadAll(input)
	if err != nil {
		return nil, nil, err
	}
	dp := &dotParse{
		loader: l,
		vids: make(map[string]int),
	}
	err = dot.StreamParse(text, dp)
	if err != nil {
		return nil, nil, err
	}
	return l.Builder.Build(nil, nil), l.Attrs, nil
}

func (l *DotLoader) addVertex(id int, color int, label string, attrs map[string]interface{}) (err error) {
	vertex := l.Builder.AddVertex(color)
	l.vidxs[id] = vertex.Idx
	if l.Attrs != nil && attrs != nil {
		attrs["oid"] = id
		attrs["color"] = color
		l.Attrs[vertex.Idx] = attrs
	}
	return nil
}

func (l *DotLoader) addEdge(sid, tid int, color int, label string) (err error) {
	if sidx, has := l.vidxs[sid]; !has {
		return errors.Errorf("unknown src id %v", tid)
	} else if tidx, has := l.vidxs[tid]; !has{
		return errors.Errorf("unknown targ id %v", tid)
	} else {
		l.Builder.AddEdge(&l.Builder.V[sidx], &l.Builder.V[tidx], color)
	}
	return nil
}


type dotParse struct {
	loader *DotLoader
	graphId int
	curGraph string
	subgraph int
	nextId int
	vids map[string]int
}

func (p *dotParse) Enter(name string, n *combos.Node) error {
	if name == "SubGraph" {
		p.subgraph += 1
		return nil
	}
	p.curGraph = fmt.Sprintf("%v-%d", n.Get(1).Value.(string), p.graphId)
	// errors.Logf("DEBUG", "enter %v %v", p.curGraph, n)
	return nil
}

func (p *dotParse) Stmt(n *combos.Node) error {
	if false {
		errors.Logf("DEBUG", "stmt %v", n)
	}
	if p.subgraph > 0 {
		return nil
	}
	switch n.Label {
	case "Node":
		p.loadVertex(n)
		// errors.Logf("DEBUG", "node %v", n)
	case "Edge":
		p.loadEdge(n)
		// errors.Logf("DEBUG", "edge %v", n)
	}
	return nil
}

func (p *dotParse) Exit(name string) error {
	if name == "SubGraph" {
		p.subgraph--
		return nil
	}
	p.graphId++
	return nil
}

func (p *dotParse) loadVertex(n *combos.Node) (err error) {
	sid := n.Get(0).Value.(string)
	attrs := make(map[string]interface{})
	for _, attr := range n.Get(1).Children {
		name := attr.Get(0).Value.(string)
		value := attr.Get(1).Value.(string)
		attrs[name] = value
	}
	attrs["graphId"] = p.graphId
	id := p.nextId
	p.nextId++
	p.vids[sid] = id
	label := sid
	if l, has := attrs["label"]; has {
		label = l.(string)
	}
	return p.loader.addVertex(id, p.loader.Labels.Color(label), label, attrs)
}

func (p *dotParse) loadEdge(n *combos.Node) (err error) {
	getId := func(sid string) (int, error) {
		if _, has := p.vids[sid]; !has {
			err := p.loadVertex(combos.NewNode("Node").
				AddKid(combos.NewValueNode("ID", sid)).
				AddKid(combos.NewNode("Attrs")))
			if err != nil {
				return 0, err
			}
		}
		return p.vids[sid], nil
	}
	srcSid := n.Get(0).Value.(string)
	sid, err := getId(srcSid)
	if err != nil {
		return err
	}
	targSid := n.Get(1).Value.(string)
	tid, err := getId(targSid)
	if err != nil {
		return err
	}
	label := ""
	for _, attr := range n.Get(2).Children {
		name := attr.Get(0).Value.(string)
		if name == "label" {
			label = attr.Get(1).Value.(string)
			break
		}
	}
	return p.loader.addEdge(sid, tid, p.loader.Labels.Color(label), label)
}

