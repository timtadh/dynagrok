package subgraph

import (
	"encoding/binary"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

import (
	"github.com/timtadh/data-structures/errors"
	"github.com/timtadh/goiso/bliss"
)

import (
	"github.com/timtadh/dynagrok/localize/digraph/digraph"
)

type Labels interface {
	Label(int) string
}

type SubGraph struct {
	V          Vertices
	E          Edges
	Adj        [][]int
	InDeg      []int
	OutDeg     []int
	labelCache []byte
}

type Vertices []Vertex
type Edges []Edge

type Vertex struct {
	Idx   int
	Color int
}

type Edge struct {
	Src, Targ, Color int
}

func EmptySubGraph() *SubGraph {
	return &SubGraph{
		V:   make(Vertices, 0),
		E:   make(Edges, 0),
		Adj: make([][]int, 0),
	}
}

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

func (sg *SubGraph) EmbeddingExists(emb *Embedding, G *digraph.Digraph) bool {
	seen := make(map[int]bool, len(sg.V))
	ids := make([]int, len(sg.V))
	for e := emb; e != nil; e = e.Prev {
		if seen[e.EmbIdx] {
			return false
		}
		seen[e.EmbIdx] = true
		ids[e.SgIdx] = e.EmbIdx
		if G.V[e.EmbIdx].Color != sg.V[e.SgIdx].Color {
			return false
		}
	}
	for i := range sg.E {
		e := &sg.E[i]
		found := false
		for _, x := range G.Kids[ids[e.Src]] {
			ke := &G.E[x]
			if ke.Color != e.Color {
				continue
			}
			if G.V[ke.Src].Color != sg.V[e.Src].Color {
				continue
			}
			if G.V[ke.Targ].Color != sg.V[e.Targ].Color {
				continue
			}
			if ke.Src != ids[e.Src] {
				continue
			}
			if ke.Targ != ids[e.Targ] {
				continue
			}
			found = true
			break
		}
		if !found {
			return false
		}
	}
	return true
}

func LoadSubGraph(label []byte) (*SubGraph, error) {
	sg := new(SubGraph)
	err := sg.UnmarshalBinary(label)
	if err != nil {
		return nil, err
	}
	return sg, nil
}

func ParsePretty(str string, labels *digraph.Labels) (*SubGraph, error) {
	size := regexp.MustCompile(`^\{([0-9]+):([0-9]+)\}`)
	edge := regexp.MustCompile(`^\[([0-9]+)->([0-9]+):`)
	matches := size.FindStringSubmatch(str)
	E, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return nil, err
	}
	V, err := strconv.ParseInt(matches[2], 10, 64)
	if err != nil {
		return nil, err
	}
	idx := len(matches[0])
	vertices := make(Vertices, V)
	for i := range vertices {
		if str[idx] != '(' {
			return nil, errors.Errorf("expected ( got %v", str[idx:idx+1])
		}
		idx++
		parens := 1
		labelc := make([]byte, 0, 10)
		for ; idx < len(str); idx++ {
			if str[idx] == '\\' {
				continue
			} else if str[idx-1] == '\\' {
				labelc = append(labelc, str[idx])
			} else if str[idx] == '(' {
				parens += 1
				labelc = append(labelc, str[idx])
			} else if str[idx] == ')' {
				parens -= 1
				if parens > 0 {
					labelc = append(labelc, str[idx])
				} else {
					idx++
					break
				}
			} else {
				labelc = append(labelc, str[idx])
			}
		}
		label := string(labelc)
		vertices[i].Idx = i
		vertices[i].Color = labels.Color(label)
	}
	edges := make(Edges, E)
	for i := range edges {
		matches := edge.FindStringSubmatch(str[idx:])
		src, err := strconv.ParseInt(matches[1], 10, 64)
		if err != nil {
			return nil, err
		}
		targ, err := strconv.ParseInt(matches[2], 10, 64)
		if err != nil {
			return nil, err
		}
		labelc := make([]byte, 0, 10)
		idx += len(matches[0])
		sqBrackets := 1
		for ; idx < len(str); idx++ {
			if str[idx] == '\\' {
				continue
			} else if str[idx-1] == '\\' {
				labelc = append(labelc, str[idx])
				continue
			} else if str[idx] == '[' {
				sqBrackets++
			} else if str[idx] == ']' {
				sqBrackets--
				if sqBrackets <= 0 {
					idx++
					break
				}
			}
			labelc = append(labelc, str[idx])
		}
		label := string(labelc)
		edges[i].Src = int(src)
		edges[i].Targ = int(targ)
		edges[i].Color = labels.Color(label)
	}
	sg := &Builder{V: vertices, E: edges}
	return sg.Build(), nil
}

func (sg *SubGraph) Builder() *Builder {
	if sg == nil {
		return Build(1, 2)
	}
	return Build(len(sg.V)+1, len(sg.E)+1).From(sg)
}

func (sg *SubGraph) Serialize() []byte {
	return sg.Label()
}

func (sg *SubGraph) MarshalBinary() ([]byte, error) {
	return sg.Label(), nil
}

func (sg *SubGraph) UnmarshalBinary(bytes []byte) error {
	if sg.V != nil || sg.E != nil || sg.Adj != nil {
		return errors.Errorf("sg is already loaded! will not load serialized data")
	}
	if len(bytes) < 8 {
		return errors.Errorf("bytes was too small %v < 8", len(bytes))
	}
	lenE := int(binary.BigEndian.Uint32(bytes[0:4]))
	lenV := int(binary.BigEndian.Uint32(bytes[4:8]))
	off := 8
	expected := 8 + lenV*4 + lenE*12
	if len(bytes) < expected {
		return errors.Errorf("bytes was too small %v < %v", len(bytes), expected)
	}
	sg.V = make([]Vertex, lenV)
	sg.E = make([]Edge, lenE)
	sg.Adj = make([][]int, lenV)
	for i := 0; i < lenV; i++ {
		s := off + i*4
		e := s + 4
		color := int(binary.BigEndian.Uint32(bytes[s:e]))
		sg.V[i].Idx = i
		sg.V[i].Color = color
		sg.Adj[i] = make([]int, 0, 5)
	}
	off += lenV * 4
	for i := 0; i < lenE; i++ {
		s := off + i*12
		e := s + 4
		src := int(binary.BigEndian.Uint32(bytes[s:e]))
		s += 4
		e += 4
		targ := int(binary.BigEndian.Uint32(bytes[s:e]))
		s += 4
		e += 4
		color := int(binary.BigEndian.Uint32(bytes[s:e]))
		sg.E[i].Src = src
		sg.E[i].Targ = targ
		sg.E[i].Color = color
		sg.Adj[src] = append(sg.Adj[src], i)
		sg.Adj[targ] = append(sg.Adj[targ], i)
	}
	sg.labelCache = bytes
	return nil
}

func (sg *SubGraph) Label() []byte {
	if sg.labelCache != nil {
		return sg.labelCache
	}
	size := 8 + len(sg.V)*4 + len(sg.E)*12
	label := make([]byte, size)
	binary.BigEndian.PutUint32(label[0:4], uint32(len(sg.E)))
	binary.BigEndian.PutUint32(label[4:8], uint32(len(sg.V)))
	off := 8
	for i, v := range sg.V {
		s := off + i*4
		e := s + 4
		binary.BigEndian.PutUint32(label[s:e], uint32(v.Color))
	}
	off += len(sg.V) * 4
	for i, edge := range sg.E {
		s := off + i*12
		e := s + 4
		binary.BigEndian.PutUint32(label[s:e], uint32(edge.Src))
		s += 4
		e += 4
		binary.BigEndian.PutUint32(label[s:e], uint32(edge.Targ))
		s += 4
		e += 4
		binary.BigEndian.PutUint32(label[s:e], uint32(edge.Color))
	}
	sg.labelCache = label
	return label
}

func (sg *SubGraph) String() string {
	V := make([]string, 0, len(sg.V))
	E := make([]string, 0, len(sg.E))
	for _, v := range sg.V {
		V = append(V, fmt.Sprintf(
			"(%v)",
			v.Color,
		))
	}
	for _, e := range sg.E {
		E = append(E, fmt.Sprintf(
			"[%v->%v:%v]",
			e.Src,
			e.Targ,
			e.Color,
		))
	}
	return fmt.Sprintf("{%v:%v}%v%v", len(sg.E), len(sg.V), strings.Join(V, ""), strings.Join(E, ""))
}

func (sg *SubGraph) Pretty(labels *digraph.Labels) string {
	V := make([]string, 0, len(sg.V))
	E := make([]string, 0, len(sg.E))
	for _, v := range sg.V {
		V = append(V, fmt.Sprintf(
			"(%v)",
			labels.Label(v.Color),
		))
	}
	for _, e := range sg.E {
		E = append(E, fmt.Sprintf(
			"[%v->%v:%v]",
			e.Src,
			e.Targ,
			labels.Label(e.Color),
		))
	}
	return fmt.Sprintf("{%v:%v}%v%v", len(sg.E), len(sg.V), strings.Join(V, ""), strings.Join(E, ""))
}

func (sg *SubGraph) Dotty(labels *digraph.Labels, highlightVertices, highlightEdges map[int]bool) string {
	V := make([]string, 0, len(sg.V))
	E := make([]string, 0, len(sg.E))
	for vidx, v := range sg.V {
		highlight := ""
		if highlightVertices[vidx] {
			highlight = " color=red"
		}
		V = append(V, fmt.Sprintf(
			"n%v [label=\"%v\"%v];",
			vidx,
			labels.Label(v.Color),
			highlight,
		))
	}
	for eidx, e := range sg.E {
		highlight := ""
		if highlightEdges[eidx] {
			highlight = " color=red"
		}
		E = append(E, fmt.Sprintf(
			"n%v->n%v [label=\"%v\"%v]",
			e.Src,
			e.Targ,
			labels.Label(e.Color),
			highlight,
		))
	}
	return fmt.Sprintf("digraph{\n%v\n%v\n}", strings.Join(V, "\n"), strings.Join(E, "\n"))
}
