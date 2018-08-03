package opts

import (
	"github.com/timtadh/dynagrok/localize/lattice"
	"github.com/timtadh/dynagrok/localize/mine"
	"github.com/timtadh/dynagrok/localize/test"
)

type Options struct {
	ScoreName       string
	Score           mine.ScoreFunc
	MinerName       string
	Miner           mine.MinerFunc
	Lattice         *lattice.Lattice
	Binary          *test.Remote
	BinArgs         test.Arguments
	Failing         []*test.Testcase
	Passing         []*test.Testcase
	PassingProfiles []string
	FailingProfiles []string
	Opts            []mine.MinerOpt
}

func (o *Options) Copy() *Options {
	c := *o
	return &c
}
