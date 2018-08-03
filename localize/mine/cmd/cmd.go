package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/timtadh/data-structures/errors"
	"github.com/timtadh/dynagrok/cmd"
	"github.com/timtadh/dynagrok/localize/lattice"
	"github.com/timtadh/dynagrok/localize/mine"
	"github.com/timtadh/dynagrok/localize/mine/algoparsers"
	evalcmd "github.com/timtadh/dynagrok/localize/mine/eval"
	"github.com/timtadh/dynagrok/localize/mine/opts"
	"github.com/timtadh/dynagrok/localize/test"
	"github.com/timtadh/getopt"
)

var Notes string = `
Notes on Binary Args (-a,--binary-args)

    In order for the instrumented binary to be run the discflo needs to know
    how to run it. Specifically what command line flags should be given and how
    to supply in input. By default no flags are given and the input is supplied
    on standard in. Here are some usage examples:

    No flags, test input on standard in:

        $ dynagrok localize discflo <...> -a '\<$test'

    Several flags test input on standard in:

        $ dynagrok localize discflo <...> -a '\-o /dev/null --verbose <$test'

    Test input as an argument

        $ dynagrok localize discflo <...> -a '\$test'

    Test input as an argument to a flag

        $ dynagrok localize discflo <...> -a '\-i $test'
        $ dynagrok localize discflo <...> -a '\--input $test'

    Notes
    1. '\--input=$test' is currently not supported!
    2. Only one input is currently allowed
`

func NewCommand(c *cmd.Config) cmd.Runnable {
	var o opts.Options
	cmp := evalcmd.NewCompareParser(c, &o)
	eval := NewEvalParser(c, &o)
	return cmd.Concat(
		cmd.Annotate(
			NewOptionParser(c, &o),
			"mine-dsg",
			"", "[options]",
			"Mine Discriminative Subgraphs\nOptions", Notes,
		),
		cmd.Commands(map[string]cmd.Runnable{
			"": cmd.Concat(
				NewAlgorithmParser(c, &o),
				cmd.Commands(map[string]cmd.Runnable{
					"":          NewRunner(c, &o),
					eval.Name(): eval,
				}),
			),
			cmp.Name(): cmp,
		}),
	)
}

func NewRunner(c *cmd.Config, o *opts.Options) cmd.Runnable {
	return cmd.BareCmd(
		func(r cmd.Runnable, args []string, optargs []getopt.OptArg) ([]string, *cmd.Error) {
			if o.Score == nil {
				return nil, cmd.Usage(r, 2, "You must supply a score (see -s or --scores)")
			}
			subgraphs := make([]*mine.SearchNode, 0, 10)
			added := make(map[string]bool)
			miner := mine.NewMiner(o.Miner, o.Lattice, o.Score, o.Opts...)
			for n, next := miner.Mine(context.TODO())(); next != nil; n, next = next() {
				if n.Node.SubGraph == nil {
					continue
				}
				label := string(n.Node.SubGraph.Label())
				if added[label] {
					continue
				}
				added[label] = true
				subgraphs = append(subgraphs, n)
				if true {
					errors.Logf("DEBUG", "found %d %v", len(subgraphs), n)
				}
			}
			minOrder := func(n *lattice.Node) int {
				min := -1
				set := false
				for _, emb := range n.Embeddings {
					for c := emb; c != nil; c = c.Prev {
						nodeAttrs := o.Lattice.NodeAttrs[c.EmbIdx]
						if o, has := nodeAttrs["order"]; has {
							os := o.(string)
							if order, err := strconv.Atoi(os); err == nil {
								if !set || order < min {
									set = true
									min = order
								}
							}
						}
					}
				}
				return min
			}
			sort.Slice(subgraphs, func(i, j int) bool {
				oi := minOrder(subgraphs[i].Node)
				oj := minOrder(subgraphs[j].Node)
				if oi >= 0 && oj >= 0 {
					return (subgraphs[i].Score == subgraphs[j].Score && oi < oj) || subgraphs[i].Score > subgraphs[j].Score
				}
				return subgraphs[i].Score > subgraphs[j].Score
			})
			fmt.Println()
			for i, n := range subgraphs {
				m := minOrder(n.Node)
				if m >= 0 {
					fmt.Printf("  - subgraph (%d) %-5d %v\n", m, i, n)
				} else {
					fmt.Printf("  - subgraph %-5d %v\n", i, n)
				}
				fmt.Println()
			}
			fmt.Println()
			return nil, nil
		})
}

func NewAlgorithmParser(c *cmd.Config, o *opts.Options) cmd.Runnable {
	var wo algoparsers.WalkOpts
	bb := algoparsers.NewBranchAndBoundParser(c, o)
	sleap := algoparsers.NewSLeapParser(c, o)
	leap := algoparsers.NewLeapParser(c, o)
	urw := algoparsers.NewURWParser(c, o, &wo)
	swrw := algoparsers.NewSWRWParser(c, o, &wo)
	walks := algoparsers.NewWalksParser(c, o, &wo)
	topColors := algoparsers.NewWalkTopColorsParser(c, o, &wo)
	walkTypes := cmd.Commands(map[string]cmd.Runnable{
		walks.Name():     walks,
		topColors.Name(): topColors,
	})
	return cmd.Commands(map[string]cmd.Runnable{
		bb.Name():    bb,
		sleap.Name(): sleap,
		leap.Name():  leap,
		urw.Name():   cmd.Concat(urw, walkTypes),
		swrw.Name():  cmd.Concat(swrw, walkTypes),
	})
}

func NewOptionParser(c *cmd.Config, o *opts.Options) cmd.Runnable {
	return cmd.Cmd(
		"",
		`-s <score> -b <binary> -f <failing-tests> -p <passing-tests>`,
		`
--scores                          List of available suspiciousness scores
-s,--score=<score>                Suspiciousness score to use
-b,--binary=<path>                The binary to test. It should be
                                  instrumented.
                                  (see: dynagrok instrument -h)
-a,--binary-args=<string>         Argument flags/files/pattern for the 
                                  binary under test. (optional) (see notes below)
-f,--failing-tests=<path>         Failing test case to minimize. (May be
                                  specified multiple times or with a comma
                                  separated list).
-p,--passing-tests=<path>         A non-failing profile or profiles. (May be
                                  specified multiple times or with a comma
                                  separated list).
--max-edges=<int>                 Maximal number of edges in a mined pattern
--min-edges=<int>                 Minimum number of edges in a mined pattern
--min-fails=<int>                 Minimum number of failures associated with
                                  each behavior.
`,
		"s:b:a:f:p:",
		[]string{
			"score=",
			"scores",
			"binary=",
			"binary-args=",
			"passing-tests=",
			"failing-tests=",
			"max-edges=",
			"min-edges=",
			"min-fails=",
		},
		func(r cmd.Runnable, args []string, optargs []getopt.OptArg) ([]string, *cmd.Error) {
			ba, err := test.ParseArgs("<$stdin")
			if err != nil {
				return nil, cmd.Errorf(3, "Unexpected error: %v", err)
			}
			o.BinArgs = ba
			var passingPaths []string
			var failingPaths []string
			for _, oa := range optargs {
				switch oa.Opt() {
				case "--scores":
					fmt.Println("\nNames of Suspicousness Scores (and Abbrevations):")
					for name, abbrvs := range mine.ScoreNames {
						fmt.Printf("  - %v : [%v]\n", name, strings.Join(abbrvs, ", "))
					}
					return nil, cmd.Errorf(0, "")
				case "-s", "--score":
					name := oa.Arg()
					if n, has := mine.ScoreAbbrvs[oa.Arg()]; has {
						name = n
					}
					if m, has := mine.Scores[name]; has {
						fmt.Println("using method", name)
						o.Score = m
						o.ScoreName = name
					} else {
						return nil, cmd.Errorf(1, "Localization method '%v' is not supported. (use --methods to get a list)", oa.Arg())
					}
				case "-b", "--binary":
					r, err := test.NewRemote(oa.Arg(), test.MaxMegabytes(50), test.Timeout(5*time.Second), test.Config(c))
					if err != nil {
						return nil, cmd.Err(1, err)
					}
					o.Binary = r
				case "-a", "--binary-args":
					var err error
					o.BinArgs, err = test.ParseArgs(oa.Arg())
					if err != nil {
						return nil, cmd.Errorf(1, "Could not parse the arguments to %v, err: %v", oa.Opt(), err)
					}
				case "-f", "--failing-tests":
					for _, path := range strings.Split(oa.Arg(), ",") {
						failingPaths = append(failingPaths, path)
					}
				case "-p", "--passing-tests":
					for _, path := range strings.Split(oa.Arg(), ",") {
						passingPaths = append(passingPaths, path)
					}
				case "--max-edges":
					m, err := strconv.Atoi(oa.Arg())
					if err != nil {
						return nil, cmd.Errorf(1, "Could not parse arg to `%v` expected an int (got %v). err: %v", oa.Opt(), oa.Arg(), err)
					}
					o.Opts = append(o.Opts, mine.MaxEdges(m))
				case "--min-edges":
					m, err := strconv.Atoi(oa.Arg())
					if err != nil {
						return nil, cmd.Errorf(1, "Could not parse arg to `%v` expected an int (got %v). err: %v", oa.Opt(), oa.Arg(), err)
					}
					o.Opts = append(o.Opts, mine.MinEdges(m))
				case "--min-fails":
					m, err := strconv.Atoi(oa.Arg())
					if err != nil {
						return nil, cmd.Errorf(1, "Could not parse arg to `%v` expected an int (got %v). err: %v", oa.Opt(), oa.Arg(), err)
					}
					o.Opts = append(o.Opts, mine.MinFails(m))
				}
			}
			if len(failingPaths) < 1 {
				return nil, cmd.Usage(r, 2, "Expected at least one failing test. (see -f)")
			}
			if len(passingPaths) < 1 {
				return nil, cmd.Usage(r, 2, "Expected at least one passing test. (see -p)")
			}
			if o.Binary != nil {
				ex, err := test.SingleInputExecutor(o.BinArgs, o.Binary)
				if err != nil {
					return nil, cmd.Err(2, err)
				}
				failing, failingProfiles, err := runTests(failingPaths, ex)
				if err != nil {
					return nil, cmd.Err(1, err)
				}
				passing, passingProfiles, err := runTests(passingPaths, ex)
				if err != nil {
					return nil, cmd.Err(1, err)
				}
				o.Failing = failing
				o.Passing = passing
				o.Lattice, err = lattice.LoadFrom(failingProfiles, passingProfiles)
				if err != nil {
					return nil, cmd.Err(3, err)
				}
			} else {
				o.Lattice, err = lattice.LoadDot(failingPaths, passingPaths)
				if err != nil {
					return nil, cmd.Err(3, err)
				}
			}
			return args, nil
		})
}

func runTests(paths []string, ex test.Executor) ([]*test.Testcase, *bytes.Buffer, error) {
	var buf bytes.Buffer
	tests := make([]*test.Testcase, 0, len(paths))
	count := 0
	for i, path := range paths {
		fmt.Println("loading test", i, path)
		if f, err := os.Open(path); err != nil {
			return nil, nil, fmt.Errorf("Could not open test %v, err: %v", path, err)
		} else {
			bits, err := ioutil.ReadAll(f)
			f.Close()
			if err != nil {
				return nil, nil, fmt.Errorf("Could not read test %v, err: %v", path, err)
			}
			var t *test.Testcase
			for {
				t = test.Test(path, ex, bits)
				err = t.Execute()
				if err != nil {
					return nil, nil, fmt.Errorf("Could not execute the test %d. err: %v", i, err)
				}
				if !t.Usable() {
					count++
					if count < 10 {
						continue
					}
					return nil, nil, fmt.Errorf("Can't use test %d", i)
				} else {
					break
				}
			}
			_, err = buf.Write(t.Profile())
			if err != nil {
				return nil, nil, fmt.Errorf("Could not construct buffer for profiles. test %d err: %v", i, err)
			}
			tests = append(tests, t)
		}
	}
	return tests, &buf, nil
}
