package opts

import (
	"github.com/timtadh/dynagrok/localize/lattice"
	"github.com/timtadh/dynagrok/localize/test"
	"github.com/timtadh/dynagrok/localize/discflo/scores"
)

type Options struct {
	Lattice   *lattice.Lattice
	Remote    *test.Remote
	Oracle    test.Executor
	Tests     []*test.Testcase
	Score     scores.Score
	ScoreName string
	Walks     int
	Minimize  bool
}
