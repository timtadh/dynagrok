package discflo

import (
	"github.com/timtadh/dynagrok/localize/test"
	"github.com/timtadh/dynagrok/localize/lattice"
)

type Options struct {
	Lattice   *lattice.Lattice
	Remote    *test.Remote
	Oracle    test.Executor
	Tests     []*test.Testcase
	Score     Score
	ScoreName string
	Walks     int
	Minimize  bool
}

func RunLocalize(o *Options) (Result, error) {
	return RunLocalizeWithScore(o, o.Score)
}

func RunLocalizeWithScore(o *Options, s Score) (Result, error) {
	var tests []*test.Testcase
	if o.Minimize {
		tests = o.Tests
	}
	return Localize(o.Walks, tests, o.Oracle, s, o.Lattice)
}
