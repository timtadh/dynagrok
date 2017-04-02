package cmd

import (
	"fmt"
	"strconv"
)

import (
	"github.com/timtadh/getopt"
)

import (
	"github.com/timtadh/dynagrok/cmd"
	"github.com/timtadh/dynagrok/localize/discflo"
	"github.com/timtadh/dynagrok/localize/discflo/web/models"
	"github.com/timtadh/dynagrok/localize/mine"
)


func NewMarkovEvalParser(c *cmd.Config, o *discflo.Options) cmd.Runnable {
	return cmd.Cmd(
		"markov-eval",
		`[options]`,
		`
Evaluate a fault localization method from ground truth

Option Flags
    -h,--help                         Show this message
    -f,--faults=<path>                Path to a fault file.
    -m,--max=<int>                    Maximum number of states in the chain
`,
		"f:m:",
		[]string{
			"faults=",
			"max=",
		},
		func(r cmd.Runnable, args []string, optargs []getopt.OptArg) ([]string, *cmd.Error) {
			max := 1000000
			faultsPath := ""
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
				}
			}
			if faultsPath == "" {
				return nil, cmd.Errorf(1, "You must supply the `-f` flag and give a path to the faults")
			}
			faults, err := mine.LoadFaults(faultsPath)
			if err != nil {
				return nil, cmd.Err(1, err)
			}
			for _, f := range faults {
				fmt.Println(f)
			}
			if o.Score == nil {
				for name, score := range mine.Scores {
					colors, P, err := DiscfloMarkovChain(max, o, score)
					if err != nil {
						return nil, cmd.Err(1, err)
					}
					mine.MarkovEval(faults, o.Lattice, "discflo + "+name, colors, P)
					m := mine.NewMiner(o.Miner, o.Lattice, score, o.Opts...)
					colors, P = mine.DsgMarkovChain(max, m)
					mine.MarkovEval(faults, o.Lattice, "mine-dsg + "+name, colors, P)
					colors, P = mine.RankListMarkovChain(max, m)
					mine.MarkovEval(faults, o.Lattice, name, colors, P)
					colors, P = mine.BehaviorJumps(max, m)
					mine.MarkovEval(faults, o.Lattice, "behavioral jumps + " + name, colors, P)
					colors, P = mine.SpacialJumps(max, m)
					mine.MarkovEval(faults, o.Lattice, "spacial jumps + " + name, colors, P)
				}
			} else {
				colors, P, err := DiscfloMarkovChain(max, o, o.Score)
				if err != nil {
					return nil, cmd.Err(1, err)
				}
				mine.MarkovEval(faults, o.Lattice, "discflo + "+o.ScoreName, colors, P)
				m := mine.NewMiner(o.Miner, o.Lattice, o.Score, o.Opts...)
				colors, P = mine.DsgMarkovChain(max, m)
				mine.MarkovEval(faults, o.Lattice, "mine-dsg + "+o.ScoreName, colors, P)
				colors, P = mine.RankListMarkovChain(max, m)
				mine.MarkovEval(faults, o.Lattice, o.ScoreName, colors, P)
				colors, P = mine.BehaviorJumps(max, m)
				mine.MarkovEval(faults, o.Lattice, "behavioral jumps + " + o.ScoreName, colors, P)
				colors, P = mine.SpacialJumps(max, m)
				mine.MarkovEval(faults, o.Lattice, "spacial jumps + " + o.ScoreName, colors, P)
			}
			return nil, nil
		})
}

func DiscfloMarkovChain(max int, o *discflo.Options, score mine.ScoreFunc) (blockStates map[int][]int, P [][]float64, err error) {
	prJump := 1./10.
	opts := o.Copy()
	opts.Score = score
	localizer := models.Localize(opts)
	clusters, err := localizer.Clusters()
	if err != nil {
		return nil, nil, err
	}
	groups := clusters.Blocks().Group()
	neighbors := make(map[int]map[int]bool)
	colors := make([][]int, 0, len(groups))
	for _, group := range groups {
		colorGroup := make([]int, 0, len(group))
		for _, block := range group {
			colorGroup = append(colorGroup, block.Color)
			neighbors[block.Color] = make(map[int]bool)
			for _, cluster := range block.In {
				for _, n := range cluster.Nodes {
					for _, v := range n.Node.SubGraph.V {
						neighbors[block.Color][v.Color] = true
					}
				}
			}
		}
		colors = append(colors, colorGroup)
	}
	blockStates, P = mine.RankListWithJumpsMarkovChain(max, colors, prJump, neighbors)
	return blockStates, P, nil
}


