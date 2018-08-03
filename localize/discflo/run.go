package discflo

import (
	"github.com/timtadh/dynagrok/localize/mine"
	"github.com/timtadh/dynagrok/localize/mine/opts"
)

type Options struct {
	opts.Options
	DiscfloOpts []DiscfloOption
}

func (o *Options) Copy() *Options {
	c := *o
	return &c
}

func Localizer(o *Options) func(m *mine.Miner) (Clusters, error) {
	return func(m *mine.Miner) (Clusters, error) {
		return Localize(m, o.DiscfloOpts...)
	}
}
