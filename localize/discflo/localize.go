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
	Clusters   []*Cluster
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
			color,
			lat.Info.Positions[color],
			lat.Info.FnNames[color],
			lat.Info.BBIds[color],
			s,
		})
	}
	result.Sort()
	return result
}


func Localize(walks int, tests []*test.Testcase, oracle test.Executor, score Score, lat *lattice.Lattice) (Clusters, error) {
	min := func(a, b int) int {
		if a < b {
			return a
		}
		return b
	}
	max := func(a, b int) int {
		if a > b {
			return a
		}
		return b
	}
	WALKS := walks
	nodes := make([]*SearchNode, 0, WALKS)
	seen := make(map[string]*SearchNode, WALKS)
	db := NewDbScan(.35)
	// for i := 0; i < WALKS; i++ {
	// 	n := Walk(score, lat)
	total := min(len(lat.Labels.Labels()), max(50, min(len(lat.Labels.Labels())/32, 500)))
	// total = len(lat.Labels.Labels())
	prevScore := 0.0
	groups := 0
	for i, l := range LocalizeNodes(score, lat) {
	// for color := range lat.Labels.Labels() {
		color := l.Color
		if i >= total && groups > 1 {
			break
		}
		for w := 0; w < walks; w++ {
			n := WalkFromColor(color, score, lat)
			if n.Node.SubGraph == nil { // || len(n.Node.SubGraph.E) < 1 {
				continue
			}
			label := string(n.Node.SubGraph.Label())
			if _, has := seen[label]; !has {
				db.Add(n)
				nodes = append(nodes, n)
				seen[label] = n
				if true {
					errors.Logf("DEBUG", "found %d %d/%d %d %v", groups, i, total, len(nodes), n)
				}
			} else {
				if false {
					errors.Logf("DEBUG", "repeat %v", len(nodes), n)
				}
			}
		}
		if prevScore - l.Score  > .0001 {
			groups++
		}
		prevScore = l.Score
	}
	if false {
		errors.Logf("DEBUG", "groups %v", groups)
	}
	if len(nodes) == 0 {
		fmt.Println("no graphs")
	}
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Score > nodes[j].Score
	})
	clusters := db.Clusters()
	sort.Slice(clusters, func(i, j int) bool {
		return clusters[i].Score > clusters[j].Score
	})
	for _, c := range clusters {
		sort.Slice(c.Nodes, func(i, j int) bool {
			return c.Nodes[i].Score > c.Nodes[j].Score
		})
	}
	passing := make([]*SearchNode, 0, len(nodes))
	filtered := make([]*Cluster, 0, len(nodes))
	if len(tests) > 0 {
		for i, c := range clusters {
			if len(filtered) >= 5 && i > 5 || len(filtered) >= 2 && i > 10 || len(filtered) >= 1 && i > 15 {
				break
			}
			fmt.Printf("------------ cluster %d --------------\n", i)
			fmt.Println(c)
			fmt.Println("--------------------------------------")
			filterCount := 0
			for j, n := range c.Nodes {
				fmt.Println(n)
				fmt.Printf("------------ node %d -----------------\n", j)
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
					fmt.Print(min)
					if len(min.Case) <= 0 || min.Case[len(min.Case)-1] != '\n' {
						fmt.Println()
					}
					fmt.Println("--------------------------------------")
					break
				}
				if n.Test == nil {
					// skip this graph
					errors.Logf("INFO", "filtered (no test) %d %v", i, n)
					fmt.Println("--------------------------------------")
				} else if oracle == nil {
					filtered = append(filtered, c)
					break
				} else {
					var profile []byte
					var failures []byte
					var ok bool
					for len(profile) <= 0 {
						var err error
						_, _, profile, failures, ok, err = n.Test.ExecuteWith(oracle)
						if err != nil {
							return nil, err
						}
					}
					if false {
						errors.Logf("INFO", "ran failure oracle %v %v %v", len(n.Test.Case), len(failures), ok)
					}
					if len(failures) > 0 {
						filtered = append(filtered, c)
						break
					} else {
						errors.Logf("INFO", "filtered (passing test) %d %v", j, n)
						fmt.Println("--------------------------------------")
						passing = append(passing, n)
						filterCount++
						if filterCount >= 2 {
							break
						}
					}
				}
			}
		}
	} else {
		filtered = clusters
	}
	return filtered, nil
}


func (clusters Clusters) Colors() map[int][]*Cluster {
	colors := make(map[int][]*Cluster)
	for _, clstr := range clusters {
		added := make(map[int]bool)
		if false {
			errors.Logf("DEBUG", "%v", clstr)
		}
		for _, n := range clstr.Nodes {
			for j := range n.Node.SubGraph.V {
				if added[n.Node.SubGraph.V[j].Color] {
					continue
				}
				colors[n.Node.SubGraph.V[j].Color] = append(colors[n.Node.SubGraph.V[j].Color], clstr)
				added[n.Node.SubGraph.V[j].Color] = true
			}
		}
	}
	return colors
}

func (clusters Clusters) RankColors(score Score, lat *lattice.Lattice) Result {
	return RankColors(score, lat, clusters.Colors())
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
			color,
			lat.Info.Positions[color],
			lat.Info.FnNames[color],
			lat.Info.BBIds[color],
			s,
		})
	}
	result.Sort()
	return result
}

func ScoreColor(score Score, lat *lattice.Lattice, color int, in []*Cluster) float64 {
	epsilon := .025
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
	for _, c := range in {
		rm := s/float64(t)
		if t < 1 || abs(c.Score - rm) < epsilon {
			s += c.Score
			t++
		} else {
			if false {
				errors.Logf("DEBUG", "skipped %v %v %v", lat.Labels.Label(color), rm, c.Score)
			}
		}
	}
	return colorScore * (s / float64(t))
}

func RankColors(score Score, lat *lattice.Lattice, colors map[int][]*Cluster) Result {
	if score == nil {
		panic("nil score")
	}
	result := make(Result, 0, len(colors))
	for color, clusters := range colors {
		result = append(result, Location{
			stat.Location{
				color,
				lat.Info.Positions[color],
				lat.Info.FnNames[color],
				lat.Info.BBIds[color],
				ScoreColor(score, lat, color, clusters),
			},
			clusters,
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
	return WalkFrom(cur, score, lat)
}

func WalkFromColor(color int, score Score, lat *lattice.Lattice) (*SearchNode) {
	// color := lat.Labels.Color("(*dynagrok/examples/avl.Avl).Verify blk 3")
	vsg := subgraph.Build(1, 0).FromVertex(color).Build()
	embIdxs := lat.Fail.ColorIndex[color]
	embs := make([]*subgraph.Embedding, 0, len(embIdxs))
	for _, embIdx := range embIdxs {
		embs = append(embs, subgraph.StartEmbedding(subgraph.VertexEmbedding{SgIdx: 0, EmbIdx: embIdx}))
	}
	colorNode := lattice.NewNode(lat, vsg, embs)
	cur := &SearchNode{
		Node: colorNode,
		Score: score(lat, colorNode),
	}
	return WalkFrom(cur, score, lat)
}

func WalkFrom(cur *SearchNode, score Score, lat *lattice.Lattice) (*SearchNode) {
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
	for ; i < len(weights) - 1 && r > weights[i]; i++ {
		r -= weights[i]
	}
	return i
}
