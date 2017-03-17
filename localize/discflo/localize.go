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
	"github.com/timtadh/dynagrok/localize/lattice/subgraph"
	"github.com/timtadh/dynagrok/localize/mine"
	"github.com/timtadh/dynagrok/localize/stat"
	"github.com/timtadh/dynagrok/localize/test"
)

type Location struct {
	stat.Location
	Clusters []*Cluster
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

func Localize(m *mine.Miner, tests []*test.Testcase, oracle test.Executor) (Clusters, error) {
	added := make(map[string]bool)
	db := NewDbScan(.05)
	i := 0
	for n, next := m.Mine()(); next != nil; n, next = next() {
		if n.Node.SubGraph == nil {
			continue
		}
		label := string(n.Node.SubGraph.Label())
		if added[label] {
			continue
		}
		added[label] = true
		db.Add(n)
		if true {
			errors.Logf("DEBUG", "found %d %v", i, n)
		}
		i++
	}
	return clusters(tests, oracle, m, db)
}

func clusters(tests []*test.Testcase, oracle test.Executor, m *mine.Miner, db *DbScan) (Clusters, error) {
	clusters := db.Clusters()
	sort.Slice(clusters, func(i, j int) bool {
		return clusters[i].Score > clusters[j].Score
	})
	for _, c := range clusters {
		sort.Slice(c.Nodes, func(i, j int) bool {
			return c.Nodes[i].Score > c.Nodes[j].Score
		})
	}
	passing := make([]*mine.SearchNode, 0, 10)
	filtered := make([]*Cluster, 0, 10)
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
				fmt.Println(RankNodes(m, n.Node.SubGraph))
				fmt.Println("--------------------------------------")
				for count := 0; count < len(tests); count++ {
					j := rand.Intn(len(tests))
					t := tests[j]
					min, err := t.Minimize(m.Lattice, n.Node.SubGraph)
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

func (clusters Clusters) RankColors(m *mine.Miner) Result {
	return RankColors(m, clusters.Colors())
}

func ScoreColor(m *mine.Miner, color int, in []*Cluster) float64 {
	abs := func(x float64) float64 {
		if x < 0 {
			return -x
		}
		return x
	}
	epsilon := .025
	n := mine.ColorNode(m.Lattice, m.Score, color)
	var s float64
	t := 0
	for _, c := range in {
		rm := s / float64(t)
		if t < 1 || abs(c.Score-rm) < epsilon {
			s += c.Score
			t++
		} else {
			if false {
				errors.Logf("DEBUG", "skipped %v %v %v", m.Lattice.Labels.Label(color), rm, c.Score)
			}
		}
	}
	return n.Score * (s / float64(t))
}

func RankColors(m *mine.Miner, colors map[int][]*Cluster) Result {
	result := make(Result, 0, len(colors))
	for color, clusters := range colors {
		bbid, fnName, pos := m.Lattice.Info.Get(color)
		result = append(result, Location{
			stat.Location{
				color,
				pos,
				fnName,
				bbid,
				ScoreColor(m, color, clusters),
			},
			clusters,
		})
	}
	result.Sort()
	return result
}

func RankNodes(m *mine.Miner, sg *subgraph.SubGraph) stat.Result {
	result := make(stat.Result, 0, len(sg.V))
	for i := range sg.V {
		color := sg.V[i].Color
		n := mine.ColorNode(m.Lattice, m.Score, color)
		bbid, fnName, pos := m.Lattice.Info.Get(color)
		result = append(result, stat.Location{
			color,
			pos,
			fnName,
			bbid,
			n.Score,
		})
	}
	result.Sort()
	return result
}
