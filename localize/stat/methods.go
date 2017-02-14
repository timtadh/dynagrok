package stat

import (
	"fmt"
	"sort"
	"strings"
	"math"
)

import (
	"github.com/timtadh/dynagrok/localize/lattice"
)

type Location struct {
	Position string
	FnName   string
	BasicBlockId int
	Score float64
}

func (l *Location) String() string {
	return fmt.Sprintf("%v, %v, %v, %v", l.Position, l.FnName, l.BasicBlockId, l.Score)
}

type Result []Location

func (r Result) String() string {
	parts := make([]string, 0, len(r))
	for _, l := range r {
		parts = append(parts, l.String())
	}
	return strings.Join(parts, "\n")
}

type Method func(lat *lattice.Lattice) Result

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
	return 0 < l.Score && l.Score <= 1 && l.Position != ""
}

func (l *Location) ValidRelativeMeasure() bool {
	return -1 <= l.Score && l.Score <= 1 && l.Position != ""
}

func (r Result) Sort() {
	sort.SliceStable(r, func(i, j int) bool {
		return r[i].Score > r[j].Score
	})
}

func prFailGivenLine(lat *lattice.Lattice) Result {
	lineFails := make(map[int]int)
	lineTotal := make(map[int]int)
	for color, instances := range lat.Fail.ColorIndex {
		lineFails[color] += len(instances)
		lineTotal[color] += len(instances)
	}
	for color, instances := range lat.Ok.ColorIndex {
		lineTotal[color] += len(instances)
	}
	lines := lat.Labels.Labels()
	result := make(Result, 0, len(lines))
	for color := range lines {
		l := Location{
			lat.Positions[color],
			lat.FnNames[color],
			lat.BBIds[color],
			float64(lineFails[color])/float64(lineTotal[color]),
		}
		if l.ValidProbability() {
			result = append(result, l)
		}
	}
	result.Sort()
	return result
}

func prLineGivenFail(lat *lattice.Lattice) Result {
	lineFails := make(map[int]int)
	for color, instances := range lat.Fail.ColorIndex {
		lineFails[color] += len(instances)
	}
	lines := lat.Labels.Labels()
	result := make(Result, 0, len(lines))
	for color := range lines {
		l := Location{
			lat.Positions[color],
			lat.FnNames[color],
			lat.BBIds[color],
			float64(lineFails[color])/float64(lat.Fail.G.Graphs),
		}
		if l.ValidProbability() {
			result = append(result, l)
		}
	}
	result.Sort()
	return result
}


func relativePrecision(lat *lattice.Lattice) Result {
	lineFails := make(map[int]int)
	lineTotal := make(map[int]int)
	for color, instances := range lat.Fail.ColorIndex {
		lineFails[color] += len(instances)
		lineTotal[color] += len(instances)
	}
	for color, instances := range lat.Ok.ColorIndex {
		lineTotal[color] += len(instances)
	}
	totalTests := lat.Fail.G.Graphs + lat.Ok.G.Graphs
	lines := lat.Labels.Labels()
	result := make(Result, 0, len(lines))
	for color := range lines {
		l := Location{
			lat.Positions[color],
			lat.FnNames[color],
			lat.BBIds[color],
			float64(lineFails[color])/float64(lineTotal[color]) - float64(lat.Fail.G.Graphs)/float64(totalTests),
		}
		if l.ValidRelativeMeasure() {
			result = append(result, l)
		}
	}
	result.Sort()
	return result
}

func relativeRecall(lat *lattice.Lattice) Result {
	lineFails := make(map[int]int)
	lineTotal := make(map[int]int)
	for color, instances := range lat.Fail.ColorIndex {
		lineFails[color] += len(instances)
		lineTotal[color] += len(instances)
	}
	for color, instances := range lat.Ok.ColorIndex {
		lineTotal[color] += len(instances)
	}
	totalTests := lat.Fail.G.Graphs + lat.Ok.G.Graphs
	lines := lat.Labels.Labels()
	result := make(Result, 0, len(lines))
	for color := range lines {
		l := Location{
			lat.Positions[color],
			lat.FnNames[color],
			lat.BBIds[color],
			float64(lineFails[color])/float64(lat.Fail.G.Graphs) - float64(lineTotal[color])/float64(totalTests),
		}
		if l.ValidRelativeMeasure() {
			result = append(result, l)
		}
	}
	result.Sort()
	return result
}

func precisionGain(lat *lattice.Lattice) Result {
	lineFails := make(map[int]int)
	lineTotal := make(map[int]int)
	for color, instances := range lat.Fail.ColorIndex {
		lineFails[color] += len(instances)
		lineTotal[color] += len(instances)
	}
	for color, instances := range lat.Ok.ColorIndex {
		lineTotal[color] += len(instances)
	}
	totalTests := lat.Fail.G.Graphs + lat.Ok.G.Graphs
	lines := lat.Labels.Labels()
	result := make(Result, 0, len(lines))
	for color := range lines {
		l := Location{
			lat.Positions[color],
			lat.FnNames[color],
			lat.BBIds[color],
			float64(lineTotal[color]) *
				((float64(lineFails[color])/float64(lineTotal[color]) - float64(lat.Fail.G.Graphs)/float64(totalTests)) /
					(float64(lat.Fail.G.Graphs))),
		}
		if l.ValidRelativeMeasure() {
			result = append(result, l)
		}
	}
	result.Sort()
	return result
}

func jaccard(lat *lattice.Lattice) Result {
	lineFails := make(map[int]int)
	for color, instances := range lat.Fail.ColorIndex {
		lineFails[color] += len(instances)
	}
	lineOks := make(map[int]int)
	for color, instances := range lat.Ok.ColorIndex {
		lineOks[color] += len(instances)
	}
	lines := lat.Labels.Labels()
	result := make(Result, 0, len(lines))
	for color := range lines {
		l := Location{
			lat.Positions[color],
			lat.FnNames[color],
			lat.BBIds[color],
			float64(lineFails[color])/float64(lat.Fail.G.Graphs + lineOks[color]),
		}
		if l.ValidProbability() {
			result = append(result, l)
		}
	}
	result.Sort()
	return result
}

func sorensenDice(lat *lattice.Lattice) Result {
	lineFails := make(map[int]int)
	lineTotal := make(map[int]int)
	for color, instances := range lat.Fail.ColorIndex {
		lineFails[color] += len(instances)
		lineTotal[color] += len(instances)
	}
	for color, instances := range lat.Ok.ColorIndex {
		lineTotal[color] += len(instances)
	}
	lines := lat.Labels.Labels()
	result := make(Result, 0, len(lines))
	for color := range lines {
		l := Location{
			lat.Positions[color],
			lat.FnNames[color],
			lat.BBIds[color],
			2 * float64(lineFails[color])/float64(lat.Fail.G.Graphs + lineTotal[color]),
		}
		if l.ValidProbability() {
			result = append(result, l)
		}
	}
	result.Sort()
	return result
}

func relativeF1(lat *lattice.Lattice) Result {
	lineFails := make(map[int]int)
	lineTotal := make(map[int]int)
	for color, instances := range lat.Fail.ColorIndex {
		lineFails[color] += len(instances)
		lineTotal[color] += len(instances)
	}
	for color, instances := range lat.Ok.ColorIndex {
		lineTotal[color] += len(instances)
	}
	totalTests := lat.Fail.G.Graphs + lat.Ok.G.Graphs
	lines := lat.Labels.Labels()
	result := make(Result, 0, len(lines))
	for color := range lines {
		l := Location{
			lat.Positions[color],
			lat.FnNames[color],
			lat.BBIds[color],
			2 *
				float64(lineTotal[color])/float64(lat.Fail.G.Graphs + lineTotal[color]) *
				(float64(lineFails[color])/float64(lineTotal[color]) - float64(lat.Fail.G.Graphs)/float64(totalTests)),
		}
		if l.ValidRelativeMeasure() {
			result = append(result, l)
		}
	}
	result.Sort()
	return result
}

func ochiai(lat *lattice.Lattice) Result {
	lineFails := make(map[int]int)
	lineTotal := make(map[int]int)
	for color, instances := range lat.Fail.ColorIndex {
		lineFails[color] += len(instances)
		lineTotal[color] += len(instances)
	}
	for color, instances := range lat.Ok.ColorIndex {
		lineTotal[color] += len(instances)
	}
	lines := lat.Labels.Labels()
	result := make(Result, 0, len(lines))
	for color := range lines {
		l := Location{
			lat.Positions[color],
			lat.FnNames[color],
			lat.BBIds[color],
			math.Sqrt(float64(lineTotal[color])/float64(lat.Fail.G.Graphs)) * float64(lineFails[color])/float64(lineTotal[color]),
		}
		if l.ValidProbability() {
			result = append(result, l)
		}
	}
	result.Sort()
	return result
}

func relativeOchiai(lat *lattice.Lattice) Result {
	lineFails := make(map[int]int)
	lineTotal := make(map[int]int)
	for color, instances := range lat.Fail.ColorIndex {
		lineFails[color] += len(instances)
		lineTotal[color] += len(instances)
	}
	for color, instances := range lat.Ok.ColorIndex {
		lineTotal[color] += len(instances)
	}
	totalTests := lat.Fail.G.Graphs + lat.Ok.G.Graphs
	lines := lat.Labels.Labels()
	result := make(Result, 0, len(lines))
	for color := range lines {
		l := Location{
			lat.Positions[color],
			lat.FnNames[color],
			lat.BBIds[color],
			math.Sqrt(float64(lineTotal[color])/float64(lat.Fail.G.Graphs)) *
				(float64(lineFails[color])/float64(lineTotal[color]) - float64(lat.Fail.G.Graphs)/float64(totalTests)),
		}
		if l.ValidRelativeMeasure() {
			result = append(result, l)
		}
	}
	result.Sort()
	return result
}

func symmetricKlosgen(lat *lattice.Lattice) Result {
	max := func(a, b float64) float64 {
		if a > b {
			return a
		}
		return b
	}
	lineFails := make(map[int]int)
	lineTotal := make(map[int]int)
	for color, instances := range lat.Fail.ColorIndex {
		lineFails[color] += len(instances)
		lineTotal[color] += len(instances)
	}
	for color, instances := range lat.Ok.ColorIndex {
		lineTotal[color] += len(instances)
	}
	totalTests := lat.Fail.G.Graphs + lat.Ok.G.Graphs
	lines := lat.Labels.Labels()
	result := make(Result, 0, len(lines))
	for color := range lines {
		l := Location{
			lat.Positions[color],
			lat.FnNames[color],
			lat.BBIds[color],
			math.Sqrt(float64(lineFails[color])/float64(totalTests)) *
				max(float64(lineFails[color])/float64(lineTotal[color]) - float64(lat.Fail.G.Graphs)/float64(totalTests),
					float64(lineFails[color])/float64(lat.Fail.G.Graphs) - float64(lineTotal[color])/float64(totalTests)),
		}
		if l.ValidRelativeMeasure() {
			result = append(result, l)
		}
	}
	result.Sort()
	return result
}

func enhancedTarantula(lat *lattice.Lattice) Result {
	lineFails := make(map[int]int)
	lineTotal := make(map[int]int)
	for color, instances := range lat.Fail.ColorIndex {
		lineFails[color] += len(instances)
		lineTotal[color] += len(instances)
	}
	for color, instances := range lat.Ok.ColorIndex {
		lineTotal[color] += len(instances)
	}
	totalTests := lat.Fail.G.Graphs + lat.Ok.G.Graphs
	lines := lat.Labels.Labels()
	result := make(Result, 0, len(lines))
	for color := range lines {
		l := Location{
			lat.Positions[color],
			lat.FnNames[color],
			lat.BBIds[color],
			float64(lineFails[color])/float64(lat.Fail.G.Graphs) *
				(float64(lineFails[color])/float64(lineTotal[color]) - float64(lat.Fail.G.Graphs)/float64(totalTests)),
		}
		if l.ValidRelativeMeasure() {
			result = append(result, l)
		}
	}
	result.Sort()
	return result
}
