package localize

import (
	"fmt"
	"sort"
	"strings"
	"math"
)

type Location struct {
	Position string
	Suspiciousness float64
}

func (l *Location) String() string {
	return fmt.Sprintf("%v --> %v", l.Suspiciousness, l.Position)
}

type Result []Location

func (r Result) String() string {
	parts := make([]string, 0, len(r))
	for _, l := range r {
		parts = append(parts, l.String())
	}
	return strings.Join(parts, "\n")
}

type Method func(fail, ok *Digraph) Result

var Methods = map[string]Method{
	"pr-fail-given-line": prFailGivenLine,
	"pr-line-given-fail": prLineGivenFail,
	"relative-precision": relativePrecision,
	"relative-recall": relativeRecall,
	"precision-gain": precisionGain,
	"jaccard": jaccard,
	"sorensen-dice": sorensenDice,
	"f1": sorensenDice, // from shih-feng's paper f1 is equiv to sd
	"relative-f1": relativeF1,
	"ochiai": ochiai,
	"relative-ochiai": relativeOchiai,
	"symmetric-klosgen": symmetricKlosgen,
	"enhanced-tarantula": enhancedTarantula,
}

func (l *Location) ValidProbability() bool {
	return 0 < l.Suspiciousness && l.Suspiciousness <= 1 && l.Position != ""
}

func (l *Location) ValidRelativeMeasure() bool {
	return -1 <= l.Suspiciousness && l.Suspiciousness <= 1 && l.Position != ""
}

func (r Result) Sort() {
	sort.SliceStable(r, func(i, j int) bool {
		return r[i].Suspiciousness > r[j].Suspiciousness
	})
}

func prFailGivenLine(fail, ok *Digraph) Result {
	lineFails := make(map[int]int)
	lineTotal := make(map[int]int)
	for color, instances := range fail.Indices.ColorIndex {
		lineFails[color] += len(instances)
		lineTotal[color] += len(instances)
	}
	for color, instances := range ok.Indices.ColorIndex {
		lineTotal[color] += len(instances)
	}
	lines := fail.Labels.Labels()
	result := make(Result, 0, len(lines))
	for color := range lines {
		l := Location{
			fail.Positions[color],
			float64(lineFails[color])/float64(lineTotal[color]),
		}
		if l.ValidProbability() {
			result = append(result, l)
		}
	}
	result.Sort()
	return result
}

func prLineGivenFail(fail, ok *Digraph) Result {
	lineFails := make(map[int]int)
	for color, instances := range fail.Indices.ColorIndex {
		lineFails[color] += len(instances)
	}
	lines := fail.Labels.Labels()
	result := make(Result, 0, len(lines))
	for color := range lines {
		l := Location{
			fail.Positions[color],
			float64(lineFails[color])/float64(fail.Graphs),
		}
		if l.ValidProbability() {
			result = append(result, l)
		}
	}
	result.Sort()
	return result
}


func relativePrecision(fail, ok *Digraph) Result {
	lineFails := make(map[int]int)
	lineTotal := make(map[int]int)
	for color, instances := range fail.Indices.ColorIndex {
		lineFails[color] += len(instances)
		lineTotal[color] += len(instances)
	}
	for color, instances := range ok.Indices.ColorIndex {
		lineTotal[color] += len(instances)
	}
	totalTests := fail.Graphs + ok.Graphs
	lines := fail.Labels.Labels()
	result := make(Result, 0, len(lines))
	for color := range lines {
		l := Location{
			fail.Positions[color],
			float64(lineFails[color])/float64(lineTotal[color]) - float64(fail.Graphs)/float64(totalTests),
		}
		if l.ValidRelativeMeasure() {
			result = append(result, l)
		}
	}
	result.Sort()
	return result
}

func relativeRecall(fail, ok *Digraph) Result {
	lineFails := make(map[int]int)
	lineTotal := make(map[int]int)
	for color, instances := range fail.Indices.ColorIndex {
		lineFails[color] += len(instances)
		lineTotal[color] += len(instances)
	}
	for color, instances := range ok.Indices.ColorIndex {
		lineTotal[color] += len(instances)
	}
	totalTests := fail.Graphs + ok.Graphs
	lines := fail.Labels.Labels()
	result := make(Result, 0, len(lines))
	for color := range lines {
		l := Location{
			fail.Positions[color],
			float64(lineFails[color])/float64(fail.Graphs) - float64(lineTotal[color])/float64(totalTests),
		}
		if l.ValidRelativeMeasure() {
			result = append(result, l)
		}
	}
	result.Sort()
	return result
}

func precisionGain(fail, ok *Digraph) Result {
	lineFails := make(map[int]int)
	lineTotal := make(map[int]int)
	for color, instances := range fail.Indices.ColorIndex {
		lineFails[color] += len(instances)
		lineTotal[color] += len(instances)
	}
	for color, instances := range ok.Indices.ColorIndex {
		lineTotal[color] += len(instances)
	}
	totalTests := fail.Graphs + ok.Graphs
	lines := fail.Labels.Labels()
	result := make(Result, 0, len(lines))
	for color := range lines {
		l := Location{
			fail.Positions[color],
			float64(lineTotal[color]) *
				((float64(lineFails[color])/float64(lineTotal[color]) - float64(fail.Graphs)/float64(totalTests)) /
					(float64(fail.Graphs))),
		}
		if l.ValidRelativeMeasure() {
			result = append(result, l)
		}
	}
	result.Sort()
	return result
}

func jaccard(fail, ok *Digraph) Result {
	lineFails := make(map[int]int)
	for color, instances := range fail.Indices.ColorIndex {
		lineFails[color] += len(instances)
	}
	lineOks := make(map[int]int)
	for color, instances := range ok.Indices.ColorIndex {
		lineOks[color] += len(instances)
	}
	lines := fail.Labels.Labels()
	result := make(Result, 0, len(lines))
	for color := range lines {
		l := Location{
			fail.Positions[color],
			float64(lineFails[color])/float64(fail.Graphs + lineOks[color]),
		}
		if l.ValidProbability() {
			result = append(result, l)
		}
	}
	result.Sort()
	return result
}

func sorensenDice(fail, ok *Digraph) Result {
	lineFails := make(map[int]int)
	lineTotal := make(map[int]int)
	for color, instances := range fail.Indices.ColorIndex {
		lineFails[color] += len(instances)
		lineTotal[color] += len(instances)
	}
	for color, instances := range ok.Indices.ColorIndex {
		lineTotal[color] += len(instances)
	}
	lines := fail.Labels.Labels()
	result := make(Result, 0, len(lines))
	for color := range lines {
		l := Location{
			fail.Positions[color],
			2 * float64(lineFails[color])/float64(fail.Graphs + lineTotal[color]),
		}
		if l.ValidProbability() {
			result = append(result, l)
		}
	}
	result.Sort()
	return result
}

func relativeF1(fail, ok *Digraph) Result {
	lineFails := make(map[int]int)
	lineTotal := make(map[int]int)
	for color, instances := range fail.Indices.ColorIndex {
		lineFails[color] += len(instances)
		lineTotal[color] += len(instances)
	}
	for color, instances := range ok.Indices.ColorIndex {
		lineTotal[color] += len(instances)
	}
	totalTests := fail.Graphs + ok.Graphs
	lines := fail.Labels.Labels()
	result := make(Result, 0, len(lines))
	for color := range lines {
		l := Location{
			fail.Positions[color],
			2 *
				float64(lineTotal[color])/float64(fail.Graphs + lineTotal[color]) *
				(float64(lineFails[color])/float64(lineTotal[color]) - float64(fail.Graphs)/float64(totalTests)),
		}
		if l.ValidRelativeMeasure() {
			result = append(result, l)
		}
	}
	result.Sort()
	return result
}

func ochiai(fail, ok *Digraph) Result {
	lineFails := make(map[int]int)
	lineTotal := make(map[int]int)
	for color, instances := range fail.Indices.ColorIndex {
		lineFails[color] += len(instances)
		lineTotal[color] += len(instances)
	}
	for color, instances := range ok.Indices.ColorIndex {
		lineTotal[color] += len(instances)
	}
	lines := fail.Labels.Labels()
	result := make(Result, 0, len(lines))
	for color := range lines {
		l := Location{
			fail.Positions[color],
			math.Sqrt(float64(lineTotal[color])/float64(fail.Graphs)) * float64(lineFails[color])/float64(lineTotal[color]),
		}
		if l.ValidProbability() {
			result = append(result, l)
		}
	}
	result.Sort()
	return result
}

func relativeOchiai(fail, ok *Digraph) Result {
	lineFails := make(map[int]int)
	lineTotal := make(map[int]int)
	for color, instances := range fail.Indices.ColorIndex {
		lineFails[color] += len(instances)
		lineTotal[color] += len(instances)
	}
	for color, instances := range ok.Indices.ColorIndex {
		lineTotal[color] += len(instances)
	}
	totalTests := fail.Graphs + ok.Graphs
	lines := fail.Labels.Labels()
	result := make(Result, 0, len(lines))
	for color := range lines {
		l := Location{
			fail.Positions[color],
			math.Sqrt(float64(lineTotal[color])/float64(fail.Graphs)) *
				(float64(lineFails[color])/float64(lineTotal[color]) - float64(fail.Graphs)/float64(totalTests)),
		}
		if l.ValidRelativeMeasure() {
			result = append(result, l)
		}
	}
	result.Sort()
	return result
}

func symmetricKlosgen(fail, ok *Digraph) Result {
	max := func(a, b float64) float64 {
		if a > b {
			return a
		}
		return b
	}
	lineFails := make(map[int]int)
	lineTotal := make(map[int]int)
	for color, instances := range fail.Indices.ColorIndex {
		lineFails[color] += len(instances)
		lineTotal[color] += len(instances)
	}
	for color, instances := range ok.Indices.ColorIndex {
		lineTotal[color] += len(instances)
	}
	totalTests := fail.Graphs + ok.Graphs
	lines := fail.Labels.Labels()
	result := make(Result, 0, len(lines))
	for color := range lines {
		l := Location{
			fail.Positions[color],
			math.Sqrt(float64(lineFails[color])/float64(totalTests)) *
				max(float64(lineFails[color])/float64(lineTotal[color]) - float64(fail.Graphs)/float64(totalTests),
					float64(lineFails[color])/float64(fail.Graphs) - float64(lineTotal[color])/float64(totalTests)),
		}
		if l.ValidRelativeMeasure() {
			result = append(result, l)
		}
	}
	result.Sort()
	return result
}

func enhancedTarantula(fail, ok *Digraph) Result {
	lineFails := make(map[int]int)
	lineTotal := make(map[int]int)
	for color, instances := range fail.Indices.ColorIndex {
		lineFails[color] += len(instances)
		lineTotal[color] += len(instances)
	}
	for color, instances := range ok.Indices.ColorIndex {
		lineTotal[color] += len(instances)
	}
	totalTests := fail.Graphs + ok.Graphs
	lines := fail.Labels.Labels()
	result := make(Result, 0, len(lines))
	for color := range lines {
		l := Location{
			fail.Positions[color],
			float64(lineFails[color])/float64(fail.Graphs) *
				(float64(lineFails[color])/float64(lineTotal[color]) - float64(fail.Graphs)/float64(totalTests)),
		}
		if l.ValidRelativeMeasure() {
			result = append(result, l)
		}
	}
	result.Sort()
	return result
}
