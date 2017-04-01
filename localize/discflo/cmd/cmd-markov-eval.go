package cmd

import (
	"fmt"
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
`,
		"f:",
		[]string{
			"faults=",
		},
		func(r cmd.Runnable, args []string, optargs []getopt.OptArg) ([]string, *cmd.Error) {
			faultsPath := ""
			for _, oa := range optargs {
				switch oa.Opt() {
				case "-f", "--faults":
					faultsPath = oa.Arg()
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
					colors, P, err := DiscfloMarkovChain(o, score)
					if err != nil {
						return nil, cmd.Err(1, err)
					}
					mine.MarkovEval(faults, o.Lattice, "discflo + "+name, colors, P)
					m := mine.NewMiner(o.Miner, o.Lattice, score, o.Opts...)
					colors, P = mine.DsgMarkovChain(m)
					mine.MarkovEval(faults, o.Lattice, "mine-dsg + "+name, colors, P)
					colors, P = mine.RankListMarkovChain(m)
					mine.MarkovEval(faults, o.Lattice, name, colors, P)
					colors, P = mine.RankListWithJumpsMarkovChain(m)
					mine.MarkovEval(faults, o.Lattice, "jumps + " + name, colors, P)
				}
			} else {
				colors, P, err := DiscfloMarkovChain(o, o.Score)
				if err != nil {
					return nil, cmd.Err(1, err)
				}
				mine.MarkovEval(faults, o.Lattice, "discflo + "+o.ScoreName, colors, P)
				m := mine.NewMiner(o.Miner, o.Lattice, o.Score, o.Opts...)
				colors, P = mine.DsgMarkovChain(m)
				mine.MarkovEval(faults, o.Lattice, "mine-dsg + "+o.ScoreName, colors, P)
				colors, P = mine.RankListMarkovChain(m)
				mine.MarkovEval(faults, o.Lattice, o.ScoreName, colors, P)
				colors, P = mine.RankListWithJumpsMarkovChain(m)
				mine.MarkovEval(faults, o.Lattice, "jumps + " + o.ScoreName, colors, P)
			}
			return nil, nil
		})
}

func DiscfloMarkovChain(o *discflo.Options, score mine.ScoreFunc) (blockStates map[int][]int, P [][]float64, err error) {
	sum := func(slice []float64) float64 {
		sum := 0.0
		for _, x := range slice {
			sum += x
		}
		return sum
	}
	opts := o.Copy()
	opts.Score = score
	localizer := models.Localize(opts)
	clusters, err := localizer.Clusters()
	if err != nil {
		return nil, nil, err
	}
	groups := clusters.Blocks().Group()
	groupStates := make(map[int]int)
	clusterStates := make(map[int]int)
	blockStates = make(map[int][]int)
	states := 0
	grpState := func(gid int) int {
		if s, has := groupStates[gid]; has {
			return s
		} else {
			state := states
			states++
			groupStates[gid] = state
			return state
		}
	}
	clstrState := func(id int) int {
		if s, has := clusterStates[id]; has {
			return s
		} else {
			state := states
			states++
			clusterStates[id] = state
			return state
		}
	}
	blkState := func(color int) int {
		if s, has := blockStates[color]; has {
			return s[0]
		} else {
			state := states
			states++
			blockStates[color] = append(blockStates[color], state)
			return state
		}
	}
	for gid, group := range groups {
		grpState(gid)
		for _, block := range group {
			blkState(block.Color)
			for _, cluster := range block.In {
				clstrState(cluster.Id)
			}
		}
	}
	P = make([][]float64, 0, states)
	for i := 0; i < states; i++ {
		P = append(P, make([]float64, states))
	}
	for gid, group := range groups {
		gState := grpState(gid)
		for _, block := range group {
			bState := blkState(block.Color)
			P[gState][bState] = 1
			P[bState][gState] = 1
			for _, cluster := range block.In {
				cState := clstrState(cluster.Id)
				P[bState][cState] = 1
				P[cState][bState] = 1
			}
		}
		if gid > 0 {
			prev := grpState(gid - 1)
			P[prev][gState] = 1
			P[gState][prev] = 1
		}
	}
	for _, row := range P {
		total := sum(row)
		if total == 0 {
			continue
		}
		for state := range row {
			row[state] = row[state]/total
		}
	}
	return blockStates, P, nil
}


