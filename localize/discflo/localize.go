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
	seen := make(map[string]*SearchNode, WALKS)
	db := NewDbScan(.25)
	for i := 0; i < WALKS; i++ {
		n := Walk(score, lat)
		if n.Node.SubGraph == nil || len(n.Node.SubGraph.E) < 1 {
			continue
		}
		if false {
			errors.Logf("DEBUG", "found %d %v", i, n)
		}
		label := string(n.Node.SubGraph.Label())
		if _, has := seen[label]; !has {
			db.Add(n.Node)
			nodes = append(nodes, n)
			seen[label] = n
		}
	}
	if len(nodes) == 0 {
		fmt.Println("no graphs")
	}
	// clusters := db.Clusters()
	// reps := make([]*SearchNode, 0, len(clusters))
	// fmt.Printf("+ Clusters %v of %v graphs\n", len(clusters), len(nodes))
	// for i, cluster := range clusters {
	// 	j := 0
	// 	var rep *SearchNode
	// 	for _, item := range cluster {
	// 		sn := seen[string(item.Label())]
	// 		if j == 0 {
	// 			fmt.Println("  ", "-", i, sn)
	// 		} else {
	// 			fmt.Println("      ", "o", j, sn)
	// 		}
	// 		if rep == nil || sn.Score > rep.Score {
	// 			rep = sn
	// 		}
	// 		j++
	// 	}
	// 	reps = append(reps, rep)
	// }
	// nodes = reps
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Score > nodes[j].Score
	})
	passing := make([]*SearchNode, 0, len(nodes))
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
					passing = append(passing, n)
				}
			}
		}
	} else {
		filtered = nodes
	}
	colors := make(map[int][]*SearchNode)
	for i := 0; i < 100 && i < len(filtered); i++ {
		n := filtered[i]
	// for _, n := range filtered {
		errors.Logf("DEBUG", "%v", n)
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
	epsilon := .1
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
		t := 0
		for _, sn := range searchNodes {
			rm := s/float64(t)
			if t < 1 || abs(sn.Score - rm) < epsilon {
				s += sn.Score
				t++
			} else {
				errors.Logf("DEBUG", "skipped %v %v %v", lat.Labels.Label(color), rm, sn.Score)
			}
		}
		s = (colorScore * s) / float64(t)
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
	// color := lat.Labels.Color("(*dynagrok/examples/avl.Avl).Verify blk 3")
	// vsg := subgraph.Build(1, 0).FromVertex(color).Build()
	// embIdxs := lat.Fail.ColorIndex[color]
	// embs := make([]*subgraph.Embedding, 0, len(embIdxs))
	// for _, embIdx := range embIdxs {
	// 	embs = append(embs, subgraph.StartEmbedding(subgraph.VertexEmbedding{SgIdx: 0, EmbIdx: embIdx}))
	// }
	// colorNode := lattice.NewNode(lat, vsg, embs)
	// cur := &SearchNode{
	// 	Node: colorNode,
	// 	Score: score(lat, colorNode),
	// }
	cur := &SearchNode{
		Node: lat.Root(),
		Score: 0,
	}
	i := 0
	prev := cur
	for cur != nil {
		if true {
			errors.Logf("DEBUG", "cur %v", cur)
		}
		kids, err := cur.Node.Children()
		if err != nil {
			panic(err)
		}
		prev = cur
		cur = weighted(filterKids(score, cur.Score, lat, kids))
		if i == 25 {
			break
		}
		i++
	}
	return prev
}

func abs(a float64) float64 {
	if a < 0 {
		return -a
	}
	return a
}

func filterKids(score Score, parentScore float64, lat *lattice.Lattice, kids []*lattice.Node) (float64, []*SearchNode) {
	var epsilon float64 = 0
	entries := make([]*SearchNode, 0, len(kids))
	for _, kid := range kids {
		if kid.FIS() < 2 {
			continue
		}
		kidScore := score(lat, kid)
		_, _, prf, pro := Prs(lat, kid)
		if (abs(parentScore - kidScore) <= epsilon && abs(1 - prf/(pro + prf)) <= epsilon) || kidScore > parentScore {
			entries = append(entries, &SearchNode{kid, kidScore, nil})
		}
	}
	return parentScore, entries
}

func uniform(parentScore float64, slice []*SearchNode) (*SearchNode) {
	if len(slice) > 0 {
		return slice[rand.Intn(len(slice))]
	}
	return nil
}

func weighted(parentScore float64, slice []*SearchNode) (*SearchNode) {
	if len(slice) <= 0 {
		return nil
	}
	if len(slice) == 1 {
		return slice[0]
	}
	i := weightedSample(weights(parentScore, slice))
	return slice[i]
}

func weights(parentScore float64, slice []*SearchNode) []float64 {
	weights := make([]float64, 0, len(slice))
	// var total float64 = 0
	// for _, v := range slice {
	// 	total += v.Score
	// }
	// mean := total/float64(len(slice))
	// var vari float64
	// for _, v := range slice {
	// 	vari += math.Pow(v.Score - mean, 2)
	// }
	// stdev := math.Sqrt(vari)
	for _, v := range slice {
		// w := math.Pow(v.Score - parentScore, 2)
		// w := (v.Score + stdev)/mean
		w := v.Score
		weights = append(weights, w)
	}
	return weights
}

func weightedSample(weights []float64) int {
	var total float64
	for _, w := range weights {
		total += w
	}
	i := 0
	r := total * rand.Float64()
	for ; i < len(weights) && r > weights[i]; i++ {
		r -= weights[i]
	}
	return i
}
