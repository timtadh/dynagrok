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
	parts = append(parts, "Rank, FL Method, Score Name, Eval Method")
	for _, result := range results {
		if result == nil {
			continue
		}
		parts = append(parts,
			fmt.Sprintf("%v, %v, %v, %v",
				result.Rank(),
				result.Method(),
				result.Score(),
				result.Eval(),
			),
		)
	}
	return strings.Join(parts, "\n")
}

func (results EvalResults) Avg() EvalResult {
	if len(results) <= 0 {
		return nil
	}
	method := results[0].Method()
	eval := results[0].Eval()
	score := results[0].Score()
	fault := results[0].Fault()
	location := results[0].Location()
	scoreSum := 0.0
	rankSum := 0.0
	for _, r := range results {
		scoreSum += r.RawScore()
		rankSum += r.Rank()
		if method != r.Method() {
			method = ""
		}
		if score != r.Score() {
			score = ""
		}
		if eval != r.Eval() {
			eval = ""
		}
		if fault == nil || r.Fault() == nil || !fault.Equals(r.Fault()) {
			fault = nil
			location = nil
		}
	}
	return &genericEvalResult{
		method:   method,
		score:    score,
		eval:     eval,
		rank:     rankSum / float64(len(results)),
		rawScore: scoreSum / float64(len(results)),
		fault:    fault,
		location: location,
	}
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

type genericEvalResult struct {
	method   string  // fault localization method: eg. CBSFL, SBBFL, DISCFLO
	score    string  // name of score used: Precision, RF1
	eval     string  // evaluation method used: Ranked List, Markov Chain, Chain + Behavior Jumps, etc...
	rank     float64 // the rank score or equivalent
	rawScore float64 // the raw score given to this location
	fault    *fault.Fault
	location *mine.Location
}

func (r *genericEvalResult) Method() string {
	return r.method
}

func (r *genericEvalResult) Score() string {
	return r.score
}

func (r *genericEvalResult) Eval() string {
	return r.eval
}

func (r *genericEvalResult) Rank() float64 {
	return r.rank
}

func (r *genericEvalResult) RawScore() float64 {
	return r.rawScore
}

func (r *genericEvalResult) Fault() *fault.Fault {
	return r.fault
}

func (r *genericEvalResult) Location() *mine.Location {
	return r.location
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
