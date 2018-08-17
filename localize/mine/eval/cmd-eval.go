package eval

// Precision and Recall are used by
//    H. Cheng, D. Lo, Y. Zhou, X. Wang, and X. Yan, “Identifying Bug Signatures
//    Using Discriminative Graph Mining,” in Proceedings of the Eighteenth
//    International Symposium on Software Testing and Analysis, 2009, pp.
//    141–152.
//
// Precision refers to the proportion of returned results that highlight the
// bug. Recall refers to the proportion of bugs that can be discovered by the
// returned bug signatures
//
// These metrics are across a whole set of bugs in either a single program or
// multiple programs. So not relevant to this evaluation which focuses on one
// version of one program with one or more bugs.

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/timtadh/data-structures/errors"
	"github.com/timtadh/dynagrok/cmd"
	"github.com/timtadh/dynagrok/localize/eval"
	"github.com/timtadh/dynagrok/localize/fault"
	"github.com/timtadh/dynagrok/localize/mine"
	"github.com/timtadh/dynagrok/localize/mine/opts"
	"github.com/timtadh/getopt"
)

func NewEvalParser(c *cmd.Config, o *opts.Options) cmd.Runnable {
	parser := algorithmParser(c)
	return cmd.Concat(
		cmd.Cmd(
			"eval",
			`[options]`,
			`
Compare a walk based method against leap, s-leap, or branch and bound.

Option Flags
    -h,--help                            Show this message
    -t,--time-out=<seconds>              Timeout for each algorithm (default 120 seconds)
    -f,--faults=<path>                   Path to a fault file.
    -o,--output=<path>                   Place to write CSV of evaluation
    -d,--data-source=<source>            Source of data for analysis: dynagrok (default), defect4j
                                         Note: dynagrok is almost always what you want.
    --htrank-method=<method>             Computation method for HTrank: auto, exact, estimate
                                         default: auto
    --max-states-for-exact-htrank=<int>  Maximum number of states in the chain to use exact rank
                                         only applies when --htrank-method=auto
    --parallelism=<int>                  Number of cores to use for HTrank computation
`,
			"o:f:t:d:",
			[]string{
				"output=",
				"faults=",
				"time-out=",
				"data-source=",
				"htrank-method=",
				"max-states-for-exact-htrank=",
				"parallelism=",
			},
			func(r cmd.Runnable, args []string, optargs []getopt.OptArg) ([]string, *cmd.Error) {
				outputPath := ""
				faultsPath := ""
				dataSource := "dynagrok"
				timeout := 120 * time.Second
				evalOpts := make([]eval.EvaluatorOption, 0, 10)
				for _, oa := range optargs {
					switch oa.Opt() {
					case "-o", "--output":
						outputPath = oa.Arg()
					case "-f", "--faults":
						faultsPath = oa.Arg()
					case "-d", "--data-source":
						switch oa.Arg() {
						case "dynagrok", "defect4j":
							dataSource = oa.Arg()
						default:
							return nil, cmd.Errorf(1, "Flag %v expected a either dynagrok or defect4j got %q.", oa.Opt(), oa.Arg())
						}
					case "-t", "--time-out":
						t, err := time.ParseDuration(oa.Arg())
						if err != nil {
							return nil, cmd.Errorf(1, "Flag %v expected a duration got %q. err: %v", oa.Opt(), oa.Arg(), err)
						}
						timeout = t
					case "--htrank-method":
						switch oa.Arg() {
						case "auto":
						case "exact":
							evalOpts = append(evalOpts, eval.ForceExactHTRank)
						case "estimate":
							evalOpts = append(evalOpts, eval.ForceEstimateHTRank)
						default:
							return nil, cmd.Errorf(1, "Flag %v expected: `auto`, `exact`, or `estimate` got %q.", oa.Opt(), oa.Arg())
						}
					case "--max-states-for-exact-htrank":
						states, err := strconv.Atoi(oa.Arg())
						if err != nil {
							return nil, cmd.Errorf(1, "Flag %v expected an int but got %q. err: %v", oa.Opt(), oa.Arg(), err)
						}
						evalOpts = append(evalOpts, eval.MaxStatesForExactHTRank(states))
					case "--parallelism":
						p, err := strconv.Atoi(oa.Arg())
						if err != nil {
							return nil, cmd.Errorf(1, "Flag %v expected an int but got %q. err: %v", oa.Opt(), oa.Arg(), err)
						}
						evalOpts = append(evalOpts, eval.Parallelism(p))
					}
				}
				if faultsPath == "" {
					return nil, cmd.Errorf(1, "You must supply the `-f` flag and give a path to the faults")
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
				options := make([]*opts.Options, 0, 10)
				for {
					var opt *opts.Options
					var err *cmd.Error
					opt, args, err = parser(o.Copy(), args)
					if err != nil {
						return nil, err
					}
					if opt == nil {
						break
					}
					options = append(options, opt)
				}
				var faults []*fault.Fault
				if dataSource == "dynagrok" {
					var err error
					faults, err = fault.LoadFaults(faultsPath)
					if err != nil {
						return nil, cmd.Err(1, err)
					}
				} else if dataSource == "defect4j" {
					var err error
					faults, err = fault.LoadD4JFaults(faultsPath)
					if err != nil {
						return nil, cmd.Err(1, err)
					}
				} else {
					return nil, cmd.Errorf(1, "unexpected data source")
				}
				for _, f := range faults {
					fmt.Println(f)
				}
				timeit := func(m *mine.Miner) ([]*mine.SearchNode, time.Duration) {
					ctx, cancel := context.WithTimeout(context.Background(), timeout)
					defer cancel()
					s := time.Now()
					nodes := m.Mine(ctx).Unique()
					e := time.Now()
					return nodes, e.Sub(s)
				}
				evaluate := func(m *mine.Miner, options *opts.Options, nodes []*mine.SearchNode, sflType, method, score, chain string) eval.EvalResults {
					errors.Logf("INFO", "evaluating %v %v %v %v", sflType, method, score, chain)
					lattice := options.Lattice
					var evaluator *eval.Evaluator
					if dataSource == "dynagrok" {
						evaluator = eval.NewEvaluator(lattice, eval.NewDynagrokFaultIdentifier(lattice, faults), evalOpts...)
					} else if dataSource == "defect4j" {
						evaluator = eval.NewEvaluator(lattice, eval.NewDefect4J_FaultIdentifier(lattice, faults), evalOpts...)
					}
					var states map[int][]int
					var P [][]float64
					jumpPr := .5
					maxStates := 20000
					finalChainName := chain
					if sflType == "Control" {
						_, jumps := eval.BehavioralAndSpacialJumpMatrix(m)
						states, P = eval.ControlChain(jumps)
					} else if sflType == "CBSFL" {
						switch chain {
						case "Ranked-List":
							groups := eval.CBSFL(options, options.Score)
							return evaluator.RankListEval(method, score, groups)
						case "Spacial-Jumps":
							states, P = eval.SpacialJumps(jumpPr, maxStates, m)
							finalChainName = fmt.Sprintf("%v(%g)", chain, jumpPr)
						case "Behavioral-Jumps":
							states, P = eval.BehavioralJumps(jumpPr, maxStates, m)
							finalChainName = fmt.Sprintf("%v(%g)", chain, jumpPr)
						case "Behavioral+Spacial-Jumps":
							states, P = eval.BehavioralAndSpacialJumps(jumpPr, maxStates, m)
							finalChainName = fmt.Sprintf("%v(%g)", chain, jumpPr)
						default:
							panic(fmt.Errorf("no chain named %v", method))
						}
					} else if sflType == "SBBFL" {
						switch chain {
						case "Ranked-List":
							return evaluator.SBBFLRankListEval(nodes, method, score)
						case "Markov-Ranked-List":
							states, P = eval.DsgMarkovChain(maxStates, m, nodes, 0, nil)
						case "Spacial-Jumps":
							_, jumps := eval.SpacialJumpMatrix(m)
							states, P = eval.DsgMarkovChain(maxStates, m, nodes, jumpPr, jumps)
							finalChainName = fmt.Sprintf("%v(%g)", chain, jumpPr)
						case "Behavioral-Jumps":
							_, jumps := eval.BehavioralJumpMatrix(m)
							states, P = eval.DsgMarkovChain(maxStates, m, nodes, jumpPr, jumps)
							finalChainName = fmt.Sprintf("%v(%g)", chain, jumpPr)
						case "Behavioral+Spacial-Jumps":
							_, jumps := eval.BehavioralAndSpacialJumpMatrix(m)
							states, P = eval.DsgMarkovChain(maxStates, m, nodes, jumpPr, jumps)
							finalChainName = fmt.Sprintf("%v(%g)", chain, jumpPr)
						default:
							panic(fmt.Errorf("no chain named %v", chain))
						}
					} else {
						panic("unknown sfl type")
					}
					return evaluator.HTRank(method, score, finalChainName, states, P)
				}
				extractResult := func(results eval.EvalResults) (result eval.EvalResult) {
					if len(results) > 1 {
						result = results.Avg()
					} else if len(results) < 1 {
						result = nil
					} else {
						result = results[0]
					}
					errors.Logf("INFO", "computed:\n%v\n", result)
					return result
				}
				filterAppend := func(filter func(r eval.EvalResult) bool) func(slice eval.EvalResults, items ...eval.EvalResult) eval.EvalResults {
					return func(slice eval.EvalResults, items ...eval.EvalResult) eval.EvalResults {
						out := slice
						for _, x := range items {
							if !filter(x) {
								out = append(out, x)
							}
						}
						return out
					}
				}
				nonNilAppend := filterAppend(func(r eval.EvalResult) bool {
					return r == nil
				})
				minout := -1
				outputs := make([][]*mine.SearchNode, 0, len(options))
				times := make([]time.Duration, 0, len(options))
				miners := make([]*mine.Miner, 0, len(options))
				for _, opt := range options {
					a := mine.NewMiner(opt.Miner, opt.Lattice, opt.Score, opt.Opts...)
					miners = append(miners, a)
					A, aTime := timeit(a)
					outputs = append(outputs, A)
					times = append(times, aTime)
					if minout < 0 || len(A) < minout {
						minout = len(A)
					}
				}
				var results eval.EvalResults
				if false {
					fmt.Println("Control")
					results = nonNilAppend(results, extractResult(evaluate(miners[0], options[0], outputs[0], "Control", "control", "", "Control")))
				}
				if true {
					fmt.Println("CBSFL")
					scoresSeen := make(map[string]bool)
					for i := range outputs {
						if scoresSeen[options[i].ScoreName] {
							continue
						}
						scoresSeen[options[i].ScoreName] = true
						results = nonNilAppend(results, extractResult(evaluate(miners[i], options[i], outputs[i], "CBSFL", "cbsfl", options[i].ScoreName, "Ranked-List")))
						if true {
							results = nonNilAppend(results, extractResult(evaluate(miners[i], options[i], outputs[i], "CBSFL", "cbsfl", options[i].ScoreName, "Spacial-Jumps")))
							if true {
								results = nonNilAppend(results, extractResult(evaluate(miners[i], options[i], outputs[i], "CBSFL", "cbsfl", options[i].ScoreName, "Behavioral-Jumps")))
								results = nonNilAppend(results, extractResult(evaluate(miners[i], options[i], outputs[i], "CBSFL", "cbsfl", options[i].ScoreName, "Behavioral+Spacial-Jumps")))
							}
						}
					}
				}
				if true {
					fmt.Println("SBBFL")
					for i := range outputs {
						results = nonNilAppend(results, extractResult(evaluate(miners[i], options[i], outputs[i], "SBBFL", options[i].MinerName, options[i].ScoreName, "Ranked-List")))
						if true {
							results = nonNilAppend(results, extractResult(evaluate(miners[i], options[i], outputs[i], "SBBFL", options[i].MinerName, options[i].ScoreName, "Markov-Ranked-List")))
							if true {
								results = nonNilAppend(results, extractResult(evaluate(miners[i], options[i], outputs[i], "SBBFL", options[i].MinerName, options[i].ScoreName, "Spacial-Jumps")))
								results = nonNilAppend(results, extractResult(evaluate(miners[i], options[i], outputs[i], "SBBFL", options[i].MinerName, options[i].ScoreName, "Behavioral-Jumps")))
								results = nonNilAppend(results, extractResult(evaluate(miners[i], options[i], outputs[i], "SBBFL", options[i].MinerName, options[i].ScoreName, "Behavioral+Spacial-Jumps")))
							}
						}
					}
				}
				fmt.Fprintln(ouf, results)
				return args, nil
			}),
	)
}
