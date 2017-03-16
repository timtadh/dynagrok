package mine

import (
	"github.com/timtadh/dynagrok/localize/lattice"
)

type Walk func(*Miner) *SearchNode
type MinerFunc func(*Miner) SearchNodes

type Miner struct {
	Options
	Score    *Score
	Lattice  *lattice.Lattice
	Mine     MinerFunc
}

func NewMiner(lat *lattice.Lattice, sf ScoreFunc, opts ...Option) *Miner {
	m := new(Miner)
	for _, opt := range opts {
		opt(&m.Options)
	}
	if m.MaxEdges == 0 {
		m.MaxEdges = len(lat.Fail.G.E)
	}
	if m.MinEdges == 0 {
		m.MinEdges = 1
	}
	if m.MinFails == 0 {
		m.MinFails = 2
	}
	m.Score = NewScore(sf, &m.Options, lat)
	m.Lattice = lat
	return m
}

