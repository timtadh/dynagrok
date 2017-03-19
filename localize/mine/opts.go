package mine

import (
	"github.com/timtadh/dynagrok/localize/lattice"
	"github.com/timtadh/dynagrok/localize/test"
)

type Options struct {
	ScoreName string
	Score     ScoreFunc
	MinerName string
	Miner     MinerFunc
	Lattice   *lattice.Lattice
	Binary    *test.Remote
	BinArgs   test.Arguments
	Failing   []*test.Testcase
	Passing   []*test.Testcase
	Opts      []MinerOpt
}

func (o *Options) Copy() *Options {
	c := *o
	return &c
}
