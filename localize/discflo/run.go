package discflo

import (
	"github.com/timtadh/dynagrok/localize/mine"
)

type Options struct {
	mine.Options
	DiscfloOpts []DiscfloOption
}

func Localizer(o *Options) func(m *mine.Miner) (Clusters, error) {
	return func(m *mine.Miner) (Clusters, error) {
		return Localize(m, o.DiscfloOpts...)
	}
}
