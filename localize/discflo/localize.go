package discflo

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
)

import (
	"github.com/timtadh/data-structures/errors"
)

import (
	"github.com/timtadh/dynagrok/localize/lattice"
	"github.com/timtadh/dynagrok/localize/lattice/subgraph"
	"github.com/timtadh/dynagrok/localize/test"
	"github.com/timtadh/dynagrok/localize/stat"
)



type SearchNode struct {
	Node  *lattice.Node
	Score float64
	Test  *test.Testcase
}

func (s *SearchNode) String() string {
	return fmt.Sprintf("%v %v", s.Score, s.Node)
}

type Location struct {
	stat.Location
	Graphs   []*SearchNode
}

func (l *Location) String() string {
	graphLine := "--------------------- graph %-2v ----------------------------"
	testLine  := "---------------------- test %-2v ----------------------------"
	parts := make([]string, 0, len(l.Graphs))
	for i, g := range l.Graphs {
		parts = append(parts, fmt.Sprintf(graphLine, i))
		parts = append(parts, g.String())
		parts = append(parts, fmt.Sprintf(testLine, i))
		parts = append(parts, fmt.Sprintf("%v", g.Test))
	}
	graphs := strings.Join(parts, "\n")
	return fmt.Sprintf(
`---------------------- location ---------------------------
Position: %v
Function: %v
BasicBlock: %v
------------------------ score ----------------------------
Score: %v
%v
-----------------------------------------------------------`, l.Position, l.FnName, l.BasicBlockId, l.Score, graphs)
}

func (l *Location) ShortString() string {
	return fmt.Sprintf("%v", &l.Location)
}

type Result []Location

func (r Result) StatResult() stat.Result {
	result := make(stat.Result, 0, len(r))
	for _, l := range r {
		result = append(result, l.Location)
	}
	return result
}

func (r Result) String() string {
	parts := make([]string, 0, len(r))
	for _, l := range r {
		parts = append(parts, l.String())
	}
	return strings.Join(parts, "\n")
}

func (r Result) Sort() {
	sort.SliceStable(r, func(i, j int) bool {
		return r[i].Score > r[j].Score
	})
}

func LocalizeNodes(score Score, lat *lattice.Lattice) stat.Result {
	result := make(stat.Result, 0, len(lat.Fail.ColorIndex))
	for color, embIdxs := range lat.Fail.ColorIndex {
		vsg := subgraph.Build(1, 0).FromVertex(color).Build()
		embs := make([]*subgraph.Embedding, 0, len(embIdxs))
		for _, embIdx := range embIdxs {
			embs = append(embs, subgraph.StartEmbedding(subgraph.VertexEmbedding{SgIdx: 0, EmbIdx: embIdx}))
		}
		n := lattice.NewNode(lat, vsg, embs)
		s := score(lat, n)
		result = append(result, stat.Location{
			lat.Positions[color],
			lat.FnNames[color],
			lat.BBIds[color],
			s,
		})
	}
	result.Sort()
	return result
}

func Localize(walks int, tests []*test.Testcase, oracle *test.Remote, score Score, lat *lattice.Lattice) (Result, error) {
	WALKS := walks
	nodes := make([]*SearchNode, 0, WALKS)
	seen := make(map[string]bool, WALKS)
	for i := 0; i < WALKS; i++ {
		n := Walk(score, lat)
		if n.Node.SubGraph == nil || len(n.Node.SubGraph.E) < 2 {
			continue
		}
		if false {
			errors.Logf("DEBUG", "found %d %v", i, n)
		}
		label := string(n.Node.SubGraph.Label())
		if !seen[label] {
			nodes = append(nodes, n)
			seen[label] = true
		}
	}
	if len(nodes) == 0 {
		fmt.Println("no graphs")
	}
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Score > nodes[j].Score
	})
	filtered := make([]*SearchNode, 0, len(nodes))
	if len(tests) > 0 {
		for i, n := range nodes {
			fmt.Println(n)
			fmt.Printf("------------ ranks %d ----------------\n", i)
			fmt.Println(RankNodes(score, lat, n.Node.SubGraph))
			fmt.Println("--------------------------------------")
			for count := 0; count < len(tests) ; count++ {
				j := rand.Intn(len(tests))
				t := tests[j]
				min, err := t.Minimize(lat, n.Node.SubGraph)
				if err != nil {
					return nil, err
				}
				if min == nil {
					continue
				}
				n.Test = min
				fmt.Printf("------------ min test %d %d ----------\n", i, j)
				fmt.Println(min)
				fmt.Println("--------------------------------------")
				break
			}
			if n.Test == nil {
				// skip this graph
				errors.Logf("INFO", "filtered %d %v", i, n)
			} else if oracle == nil {
				filtered = append(filtered, n)
			} else {
				_, _, _, failures, _, err := n.Test.ExecuteWith(oracle)
				if err != nil {
					return nil, err
				}
				if len(failures) > 0 {
					filtered = append(filtered, n)
				} else {
					errors.Logf("INFO", "filtered %d %v", i, n)
				}
			}
		}
	} else {
		filtered = nodes
	}
	colors := make(map[int][]*SearchNode)
	for _, n := range filtered {
		for j := range n.Node.SubGraph.V {
			colors[n.Node.SubGraph.V[j].Color] = append(colors[n.Node.SubGraph.V[j].Color], n)
		}
	}
	result := RankColors(score, lat, colors)
	return result, nil
}

func RankNodes(score Score, lat *lattice.Lattice, sg *subgraph.SubGraph) stat.Result {
	result := make(stat.Result, 0, len(sg.V))
	for i := range sg.V {
		color := sg.V[i].Color
		vsg := subgraph.Build(1, 0).FromVertex(color).Build()
		embIdxs := lat.Fail.ColorIndex[color]
		embs := make([]*subgraph.Embedding, 0, len(embIdxs))
		for _, embIdx := range embIdxs {
			embs = append(embs, subgraph.StartEmbedding(subgraph.VertexEmbedding{SgIdx: 0, EmbIdx: embIdx}))
		}
		n := lattice.NewNode(lat, vsg, embs)
		s := score(lat, n)
		result = append(result, stat.Location{
			lat.Positions[color],
			lat.FnNames[color],
			lat.BBIds[color],
			s,
		})
	}
	result.Sort()
	return result
}

func RankColors(score Score, lat *lattice.Lattice, colors map[int][]*SearchNode) Result {
	result := make(Result, 0, len(colors))
	for color, searchNodes := range colors {
		vsg := subgraph.Build(1, 0).FromVertex(color).Build()
		embIdxs := lat.Fail.ColorIndex[color]
		embs := make([]*subgraph.Embedding, 0, len(embIdxs))
		for _, embIdx := range embIdxs {
			embs = append(embs, subgraph.StartEmbedding(subgraph.VertexEmbedding{SgIdx: 0, EmbIdx: embIdx}))
		}
		colorNode := lattice.NewNode(lat, vsg, embs)
		colorScore := score(lat, colorNode)
		var s float64
		for _, sn := range searchNodes {
			s += sn.Score
		}
		s = (colorScore * s) / float64(len(searchNodes))
		result = append(result, Location{
			stat.Location{
				lat.Positions[color],
				lat.FnNames[color],
				lat.BBIds[color],
				s,
			},
			searchNodes,
		})
	}
	result.Sort()
	return result
}

func Walk(score Score, lat *lattice.Lattice) (*SearchNode) {
	cur := &SearchNode{
		Node: lat.Root(),
		Score: -100000000,
	}
	i := 0
	prev := cur
	for cur != nil {
		if false {
			errors.Logf("DEBUG", "cur %v", cur)
		}
		kids, err := cur.Node.Children()
		if err != nil {
			panic(err)
		}
		prev = cur
		cur = weighted(filterKids(score, cur.Score, lat, kids))
		if i == 1 {
		}
		i++
	}
	return prev
}

func filterKids(score Score, parentScore float64, lat *lattice.Lattice, kids []*lattice.Node) []*SearchNode {
	entries := make([]*SearchNode, 0, len(kids))
	for _, kid := range kids {
		if kid.FIS() < 2 {
			continue
		}
		kidScore := score(lat, kid)
		if kidScore > parentScore {
			entries = append(entries, &SearchNode{kid, kidScore, nil})
		}
	}
	return entries
}

func uniform(slice []*SearchNode) (*SearchNode) {
	if len(slice) > 0 {
		return slice[rand.Intn(len(slice))]
	}
	return nil
}

func weighted(slice []*SearchNode) (*SearchNode) {
	if len(slice) <= 0 {
		return nil
	}
	if len(slice) == 1 {
		return slice[0]
	}
	prs := transitionPrs(slice)
	if prs == nil {
		return nil
	}
	i := weightedSample(prs)
	return slice[i]
}

func transitionPrs(slice []*SearchNode) []float64 {
	weights := make([]float64, 0, len(slice))
	var total float64 = 0
	for _, v := range slice {
		weights = append(weights, v.Score)
		total += v.Score
	}
	if total == 0 {
		return nil
	}
	prs := make([]float64, 0, len(slice))
	for _, wght := range weights {
		prs = append(prs, wght/total)
	}
	return prs
}

func weightedSample(prs []float64) int {
	var total float64
	for _, pr := range prs {
		total += pr
	}
	i := 0
	x := total * (1 - rand.Float64())
	for x > prs[i] {
		x -= prs[i]
		i += 1
	}
	return i
}
