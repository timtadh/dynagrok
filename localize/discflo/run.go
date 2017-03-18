package discflo

import (
	"github.com/timtadh/dynagrok/localize/mine"
	"github.com/timtadh/dynagrok/localize/test"
)

type Options struct {
	mine.Options
	Oracle    test.Executor
	Minimize  bool
}

func Localizer(o *Options) func(m *mine.Miner) (Clusters, error) {
	var tests []*test.Testcase
	if o.Minimize {
		tests = o.Failing
	}
	return func(m *mine.Miner) (Clusters, error) {
		return Localize(m, tests, o.Oracle)
	}
}
