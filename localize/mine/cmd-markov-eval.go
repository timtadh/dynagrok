package mine

import (
	"fmt"
	"strconv"
)

import (
	"github.com/timtadh/getopt"
)

import (
	"github.com/timtadh/dynagrok/cmd"
)

func NewMarkovEvalParser(c *cmd.Config, o *Options) cmd.Runnable {
	return cmd.Cmd(
		"markov-eval",
		`[options]`,
		`
Evaluate a fault localization method from ground truth

Option Flags
    -h,--help                         Show this message
    -f,--faults=<path>                Path to a fault file.
    -m,--max=<int>                    Maximum number of states in the chain
    -j,--jump-pr=<float64>            Probability of taking jumps in chains which have them
`,
		"f:m:j:",
		[]string{
			"faults=",
			"max=",
			"jump-pr=",
		},
		func(r cmd.Runnable, args []string, optargs []getopt.OptArg) ([]string, *cmd.Error) {
			max := 1000000
			faultsPath := ""
			jumpPr := (1. / 10.)
			for _, oa := range optargs {
				switch oa.Opt() {
				case "-f", "--faults":
					faultsPath = oa.Arg()
				case "-m", "--max":
					var err error
					max, err = strconv.Atoi(oa.Arg())
					if err != nil {
						return nil, cmd.Errorf(1, "For flag %v expected an int got %v. err: %v", oa.Opt, oa.Arg(), err)
					}
				case "-j", "--jump-pr":
					var err error
					jumpPr, err = strconv.ParseFloat(oa.Arg(), 64)
					if err != nil {
						return nil, cmd.Errorf(1, "For flag %v expected a float got %v. err: %v", oa.Opt, oa.Arg(), err)
					}
					if jumpPr < 0 || jumpPr >= 1 {
						return nil, cmd.Errorf(1, "For flag %v expected a float between 0-1. got %v", oa.Opt, oa.Arg())
					}
				}
			}
			if faultsPath == "" {
				return nil, cmd.Errorf(1, "You must supply the `-f` flag and give a path to the faults")
			}
			faults, err := LoadFaults(faultsPath)
			if err != nil {
				return nil, cmd.Err(1, err)
			}
			fmt.Println("max", max)
			for _, f := range faults {
				fmt.Println(f)
			}
			// if o.Score == nil {
			// 	for name, score := range Scores {
			// 		m := NewMiner(o.Miner, o.Lattice, score, o.Opts...)
			// 		colors, P := DsgMarkovChain(max, m)
			// 		MarkovEval(faults, o.Lattice, "mine-dsg + "+name, colors, P)
			// 		colors, P = RankListMarkovChain(max, m)
			// 		MarkovEval(faults, o.Lattice, name, colors, P)
			// 		colors, P = SpacialJumps(jumpPr, max, m)
			// 		MarkovEval(faults, o.Lattice, "spacial jumps + "+name, colors, P)
			// 		colors, P = BehavioralJumps(jumpPr, max, m)
			// 		MarkovEval(faults, o.Lattice, "behavioral jumps + "+name, colors, P)
			// 		colors, P = BehavioralAndSpacialJumps(jumpPr, max, m)
			// 		MarkovEval(faults, o.Lattice, "behavioral and spacial jumps + "+name, colors, P)
			// 	}
			// } else {
			// 	m := NewMiner(o.Miner, o.Lattice, o.Score, o.Opts...)
			// 	colors, P := DsgMarkovChain(max, m)
			// 	MarkovEval(faults, o.Lattice, "mine-dsg + "+o.ScoreName, colors, P)
			// 	colors, P = RankListMarkovChain(max, m)
			// 	MarkovEval(faults, o.Lattice, o.ScoreName, colors, P)
			// 	colors, P = SpacialJumps(jumpPr, max, m)
			// 	MarkovEval(faults, o.Lattice, "spacial jumps + "+o.ScoreName, colors, P)
			// 	colors, P = BehavioralJumps(jumpPr, max, m)
			// 	MarkovEval(faults, o.Lattice, "behavioral jumps + "+o.ScoreName, colors, P)
			// 	colors, P = BehavioralAndSpacialJumps(jumpPr, max, m)
			// 	MarkovEval(faults, o.Lattice, "behavioral and spacial jumps + "+o.ScoreName, colors, P)
			// }
			return nil, nil
		})
}
