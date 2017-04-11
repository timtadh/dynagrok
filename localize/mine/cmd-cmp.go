package mine

import (
	"fmt"
	"time"
	"sort"
	"math"
)

import (
	"github.com/timtadh/getopt"
)

import (
	"github.com/timtadh/dynagrok/cmd"
)

func NewCompareParser(c *cmd.Config, o *Options) cmd.Runnable {
	var o1, o2 Options
	var wo1, wo2 walkOpts
	bb1 := NewBranchAndBoundParser(c, &o1)
	sleap1 := NewSLeapParser(c, &o1)
	leap1 := NewLeapParser(c, &o1)
	urw1 := NewURWParser(c, &o1, &wo1)
	swrw1 := NewSWRWParser(c, &o1, &wo1)
	walks1 := NewWalksParser(c, &o1, &wo1)
	topColors1 := NewWalkTopColorsParser(c, &o1, &wo1)
	walkTypes1 := cmd.Commands(map[string]cmd.Runnable{
		walks1.Name():     walks1,
		topColors1.Name(): topColors1,
	})
	bb2 := NewBranchAndBoundParser(c, &o2)
	sleap2 := NewSLeapParser(c, &o2)
	leap2 := NewLeapParser(c, &o2)
	urw2 := NewURWParser(c, &o2, &wo2)
	swrw2 := NewSWRWParser(c, &o2, &wo2)
	walks2 := NewWalksParser(c, &o2, &wo2)
	topColors2 := NewWalkTopColorsParser(c, &o2, &wo2)
	walkTypes2 := cmd.Commands(map[string]cmd.Runnable{
		walks2.Name():     walks2,
		topColors2.Name(): topColors2,
	})
	return cmd.Concat(
		cmd.Cmd(
		"compare",
		`[options]`,
		`
Compare a walk based method against leap, s-leap, or branch and bound.

Option Flags
    -h,--help                         Show this message
`,
		"",
		[]string{},
		func(r cmd.Runnable, args []string, optargs []getopt.OptArg) ([]string, *cmd.Error) {
			o1 = *o.Copy()
			o2 = *o.Copy()
			return args, nil
		}),
		cmd.Commands(map[string]cmd.Runnable{
			bb1.Name(): bb1,
			sleap1.Name(): sleap1,
			leap1.Name(): leap1,
			urw1.Name():   cmd.Concat(urw1, walkTypes1),
			swrw1.Name():  cmd.Concat(swrw1, walkTypes1),
		}),
		cmd.Commands(map[string]cmd.Runnable{
			bb2.Name(): bb2,
			sleap2.Name(): sleap2,
			leap2.Name(): leap2,
			urw2.Name():   cmd.Concat(urw2, walkTypes2),
			swrw2.Name():  cmd.Concat(swrw2, walkTypes2),
		}),
		cmd.BareCmd(
		func(r cmd.Runnable, args []string, optargs []getopt.OptArg) ([]string, *cmd.Error) {
			min := func(a, b int) int {
				if a < b {
					return a
				}
				return b
			}
			timeit := func(m *Miner) ([]*SearchNode, time.Duration) {
				s := time.Now()
				nodes := m.Mine().unique()
				e := time.Now()
				return nodes, e.Sub(s)
			}
			sum := func(nodes []*SearchNode) float64 {
				sum := 0.0
				for _, n := range nodes {
					sum += n.Score
				}
				return sum
			}
			mean := func(nodes []*SearchNode) float64 {
				return sum(nodes)/float64(len(nodes))
			}
			stddev := func(nodes []*SearchNode) float64 {
				u := mean(nodes)
				variance := 0.0
				for _, n := range nodes {
					variance += (n.Score - u) * (n.Score - u)
				}
				if len(nodes) > 2 {
					variance = (1/(float64(len(nodes)) - 1)) * variance
				} else {
					variance = (1/float64(len(nodes))) * variance
				}
				return math.Sqrt(variance)
			}
			stderr := func(X, Y []*SearchNode) float64 {
				T := min(len(X), len(Y))
				variance := 0.0
				for i := 0; i < T; i++ {
					variance += (X[i].Score - Y[i].Score) * (X[i].Score - Y[i].Score)
				}
				if T > 2 {
					variance = (1/(float64(T) - 1)) * variance
				} else {
					variance = (1/float64(T)) * variance
				}
				return math.Sqrt(variance)
			}
			statsHeader := func() {
				fmt.Printf(
					"%-20v %15v %15v %15v %15v\n", "", "sum", "mean", "stddev", "duration")

			}
			stats := func(name string, nodes []*SearchNode, dur time.Duration) {
				fmt.Printf(
					"%-20v %15.5g %15.5g %15.5g %15v\n", name, sum(nodes), mean(nodes), stddev(nodes), dur)
			}
			output := func(name string, nodes []*SearchNode) {
				fmt.Println()
				fmt.Println(name)
				for i, n := range nodes {
					fmt.Printf("  - subgraph %-5d %v\n", i, n)
					fmt.Println()
				}
				fmt.Println()
			}
			a := NewMiner(o1.Miner, o1.Lattice, o1.Score, o1.Opts...)
			b := NewMiner(o2.Miner, o2.Lattice, o2.Score, o2.Opts...)
			A, aTime := timeit(a)
			B, bTime := timeit(b)
			A = A[:min(len(A), len(B))]
			B = B[:min(len(A), len(B))]

			output(o1.MinerName, A)
			output(o2.MinerName, B)
			statsHeader()
			stats(o1.MinerName, A, aTime)
			stats(o2.MinerName, B, bTime)
			fmt.Printf("%-20v %15.5g\n", "stderr:", stderr(A, B))
			fmt.Println()
			return args, nil
		}),
	)
}

func (nodes SearchNodes) unique() (unique []*SearchNode) {
	added := make(map[string]bool)
	for n, next := nodes(); next != nil; n, next = next() {
		if n.Node.SubGraph == nil {
			continue
		}
		label := string(n.Node.SubGraph.Label())
		if added[label] {
			continue
		}
		added[label] = true
		unique = append(unique, n)
	}
	sort.Slice(unique, func(i, j int) bool {
		return unique[i].Score > unique[j].Score
	})
	return unique
}

func (nodes SearchNodes) group() [][]*SearchNode {
	unique := nodes.unique()
	groups := make([][]*SearchNode, 0, 10)
	for _, n := range unique {
		lg := len(groups)
		if lg > 0 && n.Score == groups[lg-1][0].Score {
			groups[lg-1] = append(groups[lg-1], n)
		} else {
			groups = append(groups, make([]*SearchNode, 0, 10))
			groups[lg] = append(groups[lg], n)
		}
	}
	return groups
}
