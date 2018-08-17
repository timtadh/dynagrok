package eval

import (
	"fmt"
	"math"
	"runtime"
	"strings"

	"github.com/timtadh/dynagrok/localize/discflo"
	"github.com/timtadh/dynagrok/localize/fault"
	"github.com/timtadh/dynagrok/localize/lattice"
	"github.com/timtadh/dynagrok/localize/mine"
	"github.com/timtadh/dynagrok/localize/mine/opts"
)

type FaultIdentifier interface {
	Fault(color int) *fault.Fault
}

type Evaluator struct {
	parallelism       int
	lattice           *lattice.Lattice
	fi                FaultIdentifier
	maxStatesForExact int
	exactHTrank       *bool
}

type EvaluatorOption func(*Evaluator)

func ForceEstimateHTRank(e *Evaluator) {
	e.exactHTrank = new(bool)
	*e.exactHTrank = false
}

func ForceExactHTRank(e *Evaluator) {
	e.exactHTrank = new(bool)
	*e.exactHTrank = true
}

func MaxStatesForExactHTRank(max int) EvaluatorOption {
	return func(e *Evaluator) {
		e.maxStatesForExact = max
	}
}

func Parallelism(p int) EvaluatorOption {
	return func(e *Evaluator) {
		e.parallelism = p
	}
}

func NewEvaluator(lattice *lattice.Lattice, fi FaultIdentifier, opts ...EvaluatorOption) *Evaluator {
	e := &Evaluator{
		lattice:           lattice,
		fi:                fi,
		maxStatesForExact: 1000,
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

func (e *Evaluator) HTRankMethod(states int) string {
	if e.ExactHTRank(states) {
		return "exact"
	}
	return "estimate"
}

func (e *Evaluator) ExactHTRank(states int) bool {
	if e.exactHTrank == nil {
		return states < e.maxStatesForExact
	}
	return *e.exactHTrank
}

func (e *Evaluator) Workers() int {
	cpus := runtime.NumCPU() - 1
	if cpus < 1 {
		cpus = 1
	}
	if e.parallelism == 0 {
		return cpus
	} else if e.parallelism < 0 {
		return 1
	} else if e.parallelism > cpus+1 {
		return cpus + 1
	} else {
		return e.parallelism
	}
}

func (e *Evaluator) Fault(color int) *fault.Fault {
	return e.fi.Fault(color)
}

type DynagrokFaultIdentifier struct {
	faults  []*fault.Fault
	lattice *lattice.Lattice
}

func NewDynagrokFaultIdentifier(lattice *lattice.Lattice, faults []*fault.Fault) *DynagrokFaultIdentifier {
	return &DynagrokFaultIdentifier{
		faults:  faults,
		lattice: lattice,
	}
}

func (d *DynagrokFaultIdentifier) Fault(color int) *fault.Fault {
	bbid, fnName, _ := d.lattice.Info.Get(color)
	for _, f := range d.faults {
		if fnName == f.FnName && bbid == f.BasicBlockId {
			return f
		}
	}
	return nil
}

type Defect4J_FaultIdentifier struct {
	faults  []*fault.Fault
	lattice *lattice.Lattice
}

func NewDefect4J_FaultIdentifier(lattice *lattice.Lattice, faults []*fault.Fault) *Defect4J_FaultIdentifier {
	return &Defect4J_FaultIdentifier{
		faults:  faults,
		lattice: lattice,
	}
}

func (d *Defect4J_FaultIdentifier) Fault(color int) *fault.Fault {
	label := d.lattice.Labels.Label(color)
	label = strings.Replace(strings.Replace(label, ".", "/", -1), "#", ".java:", 1)
	for _, f := range d.faults {
		if label == f.Position {
			fmt.Println(label)
			return f
		}
	}
	return nil
}

type ColorScore struct {
	Color int
	Score float64
}

func CBSFL(o *opts.Options, s mine.ScoreFunc) [][]ColorScore {
	miner := mine.NewMiner(o.Miner, o.Lattice, s, o.Opts...)
	groups := make([][]ColorScore, 0, 10)
	for _, group := range mine.LocalizeNodes(miner.Score).Group() {
		colorGroup := make([]ColorScore, 0, len(group))
		for _, n := range group {
			// fmt.Println(n)
			colorGroup = append(colorGroup, ColorScore{n.Color, n.Score})
		}
		groups = append(groups, colorGroup)
	}
	return groups
}

func Discflo(o *discflo.Options, s mine.ScoreFunc) [][]ColorScore {
	miner := mine.NewMiner(o.Miner, o.Lattice, s, o.Opts...)
	c, err := discflo.Localizer(o)(miner)
	if err != nil {
		panic(err)
	}
	groups := make([][]ColorScore, 0, 10)
	for _, group := range c.RankColors(miner).ScoredLocations().Group() {
		colorGroup := make([]ColorScore, 0, len(group))
		for _, n := range group {
			colorGroup = append(colorGroup, ColorScore{n.Color, n.Score})
		}
		groups = append(groups, colorGroup)
	}
	return groups
}

func (e *Evaluator) RankListEval(methodName, scoreName string, groups [][]ColorScore) (results EvalResults) {
	sum := 0
	var min *RankListEvalResult
	for _, group := range groups {
		for _, cs := range group {
			if f := e.Fault(cs.Color); f != nil {
				label := e.lattice.Labels.Label(cs.Color)
				bbid, fnName, pos := e.lattice.Info.Get(cs.Color)
				r := &RankListEvalResult{
					MethodName:     methodName,
					ScoreName:      scoreName,
					RankScore:      float64(sum) + float64(len(group))/2,
					Suspiciousness: cs.Score,
					LocalizedFault: f,
					Loc: &mine.Location{
						Label:        label,
						Color:        cs.Color,
						BasicBlockId: bbid,
						FnName:       fnName,
						Position:     pos,
					},
				}
				if min == nil || r.RankScore < min.RankScore {
					min = r
				}
			}
		}
		sum += len(group)
	}
	return EvalResults{min}
}

func (e *Evaluator) SBBFLRankListEval(nodes []*mine.SearchNode, methodName, scoreName string) EvalResults {
	min := -1.0
	minScore := -1.0
	var minFault *fault.Fault
	groups := mine.GroupNodesByScore(nodes)
	sum := 0.0
	for _, g := range groups {
		count := 0
		var f *fault.Fault
		for _, n := range g {
			for _, v := range n.Node.SubGraph.V {
				if x := e.Fault(v.Color); x != nil {
					f = x
					count++
					break
				}
			}
		}
		if count > 0 {
			r := float64(len(g) - count)
			b := float64(count)
			score := ((b + r + 1) / (b + 1)) + sum
			if min <= 0 || score < min {
				minFault = f
				min = score
				minScore = g[0].Score
			}
		}
		sum += float64(len(g))
	}
	if min <= 0 {
		min = math.Inf(1)
	}
	r := &RankListEvalResult{
		MethodName:     methodName,
		ScoreName:      scoreName,
		RankScore:      min,
		Suspiciousness: minScore,
		LocalizedFault: minFault,
	}
	return EvalResults{r}
}
