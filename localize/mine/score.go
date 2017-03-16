package mine

import (
	"github.com/timtadh/dynagrok/localize/lattice"
	"github.com/timtadh/dynagrok/localize/stat"
)



type ScoreFunc func(prF, prFandNode, prO, prOandNode float64) float64

type Score struct {
	score ScoreFunc
	opts  *Options
	lat   *lattice.Lattice
}

func NewScore(sf ScoreFunc, opts *Options, lat *lattice.Lattice) *Score {
	return &Score{
		score: sf,
		opts: opts,
		lat: lat,
	}
}

func (s *Score) Score(n *lattice.Node) float64 {
	prF, prFandNode := FailureProbability(s.lat, n)
	prO, prOandNode := OkProbability(s.lat, n)
	return s.score(prF, prFandNode, prO, prOandNode)
}

func (s *Score) Max(n *lattice.Node) float64 {
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
	f := float64(n.Support())
	return O/T, f/T
}

func totalEdgeAndVertexOkPr(lat *lattice.Lattice, n *lattice.Node) (o float64) {
	F := float64(lat.Fail.G.Graphs)
	O := float64(lat.Ok.G.Graphs)
	T := F + O
	for i := range n.SubGraph.E {
		count := lat.Ok.EdgeCounts[n.SubGraph.Colors(i)]
		o += float64(count)/T
	}
	for i := range n.SubGraph.V {
		count := float64(len(lat.Ok.ColorIndex[n.SubGraph.V[i].Color]))
		o += float64(count)/T
	}
	return o
}

// Pr[F=0], Pr[F=0 and sg]
func OkProbability(lat *lattice.Lattice, n *lattice.Node) (prO, prOandNode float64) {
	F := float64(lat.Fail.G.Graphs)
	O := float64(lat.Ok.G.Graphs)
	T := F + O
	size := float64(len(n.SubGraph.V) + len(n.SubGraph.E))
	prOandNode = totalEdgeAndVertexOkPr(lat, n)/size
	return O/T, prOandNode
}

func MinFailureProbability(o *Options, lat *lattice.Lattice) (minPrFandNode float64) {
	F := float64(lat.Fail.G.Graphs)
	O := float64(lat.Ok.G.Graphs)
	T := F + O
	return float64(o.MinFails)/T
}

func MinOkProbability(o *Options, lat *lattice.Lattice, n *lattice.Node) (minPrOandNode float64) {
	largest := float64(2*o.MaxEdges + 1)
	return totalEdgeAndVertexOkPr(lat, n)/largest
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
