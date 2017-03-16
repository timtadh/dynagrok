package discflo

import (
	"github.com/timtadh/dynagrok/localize/test"
	"github.com/timtadh/dynagrok/localize/lattice"
	"github.com/timtadh/dynagrok/localize/mine"
)

type Options struct {
	Lattice   *lattice.Lattice
	Remote    *test.Remote
	Oracle    test.Executor
	Tests     []*test.Testcase
	Score     mine.ScoreFunc
	Miner     mine.MinerFunc
	ScoreName string
	Opts      []mine.Option
	Minimize  bool
}

func Localizer(o *Options) func(m *mine.Miner) (Clusters, error) {
	var tests []*test.Testcase
	if o.Minimize {
		tests = o.Tests
	}
	return func(m *mine.Miner) (Clusters, error) {
		return Localize(m, tests, o.Oracle)
	}
}

