package mine

import (
	"github.com/timtadh/data-structures/errors"
)

import (
	"github.com/timtadh/dynagrok/localize/lattice"
	"github.com/timtadh/dynagrok/localize/stat"
)

type ScoreFunc func(prF, prFandNode, prO, prOandNode float64) float64

type Score struct {
	score ScoreFunc
	opts  *MinerConfig
	lat   *lattice.Lattice
}

func NewScore(sf ScoreFunc, opts *MinerConfig, lat *lattice.Lattice) *Score {
	return &Score{
		score: sf,
		opts:  opts,
		lat:   lat,
	}
}

func (s *Score) Score(n *lattice.Node) float64 {
	prF, prFandNode := FailureProbability(s.lat, n)
	prO, prOandNode := OkProbability(s.lat, n)
	score := s.score(prF, prFandNode, prO, prOandNode)
	if false {
		errors.Logf("DEBUG", "score %v (%v %v) (%v %v) %v", score, prF, prFandNode, prO, prOandNode, n)
	}
	return score
}

func (s *Score) Max(n *lattice.Node) float64 {
	if n == nil || n.SubGraph == nil {
		F := float64(s.lat.Fail.G.Graphs)
		O := float64(s.lat.Ok.G.Graphs)
		prF := F/(F+O)
		prO := O/(F+O)
		x := s.score(prF, 0, prO, 1)
		y := s.score(prF, 1, prO, 0)
		return max(x, y)
	}
	prF, prFandNode := FailureProbability(s.lat, n)
	prO, prOandNode := OkProbability(s.lat, n)
	minPrFandNode := MinFailureProbability(s.opts, s.lat)
	minPrOandNode := MinOkProbability(s.opts, s.lat, n)
	x := s.score(prF, minPrFandNode, prO, prOandNode)
	y := s.score(prF, prFandNode, prO, minPrOandNode)
	return max(x, y)
}

func max(x, y float64) float64 {
	if x > y {
		return x
	}
	return y
}

// Pr[F=1], Pr[F=1 and sg]
func FailureProbability(lat *lattice.Lattice, n *lattice.Node) (prF, prFandNode float64) {
	F := float64(lat.Fail.G.Graphs)
	O := float64(lat.Ok.G.Graphs)
	T := F + O
	f := float64(n.FIS())
	return O / T, f / T
}

func totalEdgeAndVertexOkPr(lat *lattice.Lattice, n *lattice.Node) (o float64) {
	F := float64(lat.Fail.G.Graphs)
	O := float64(lat.Ok.G.Graphs)
	T := F + O
	for i := range n.SubGraph.E {
		count := lat.Ok.EdgeCounts[n.SubGraph.Colors(i)]
		o += float64(count) / T
	}
	for i := range n.SubGraph.V {
		count := float64(len(lat.Ok.ColorIndex[n.SubGraph.V[i].Color]))
		o += float64(count) / T
	}
	return o
}

// Pr[F=0], Pr[F=0 and sg]
func OkProbability(lat *lattice.Lattice, n *lattice.Node) (prO, prOandNode float64) {
	F := float64(lat.Fail.G.Graphs)
	O := float64(lat.Ok.G.Graphs)
	T := F + O
	size := float64(len(n.SubGraph.V) + len(n.SubGraph.E))
	if len(n.SubGraph.E) > 0 || len(n.SubGraph.V) >= 1 {
		prOandNode = totalEdgeAndVertexOkPr(lat, n) / size
	} else {
		prOandNode = O / T
	}
	return O / T, prOandNode
}

func MinFailureProbability(o *MinerConfig, lat *lattice.Lattice) (minPrFandNode float64) {
	F := float64(lat.Fail.G.Graphs)
	O := float64(lat.Ok.G.Graphs)
	T := F + O
	return float64(o.MinFails) / T
}

func MinOkProbability(o *MinerConfig, lat *lattice.Lattice, n *lattice.Node) (minPrOandNode float64) {
	largest := float64(2*o.MaxEdges + 1)
	return totalEdgeAndVertexOkPr(lat, n) / largest
}

func LocalizeNodes(score *Score) stat.Result {
	lat := score.lat
	result := make(stat.Result, 0, len(lat.Fail.ColorIndex))
	for color, _ := range lat.Fail.ColorIndex {
		n := ColorNode(lat, score, color)
		bbid, fnName, pos := lat.Info.Get(color)
		result = append(result, stat.Location{
			color,
			pos,
			fnName,
			bbid,
			n.Score,
		})
	}
	result.Sort()
	return result
}
