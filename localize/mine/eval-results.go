package mine

import (
	"fmt"
	"strings"
	"sort"
)

type Location struct {
	Color        int
	Position     string
	FnName       string
	BasicBlockId int
}

type ScoredLocation struct {
	Location
	Score float64
}

type ScoredLocations []*ScoredLocation

func (l *Location) String() string {
	return fmt.Sprintf("%v, %v, %v", l.Position, l.FnName, l.BasicBlockId)
}

func (l *ScoredLocation) String() string {
	return fmt.Sprintf("%v, %v", l.Location.String(), l.Score)
}

func (r ScoredLocations) String() string {
	parts := make([]string, 0, len(r))
	for _, l := range r {
		parts = append(parts, l.String())
	}
	return strings.Join(parts, "\n")
}

func (r ScoredLocations) Group() []ScoredLocations {
	r.Sort()
	groups := make([]ScoredLocations, 0, 10)
	for _, n := range r {
		lg := len(groups)
		if lg > 0 && n.Score == groups[lg-1][0].Score {
			groups[lg-1] = append(groups[lg-1], n)
		} else {
			groups = append(groups, make([]*ScoredLocation, 0, 10))
			groups[lg] = append(groups[lg], n)
		}
	}
	return groups
}

func (r ScoredLocations) Sort() {
	sort.SliceStable(r, func(i, j int) bool {
		return r[i].Score > r[j].Score
	})
}

type EvalResults []EvalResult

type EvalResult interface {
	Method() string // fault localization method: eg. CBSFL, SBBFL, DISCFLO
	Score() string // name of score used: Precision, RF1
	Eval() string // evaluation method used: Ranked List, Markov Chain, Chain + Behavior Jumps, etc...
	Rank() float64 // the rank score or equivalent
	RawScore() float64 // the raw score given to this location
	Fault() *Fault
	Location() *Location
}

type MarkovEvalResult struct {
	MethodName  string
	ScoreName   string
	ChainName   string
	HT_Rank     float64
	HittingTime float64
	loc         *Location
	fault       *Fault
}

func (r *MarkovEvalResult) Method() string {
	return r.MethodName
}

func (r *MarkovEvalResult) Score() string {
	return r.ScoreName
}

func (r *MarkovEvalResult) Eval() string {
	return "Markov +" + r.ChainName
}

func (r *MarkovEvalResult) Rank() float64 {
	return r.HT_Rank
}

func (r *MarkovEvalResult) RawScore() float64 {
	return r.HittingTime
}

func (r *MarkovEvalResult) Fault() *Fault {
	return r.fault
}

func (r *MarkovEvalResult) Location() *Location {
	return r.loc
}

type RankListEvalResult struct {
	MethodName  string
	ScoreName   string
	RankScore   float64
	Suspiciousness       float64
	Loc         *Location
	LocalizedFault       *Fault
}

func (r *RankListEvalResult) Method() string {
	return r.MethodName
}

func (r *RankListEvalResult) Score() string {
	return r.ScoreName
}

func (r *RankListEvalResult) Eval() string {
	return "RankList"
}

func (r *RankListEvalResult) Rank() float64 {
	return r.RankScore
}

func (r *RankListEvalResult) RawScore() float64 {
	return r.Suspiciousness
}

func (r *RankListEvalResult) Fault() *Fault {
	return r.LocalizedFault
}

func (r *RankListEvalResult) Location() *Location {
	return r.Loc
}

