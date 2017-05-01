package mine

import (
	"fmt"
	"math"
	"os"
	"sort"
	"time"

	"github.com/timtadh/dynagrok/cmd"
	"github.com/timtadh/getopt"
)

// TODO(tim):
//
// Thread timeouts through all algorithms
// Provide FP-filtering analysis. (maybe????)
// Provide Fault Localization Accuracy Report
// Compare top-k, top-k maximal for branch-and-bound, sLeap, LEAP

func algorithmParser(c *cmd.Config) func(o *Options, args []string) (*Options, []string, *cmd.Error) {
	var wo walkOpts
	return func(o *Options, args []string) (*Options, []string, *cmd.Error) {
		bb := NewBranchAndBoundParser(c, o)
		sleap := NewSLeapParser(c, o)
		leap := NewLeapParser(c, o)
		urw := NewURWParser(c, o, &wo)
		swrw := NewSWRWParser(c, o, &wo)
		walks := NewWalksParser(c, o, &wo)
		topColors := NewWalkTopColorsParser(c, o, &wo)
		walkTypes := cmd.Commands(map[string]cmd.Runnable{
			walks.Name():     walks,
			topColors.Name(): topColors,
		})
		options := map[string]cmd.Runnable{
			bb.Name():    bb,
			sleap.Name(): sleap,
			leap.Name():  leap,
			urw.Name():   cmd.Concat(urw, walkTypes),
			swrw.Name():  cmd.Concat(swrw, walkTypes),
		}
		command := cmd.Commands(options)
		if len(args) <= 0 {
			return nil, nil, nil
		}
		if _, has := options[args[0]]; !has {
			return nil, args, nil
		}
		leftover, err := command.Run(args)
		if err != nil {
			return nil, nil, err
		}
		return o, leftover, nil
	}
}

func NewCompareParser(c *cmd.Config, o *Options) cmd.Runnable {
	parser := algorithmParser(c)
	return cmd.Concat(
		cmd.Cmd(
			"compare",
			`[options]`,
			`
Compare a walk based method against leap, s-leap, or branch and bound.

Option Flags
    -h,--help                         Show this message
    -f,--faults=<path>                Path to a fault file.
    -o,--output=<path>                Place to write CSV of evaluation
`,
			"o:f:",
			[]string{
				"output=",
				"faults=",
			},
			func(r cmd.Runnable, args []string, optargs []getopt.OptArg) ([]string, *cmd.Error) {
				outputPath := ""
				faultsPath := ""
				for _, oa := range optargs {
					switch oa.Opt() {
					case "-o", "--output":
						outputPath = oa.Arg()
					case "-f", "--faults":
						faultsPath = oa.Arg()
					}
				}
				if faultsPath == "" {
					return nil, cmd.Errorf(1, "You must supply the `-f` flag and give a path to the faults")
				}
				faults, err := LoadFaults(faultsPath)
				if err != nil {
					return nil, cmd.Err(1, err)
				}
				for _, f := range faults {
					fmt.Println(f)
				}
				ouf := os.Stdout
				if outputPath != "" {
					f, err := os.Create(outputPath)
					if err != nil {
						return nil, cmd.Err(1, err)
					}
					defer f.Close()
					ouf = f
				}
				opts := make([]*Options, 0, 10)
				for {
					var opt *Options
					var err *cmd.Error
					opt, args, err = parser(o.Copy(), args)
					if err != nil {
						return nil, err
					}
					if opt == nil {
						break
					}
					opts = append(opts, opt)
				}
				min := func(a, b int) int {
					if a < b {
						return a
					}
					return b
				}
				timeit := func(m *Miner) ([]*SearchNode, time.Duration) {
					s := time.Now()
					nodes := m.Mine().Unique()
					e := time.Now()
					return nodes, e.Sub(s)
				}
				rankScore := func(nodes []*SearchNode) float64 {
					min := -1.0
					for _, f := range faults {
						sum := 0.0
					groupLoop:
						for _, g := range group(nodes) {
							for _, n := range g {
								for _, v := range n.Node.SubGraph.V {
									b, fn, _ := o.Lattice.Info.Get(v.Color)
									if fn == f.FnName && b == f.BasicBlockId {
										score := (float64(len(g)) / 2.0) + sum
										if min <= 0 || score < min {
											min = score
										}
										break groupLoop
									}
								}
							}
							sum += float64(len(g))
						}
					}
					if min <= 0 {
						return math.Inf(1)
					}
					return min
				}
				sum := func(nodes []*SearchNode) float64 {
					sum := 0.0
					for _, n := range nodes {
						sum += n.Score
					}
					return sum
				}
				mean := func(nodes []*SearchNode) float64 {
					return sum(nodes) / float64(len(nodes))
				}
				stddev := func(nodes []*SearchNode) float64 {
					u := mean(nodes)
					variance := 0.0
					for _, n := range nodes {
						variance += (n.Score - u) * (n.Score - u)
					}
					if len(nodes) > 2 {
						variance = (1 / (float64(len(nodes)) - 1)) * variance
					} else {
						variance = (1 / float64(len(nodes))) * variance
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
						variance = (1 / (float64(T) - 1)) * variance
					} else {
						variance = (1 / float64(T)) * variance
					}
					if sum(X) >= sum(Y) {
						return math.Sqrt(variance)
					} else {
						return -math.Sqrt(variance)
					}
				}
				statsHeader := func() {
					fmt.Fprintf(ouf,
						"%9v, %9v, %3v, %-30v, %12v, %12v, %12v, %12v, %12v, %12v, %12v, %12v\n",
						"max-edges", "min-fails", "row", "name", "sum", "mean", "stddev", "stderr (0)", "stderr (1)", "rank-score", "dur (sec)", "duration")

				}
				stats := func(maxEdges, minFails, row int, name string, minout int, base1, base2, nodes []*SearchNode, dur time.Duration) {
					base1c := base1[:minout]
					base2c := base2[:minout]
					clamp := nodes[:minout]
					fmt.Fprintf(ouf,
						"%9v, %9v, %3v, %-30v, %12.5g, %12.5g, %12.5g, %12.5g, %12.5g, %12.5g, %12.5g, %12v\n",
						maxEdges, minFails, row, name,
						sum(clamp), mean(clamp), stddev(clamp), stderr(base1c, clamp), stderr(base2c, clamp), rankScore(nodes),
						dur.Seconds(), dur)
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
				maxEdges := 0
				minFails := 0
				minout := -1
				outputs := make([][]*SearchNode, 0, len(opts))
				times := make([]time.Duration, 0, len(opts))
				for _, opt := range opts {
					a := NewMiner(opt.Miner, opt.Lattice, opt.Score, opt.Opts...)
					A, aTime := timeit(a)
					outputs = append(outputs, A)
					times = append(times, aTime)
					if minout < 0 || len(A) < minout {
						minout = len(A)
					}
					maxEdges = a.MaxEdges
					minFails = a.MinFails
				}
				for i := range outputs {
					output(opts[i].MinerName, outputs[i][:minout])
				}
				statsHeader()
				for i := range outputs {
					stats(maxEdges, minFails, i, opts[i].MinerName, minout, outputs[0], outputs[1], outputs[i], times[i])
				}
				fmt.Println()
				return args, nil
			}),
	)
}

func (nodes SearchNodes) Unique() (unique []*SearchNode) {
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

func (nodes SearchNodes) Group() [][]*SearchNode {
	return group(nodes.Unique())
}

func group(unique []*SearchNode) [][]*SearchNode {
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
