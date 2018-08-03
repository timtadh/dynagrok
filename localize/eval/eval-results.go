package eval

import (
	"fmt"
	"strings"

	"github.com/timtadh/dynagrok/localize/fault"
	"github.com/timtadh/dynagrok/localize/mine"
)

type EvalResults []EvalResult

func (results EvalResults) String() string {
	parts := make([]string, 0, len(results))
	parts = append(parts, "Rank, FL Method, Score, Eval Method, Raw Score, Location")
	for _, result := range results {
		parts = append(parts,
			fmt.Sprintf("%v, %v, %v, %v, %v, %v",
				result.Rank(),
				result.Method(),
				result.Score(),
				result.Eval(),
				result.RawScore(),
				result.Location(),
			),
		)
	}
	return strings.Join(parts, "\n")
}

type EvalResult interface {
	Method() string    // fault localization method: eg. CBSFL, SBBFL, DISCFLO
	Score() string     // name of score used: Precision, RF1
	Eval() string      // evaluation method used: Ranked List, Markov Chain, Chain + Behavior Jumps, etc...
	Rank() float64     // the rank score or equivalent
	RawScore() float64 // the raw score given to this location
	Fault() *fault.Fault
	Location() *mine.Location
}

type MarkovEvalResult struct {
	MethodName  string
	ScoreName   string
	ChainName   string
	HT_Rank     float64
	HittingTime float64
	loc         *mine.Location
	fault       *fault.Fault
}

func (r *MarkovEvalResult) Method() string {
	return r.MethodName
}

func (r *MarkovEvalResult) Score() string {
	return r.ScoreName
}

func (r *MarkovEvalResult) Eval() string {
	return "Markov + " + r.ChainName
}

func (r *MarkovEvalResult) Rank() float64 {
	return r.HT_Rank
}

func (r *MarkovEvalResult) RawScore() float64 {
	return r.HittingTime
}

func (r *MarkovEvalResult) Fault() *fault.Fault {
	return r.fault
}

func (r *MarkovEvalResult) Location() *mine.Location {
	return r.loc
}

type RankListEvalResult struct {
	MethodName     string
	ScoreName      string
	RankScore      float64
	Suspiciousness float64
	Loc            *mine.Location
	LocalizedFault *fault.Fault
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

func (r *RankListEvalResult) Fault() *fault.Fault {
	return r.LocalizedFault
}

func (r *RankListEvalResult) Location() *mine.Location {
	return r.Loc
}
