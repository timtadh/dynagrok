package eval

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/timtadh/dynagrok/cmd"
	"github.com/timtadh/dynagrok/localize/discflo"
	"github.com/timtadh/dynagrok/localize/eval"
	"github.com/timtadh/dynagrok/localize/mine"
	"github.com/timtadh/getopt"
)

func NewCommand(c *cmd.Config, o *discflo.Options) cmd.Runnable {
	return cmd.Cmd(
		"eval",
		`[options]`,
		`
Evaluate a fault localization method from ground truth

Option Flags
    -h,--help                         Show this message
    -f,--faults=<path>                Path to a fault file.
    --max=<int>                       Maximum number of states in the chain
    -j,--jump-prs=<float64>           Probability of taking jumps in chains which have them
    -m,--method=<method>
    -e,--eval-method=<eval-method>

Methods
	DISCFLO
	SBBFL
	CBSFL

Eval Methods
	RankList
	Markov
`,
		"f:j:m:e:",
		[]string{
			"faults=",
			"max=",
			"jump-prs=",
			"method=",
			"eval-method=",
		},
		func(r cmd.Runnable, args []string, optargs []getopt.OptArg) ([]string, *cmd.Error) {
			methods := make([]string, 0, 10)
			evalMethods := make([]string, 0, 10)
			max := 100
			faultsPath := ""
			jumpPrs := []float64{}
			for _, oa := range optargs {
				switch oa.Opt() {
				case "-f", "--faults":
					faultsPath = oa.Arg()
				case "-m", "--method":
					for _, part := range strings.Split(oa.Arg(), ",") {
						methods = append(methods, strings.TrimSpace(part))
					}
				case "-e", "--eval-method":
					for _, part := range strings.Split(oa.Arg(), ",") {
						evalMethods = append(evalMethods, strings.TrimSpace(part))
					}
				case "--max":
					var err error
					max, err = strconv.Atoi(oa.Arg())
					if err != nil {
						return nil, cmd.Errorf(1, "For flag %v expected an int got %v. err: %v", oa.Opt, oa.Arg(), err)
					}
				case "-j", "--jump-prs":
					for _, part := range strings.Split(oa.Arg(), ",") {
						jumpPr, err := strconv.ParseFloat(part, 64)
						if err != nil {
							return nil, cmd.Errorf(1, "For flag %v expected a float got %v. err: %v", oa.Opt, oa.Arg(), err)
						}
						if jumpPr < 0 || jumpPr >= 1 {
							return nil, cmd.Errorf(1, "For flag %v expected a float between 0-1. got %v", oa.Opt, oa.Arg())
						}
						jumpPrs = append(jumpPrs, jumpPr)
					}
				}
			}
			if len(jumpPrs) <= 0 {
				jumpPrs = append(jumpPrs, (1. / 10.))
			}
			if len(methods) <= 0 {
				methods = append(methods, "DISCFLO", "SBBFL", "CBSFL")
			}
			if len(evalMethods) <= 0 {
				evalMethods = append(evalMethods, "RankList", "Markov")
			}
			if faultsPath == "" {
				return nil, cmd.Errorf(1, "You must supply the `-f` flag and give a path to the faults")
			}
			faults, err := mine.LoadFaults(faultsPath)
			if err != nil {
				return nil, cmd.Err(1, err)
			}
			fmt.Println("max", max)
			for _, f := range faults {
				fmt.Println(f)
			}
			results := make(eval.EvalResults, 0, 10)
			for _, evalMethod := range evalMethods {
				for _, method := range methods {
					if evalMethod == "Markov" {
						for _, chain := range eval.Chains[method] {
							r, err := eval.Evaluate(faults, o, o.Score, evalMethod, method, o.ScoreName, chain, max, jumpPrs)
							if err != nil {
								return nil, cmd.Err(1, err)
							}
							results = append(results, r...)
						}
					} else if method == "SBBFL" {
						continue
					} else {
						r, err := eval.Evaluate(faults, o, o.Score, evalMethod, method, o.ScoreName, "", max, jumpPrs)
						if err != nil {
							return nil, cmd.Err(1, err)
						}
						results = append(results, r...)
					}
				}
			}
			fmt.Println(results)
			// if o.Score == nil {
			// 	for name, score := range mine.Scores {
			// 		eval.Eval(faults, o.Lattice, "Discflo + "+name, eval.Discflo(o, o.Lattice, score))
			// 		eval.Eval(faults, o.Lattice, name, eval.CBSFL(o, o.Lattice, score))
			// 		colors, P, err := DiscfloMarkovChain(jumpPr, max, o, score)
			// 		if err != nil {
			// 			return nil, cmd.Err(1, err)
			// 		}
			// 		mine.MarkovEval(faults, o.Lattice, "discflo + "+name, colors, P)
			// 		m := mine.NewMiner(o.Miner, o.Lattice, score, o.Opts...)
			// 		colors, P = mine.DsgMarkovChain(max, m)
			// 		mine.MarkovEval(faults, o.Lattice, "mine-dsg + "+name, colors, P)
			// 		colors, P = mine.RankListMarkovChain(max, m)
			// 		mine.MarkovEval(faults, o.Lattice, name, colors, P)
			// 		colors, P = mine.SpacialJumps(jumpPr, max, m)
			// 		mine.MarkovEval(faults, o.Lattice, "spacial jumps + "+name, colors, P)
			// 		colors, P = mine.BehavioralJumps(jumpPr, max, m)
			// 		mine.MarkovEval(faults, o.Lattice, "behavioral jumps + "+name, colors, P)
			// 		colors, P = mine.BehavioralAndSpacialJumps(jumpPr, max, m)
			// 		mine.MarkovEval(faults, o.Lattice, "behavioral and spacial jumps + "+name, colors, P)
			// 	}
			// } else {
			// 	eval.Eval(faults, o.Lattice, "Discflo + "+o.ScoreName, eval.Discflo(o, o.Lattice, o.Score))
			// 	eval.Eval(faults, o.Lattice, o.ScoreName, eval.CBSFL(o, o.Lattice, o.Score))
			// 	colors, P, err := DiscfloMarkovChain(jumpPr, max, o, o.Score)
			// 	if err != nil {
			// 		return nil, cmd.Err(1, err)
			// 	}
			// 	mine.MarkovEval(faults, o.Lattice, "discflo + "+o.ScoreName, colors, P)
			// 	m := mine.NewMiner(o.Miner, o.Lattice, o.Score, o.Opts...)
			// 	colors, P = mine.DsgMarkovChain(max, m)
			// 	mine.MarkovEval(faults, o.Lattice, "mine-dsg + "+o.ScoreName, colors, P)
			// 	colors, P = mine.RankListMarkovChain(max, m)
			// 	mine.MarkovEval(faults, o.Lattice, o.ScoreName, colors, P)
			// 	colors, P = mine.SpacialJumps(jumpPr, max, m)
			// 	mine.MarkovEval(faults, o.Lattice, "spacial jumps + "+o.ScoreName, colors, P)
			// 	colors, P = mine.BehavioralJumps(jumpPr, max, m)
			// 	mine.MarkovEval(faults, o.Lattice, "behavioral jumps + "+o.ScoreName, colors, P)
			// 	colors, P = mine.BehavioralAndSpacialJumps(jumpPr, max, m)
			// 	mine.MarkovEval(faults, o.Lattice, "behavioral and spacial jumps + "+o.ScoreName, colors, P)
			// }
			return nil, nil
		})
}
