package locavore

import (
	"fmt"
	"github.com/timtadh/dynagrok/dgruntime/dgtypes"
)

type Localizer struct {
	ok      []dgtypes.Clusterable
	fail    []dgtypes.Clusterable
	inputs  []dgtypes.Clusterable
	outputs []dgtypes.Clusterable
	profs   []dgtypes.Clusterable
	inBins  map[dgtypes.Clusterable][]dgtypes.Clusterable
	outBins map[dgtypes.Clusterable][]dgtypes.Clusterable
}

func Localize(okf []dgtypes.FuncProfile, failf []dgtypes.FuncProfile, types []dgtypes.Type, numbins int) {
	var ok, fail []dgtypes.Clusterable = make([]dgtypes.Clusterable, 0), make([]dgtypes.Clusterable, 0)
	var in, out []dgtypes.Clusterable = make([]dgtypes.Clusterable, 0), make([]dgtypes.Clusterable, 0)
	for _, prof := range okf {
		for _, objprof := range append(prof.In, prof.Out...) {
			ok = append(ok, dgtypes.Clusterable(objprof))
		}
	}
	for _, prof := range failf {
		for _, objprof := range append(prof.In, prof.Out...) {
			fail = append(fail, dgtypes.Clusterable(objprof))
		}
	}
	for _, prof := range append(okf, failf...) {
		for _, objprof := range prof.In {
			in = append(in, dgtypes.Clusterable(objprof))
		}
		for _, objprof := range prof.Out {
			out = append(out, dgtypes.Clusterable(objprof))
		}
	}
	profs := append(ok, fail...)
	fmt.Printf("%v", profs)

	// Step 1:   Bin the inputs
	// Step 1.5: Bin the outputs
	l := Localizer{
		ok:      ok,
		fail:    fail,
		inputs:  in,
		outputs: out,
		profs:   profs,
	}
	l.bin(numbins)
	// Step 2: {optional} Propensity scoring
	// Step 3: Matching outputs with different outcomes, based on covariant
	//			similarity
	// Step 4: ??
}

func (l Localizer) bin(numbins int) {
	l.inBins = KMedoids(numbins, l.inputs)
	l.outBins = KMedoids(numbins, l.outputs)
	fmt.Printf("Some input clusters: %v", l.inBins)
	fmt.Printf("Some output clusters: %v", l.outBins)
}
