package mine

import (
	"context"
	"fmt"

	"github.com/timtadh/dynagrok/localize/lattice"
)

type Walk func(*Miner) *SearchNode
type MinerFunc func(context.Context, *Miner) SearchNodes

type Miner struct {
	MinerConfig
	Score   *Score
	Lattice *lattice.Lattice
	Miner   MinerFunc
}

func NewMiner(mf MinerFunc, lat *lattice.Lattice, sf ScoreFunc, opts ...MinerOpt) *Miner {
	m := new(Miner)
	for _, opt := range opts {
		opt(&m.MinerConfig)
	}
	if m.MaxEdges == 0 {
		m.MaxEdges = len(lat.Fail.G.E)
	}
	if m.MinFails == 0 {
		m.MinFails = 2
	}
	m.Score = NewScore(sf, &m.MinerConfig, lat)
	m.Lattice = lat
	m.Miner = mf
	return m
}

func (m *Miner) Mine(ctx context.Context) SearchNodes {
	return m.Miner(ctx, m)
}

type MinerConfig struct {
	MaxEdges int
	MinEdges int
	MinFails int
}

type MinerOpt func(*MinerConfig)

func MinEdges(minEdges int) MinerOpt {
	if minEdges < 0 {
		panic(fmt.Errorf("minEdges must be >= 0 (got %v)", minEdges))
	}
	return func(o *MinerConfig) {
		o.MinEdges = minEdges
	}
}

func MaxEdges(maxEdges int) MinerOpt {
	if maxEdges < 2 {
		panic(fmt.Errorf("MaxEdges must be >= 2 (got %v)", maxEdges))
	}
	return func(o *MinerConfig) {
		o.MaxEdges = maxEdges
	}
}

func MinFails(minFails int) MinerOpt {
	if minFails < 1 {
		panic(fmt.Errorf("minFails must be >= 1 (got %v)", minFails))
	}
	return func(o *MinerConfig) {
		o.MinFails = minFails
	}
}
