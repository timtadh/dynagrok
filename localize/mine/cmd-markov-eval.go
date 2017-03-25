package mine

import (
	"fmt"
)

import (
	"github.com/timtadh/getopt"
	"github.com/timtadh/matrix"
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
			faults, err := LoadFaults(faultsPath)
			if err != nil {
				return nil, cmd.Err(1, err)
			}
			for _, f := range faults {
				fmt.Println(f)
			}
			eval := func(name string, m *Miner) {
				colors, P := MarkovChain(m)
				scores := make(map[int]float64)
				for color, state := range colors {
					scores[color] = ExpectedHittingTime(0, state, P)
				}
				for color, score := range scores {
					b, fn, pos := o.Lattice.Info.Get(color)
					fmt.Printf(
						"    {\n\thitting time: %v,\n\tfn: %v (%d),\n\tpos: %v\n    }\n",
						score,
						fn, b, pos,
					)
				}
				for _, f := range faults {
					for color, score := range scores {
						b, fn, pos := o.Lattice.Info.Get(color)
						if fn == f.FnName && b == f.BasicBlockId {
							fmt.Printf(
								"    %v {\n\thitting time: %v,\n\tfn: %v (%d),\n\tpos: %v\n    }\n",
								name,
								score,
								fn, b, pos,
							)
							break
						}
					}
				}
			}
			if o.Score == nil {
				for name, score := range Scores {
					m := NewMiner(o.Miner, o.Lattice, score, o.Opts...)
					eval("mine-dsg + "+name, m)
				}
			} else {
				m := NewMiner(o.Miner, o.Lattice, o.Score, o.Opts...)
				eval("mine-dsg + "+o.ScoreName, m)
			}
			return nil, nil
		})
}

func MarkovChain(m *Miner) (blockStates map[int]int, P [][]float64) {
	sum := func(slice []float64) float64 {
		sum := 0.0
		for _, x := range slice {
			sum += x
		}
		return sum
	}
	groups := m.Mine().group()
	groupStates := make(map[int]int)
	graphStates := make(map[int]int)
	blockStates = make(map[int]int)
	states := 0
	for gid, group := range groups {
		groupStates[gid] = states
		states++
		for nid, n := range group {
			graphStates[nid] = states
			states++
			for _, v := range n.Node.SubGraph.V {
				if _, has := blockStates[v.Color]; !has {
					blockStates[v.Color] = states
					states++
				}
			}
		}
	}
	P = make([][]float64, 0, states)
	for i := 0; i < states; i++ {
		P = append(P, make([]float64, states))
	}
	for gid, group := range groups {
		groupState := groupStates[gid]
		for nid, n := range group {
			graphState := graphStates[nid]
			P[groupState][graphState] = 1
			P[graphState][groupState] = 1
			for _, v := range n.Node.SubGraph.V {
				blockState := blockStates[v.Color]
				P[graphState][blockState] = 1
				P[blockState][graphState] = 1
			}
		}
		if gid > 0 {
			prev := groupStates[gid - 1]
			P[prev][groupState] = 1
			P[groupState][prev] = 1
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
	return blockStates, P
}

func ExpectedHittingTimes(transitions [][]float64) [][]float64 {
	P := matrix.MakeDenseMatrixStacked(transitions)
	M := matrix.Zeros(P.Rows(), P.Cols())
	for i := 0; i < P.Rows()*P.Rows(); i++ {
		prevM := M.Copy()
		for t := 0; t < M.Rows(); t++ {
			for s := 0; s < M.Cols(); s++ {
				if t == s {
					M.Set(t, s, 0.0)
				} else {
					sum := 0.0
					for k := 0; k < P.Rows(); k++ {
						if k != s {
							sum += P.Get(t, k) * (M.Get(k, s) + 1)
						}
					}
					M.Set(t, s, P.Get(t, s) + sum)
				}
			}
		}
		diff, err := M.Minus(prevM)
		if err != nil {
			panic(err)
		}
		if diff.DenseMatrix().TwoNorm() < .01 {
			break
		}
	}
	return M.Arrays()
}

func ExpectedHittingTime(x, y int, transitions [][]float64) float64 {
	P := matrix.MakeDenseMatrixStacked(transitions)
	for s := 0; s < P.Cols(); s++ {
		P.Set(y, s, 0)
	}
	P.Set(y, y, 1)
	last := P.Rows()
	P.SwapRows(y, last-1)
	P = P.Transpose()
	P.SwapRows(y, last-1)
	P = P.Transpose()
	Q := P.GetMatrix(0, 0, last-1, last-1)
	I := matrix.Eye(Q.Rows())
	c := matrix.Ones(Q.Rows(), 1)
	IQ, err := I.Minus(Q)
	if err != nil {
		panic(err)
	}
	N, err := IQ.DenseMatrix().Inverse()
	if err != nil {
		panic(err)
	}
	Nc, err := N.Times(c)
	if err != nil {
		panic(err)
	}
	fmt.Println(x, y, Nc.Get(x,0))
	return Nc.Get(x, 0)
}
