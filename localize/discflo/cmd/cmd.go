package cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"
)

import (
	"github.com/timtadh/getopt"
)

import (
	"github.com/timtadh/dynagrok/cmd"
	"github.com/timtadh/dynagrok/localize/discflo"
	"github.com/timtadh/dynagrok/localize/discflo/web"
	"github.com/timtadh/dynagrok/localize/eval"
	"github.com/timtadh/dynagrok/localize/lattice"
	"github.com/timtadh/dynagrok/localize/mine"
	"github.com/timtadh/dynagrok/localize/test"
)

func NewCommand(c *cmd.Config) cmd.Runnable {
	var o discflo.Options
	var wo walkOpts
	bb := NewBranchAndBoundParser(c, &o, &wo)
	sleap := NewSLeapParser(c, &o, &wo)
	urw := NewURWParser(c, &o, &wo)
	swrw := NewSWRWParser(c, &o, &wo)
	walks := NewWalksParser(c, &o, &wo)
	topColors := NewWalkTopColorsParser(c, &o, &wo)
	walkTypes := cmd.Commands(map[string]cmd.Runnable{
		walks.Name():     walks,
		topColors.Name(): topColors,
	})
	evaluate := eval.NewCommand(c, &o)
	web := web.NewCommand(c, &o)
	return cmd.Concat(
		NewOptionParser(c, &o),
		cmd.Commands(map[string]cmd.Runnable{
			bb.Name():    bb,
			sleap.Name(): sleap,
			urw.Name():   cmd.Concat(urw, walkTypes),
			swrw.Name():  cmd.Concat(swrw, walkTypes),
		}),
		cmd.Commands(map[string]cmd.Runnable{
			"":              NewRunner(c, &o),
			evaluate.Name(): evaluate,
			web.Name():      web,
		}),
	)
	// return cmd.Concat(
	// 	NewOptionParser(c, &o),
	// 	cmd.Commands(map[string]cmd.Runnable {
	// 		"": NewRunner(c, &o),
	// 		// "web": web.NewCommand(c, &o),
	// 	}),
	// )
}

func NewOptionParser(c *cmd.Config, o *discflo.Options) cmd.Runnable {
	return cmd.Cmd(
		"disc-flo",
		`[options]`,
		`
Use DISCriminative subgraph Fault LOcalization (disc-flo) to localize faults
from failing and passing runs.

<succeeding-profiles> should be a directory (or file) containing flow-graphs
                      from successful executions of an instrumented copy of the
                      program under test (PUT).

Option Flags
    -h,--help                         Show this message
    -b,--binary=<path>                The binary to test. It should be
                                      instrumented.
                                      (see: dynagrok instrument -h)
    -a,--binary-args=<string>         Argument flags/files/pattern for the 
                                      binary under test. (optional) (see notes below)
    -t,--test=<path>                  Failing test case to minimize. (May be
                                      specified multiple times or with a comma
                                      separated list).
    -s,--score=<score>                Suspiciousness score to use
    --scores                          List of available suspiciousness scores
    --minimize-tests                  Do the test case minimization
    --failure-oracle=<path>           A failure oracle to filter out graphs with
                                      non-failing minimized tests.
    -n,--non-failing=<profile>        A non-failing profile or profiles. (May be
                                      specified multiple times or with a comma
                                      separated list).
    --max-edges=<int>                 Maximal number of edges in a mined pattern
    --min-edges=<int>                 Minimum number of edges in a mined pattern
    --min-fails=<int>                 Minimum number of failures associated with
                                      each behavior.
`,
		"s:b:a:t:n:",
		[]string{
			"binary=",
			"binary-args=",
			"test=",
			"score=",
			"scores",
			"minimize-tests",
			"failure-oracle=",
			"non-failing=",
			"max-edges=",
			"min-edges=",
			"min-fails=",
		},
		func(r cmd.Runnable, args []string, optargs []getopt.OptArg) ([]string, *cmd.Error) {
			binArgs, err := test.ParseArgs("<$stdin")
			if err != nil {
				return nil, cmd.Errorf(3, "Unexpected error: %v", err)
			}
			var oracle *test.Remote
			var testPaths []string
			var okPaths []string
			for _, oa := range optargs {
				switch oa.Opt() {
				case "-b", "--binary":
					r, err := test.NewRemote(oa.Arg(), test.Timeout(10*time.Second), test.Config(c))
					if err != nil {
						return nil, cmd.Err(1, err)
					}
					o.Remote = r
				case "-a", "--binary-args":
					var err error
					binArgs, err = test.ParseArgs(oa.Arg())
					if err != nil {
						return nil, cmd.Errorf(1, "Could not parse the arguments to %v, err: %v", oa.Opt(), err)
					}
				case "--failure-oracle":
					r, err := test.NewRemote(oa.Arg(), test.Timeout(10*time.Second), test.Config(c))
					if err != nil {
						return nil, cmd.Err(1, err)
					}
					oracle = r
				case "-t", "--test":
					for _, path := range strings.Split(oa.Arg(), ",") {
						testPaths = append(testPaths, path)
					}
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
				case "--scores":
					fmt.Println("\nNames of Suspicousness Scores (and Abbrevations):")
					for name, abbrvs := range mine.ScoreNames {
						fmt.Printf("  - %v : [%v]\n", name, strings.Join(abbrvs, ", "))
					}
					return nil, cmd.Errorf(0, "")
				case "--minimize-tests":
					o.Minimize = true
				case "-n", "--non-failing":
					for _, path := range strings.Split(oa.Arg(), ",") {
						okPaths = append(okPaths, path)
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
			if len(testPaths) < 1 {
				return nil, cmd.Usage(r, 2, "Expected at least one test. (see -t)")
			}
			if len(okPaths) < 1 {
				return nil, cmd.Usage(r, 2, "Expected at least one non-failing profile. (see -n)")
			}
			if o.Remote == nil {
				return nil, cmd.Usage(r, 2, "You must supply a binary (see -b)")
			}
			var fails bytes.Buffer
			tests := make([]*test.Testcase, 0, len(testPaths))
			count := 0
			if oracle != nil {
				fex, err := test.SingleInputExecutor(binArgs, oracle)
				if err != nil {
					return nil, cmd.Err(2, err)
				}
				o.Oracle = fex
			}
			ex, err := test.SingleInputExecutor(binArgs, o.Remote)
			if err != nil {
				return nil, cmd.Err(2, err)
			}
			for i, path := range testPaths {
				fmt.Println("loading test", i, path)
				if f, err := os.Open(path); err != nil {
					return nil, cmd.Errorf(1, "Could not open test %v, err: %v", path, err)
				} else {
					bits, err := ioutil.ReadAll(f)
					f.Close()
					if err != nil {
						return nil, cmd.Errorf(1, "Could not read test %v, err: %v", path, err)
					}
					var t *test.Testcase
					for {
						t = test.Test(path, ex, bits)
						err = t.Execute()
						if err != nil {
							return nil, cmd.Usage(r, 2, "Could not execute the test %d. err: %v", i, err)
						}
						if !t.Usable() {
							count++
							if count < 10 {
								continue
							}
							return nil, cmd.Usage(r, 2, "Can't use test %d", i)
						} else {
							break
						}
					}
					_, err = fails.Write(t.Profile())
					if err != nil {
						return nil, cmd.Usage(r, 2, "Could not construct buffer for profiles. test %d err: %v", i, err)
					}
					tests = append(tests, t)
				}
			}
			o.Tests = tests
			oks, okClose, err := cmd.Inputs(okPaths)
			if err != nil {
				return nil, cmd.Usage(r, 2, "Could not open ok profiles, %v. err: %v", okPaths, err)
			}
			defer okClose()
			o.Lattice, err = lattice.LoadFrom(&fails, oks)
			if err != nil {
				return nil, cmd.Err(3, err)
			}
			return args, nil
		})
}

func NewRunner(c *cmd.Config, o *discflo.Options) cmd.Runnable {
	return cmd.BareCmd(
		func(r cmd.Runnable, args []string, optargs []getopt.OptArg) ([]string, *cmd.Error) {
			miner := mine.NewMiner(o.Miner, o.Lattice, o.Score, o.Opts...)
			if o.Score == nil {
				return nil, cmd.Usage(r, 2, "You must supply a score (see -s or --scores)")
			}
			clusters, err := discflo.Localizer(o)(miner)
			if err != nil {
				return nil, cmd.Err(3, err)
			}
			result := clusters.RankColors(miner)
			fmt.Println(result)
			fmt.Println(result.StatResult())
			return nil, nil
		})
}
