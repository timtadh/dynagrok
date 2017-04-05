package locavore

import (
	"fmt"
	"github.com/timtadh/dynagrok/dgruntime/dgtypes"
	"log"
)

type Localizer struct {
	ok      []dgtypes.Clusterable
	fail    []dgtypes.Clusterable
	inputs  []dgtypes.Clusterable
	outputs []dgtypes.Clusterable
	profs   []dgtypes.Clusterable
}

type CausalEstimator struct {
	ok         []dgtypes.Clusterable
	fail       []dgtypes.Clusterable
	inputs     []dgtypes.Clusterable
	outputs    []dgtypes.Clusterable
	profs      []dgtypes.Clusterable
	inBins     [][]dgtypes.Clusterable
	inMedoids  []dgtypes.Clusterable
	outBins    [][]dgtypes.Clusterable
	outMedoids []dgtypes.Clusterable
}

func Localize(okf []dgtypes.FuncProfile, failf []dgtypes.FuncProfile, types []dgtypes.Type, numbins int) {
	for _, okprof := range okf {
		for _, failprof := range failf {
			if okprof.FuncName == failprof.FuncName {
				CausalEffect(okprof, failprof, numbins)
			}
		}
	}
}

func CausalEffect(okf dgtypes.FuncProfile, failf dgtypes.FuncProfile, numbins int) {
	var ok, fail []dgtypes.Clusterable = make([]dgtypes.Clusterable, 0), make([]dgtypes.Clusterable, 0)
	var in, out []dgtypes.Clusterable = make([]dgtypes.Clusterable, 0), make([]dgtypes.Clusterable, 0)
	for _, objprof := range append(okf.In, okf.Out...) {
		ok = append(ok, dgtypes.Clusterable(objprof))
	}
	for _, objprof := range append(failf.In, failf.Out...) {
		fail = append(fail, dgtypes.Clusterable(objprof))
	}
	for _, objprof := range append(okf.In, failf.In...) {
		in = append(in, dgtypes.Clusterable(objprof))
	}
	for _, objprof := range append(okf.Out, failf.Out...) {
		out = append(out, dgtypes.Clusterable(objprof))
	}

	profs := append(ok, fail...)
	log.Printf("Clustering profiles of %v...\n", okf.FuncName)

	// Step 1:   Bin the inputs
	// Step 1.5: Bin the outputs
	C := CausalEstimator{
		ok:      ok,
		fail:    fail,
		inputs:  in,
		outputs: out,
		profs:   profs,
	}
	C.bin(numbins)
	// Step 2: {optional} Propensity scoring
	// Step 3: Matching outputs with different outcomes, based on covariant
	//			similarity
	// Step 4: ??
}

func (c *CausalEstimator) bin(numbins int) {
	c.inBins, c.inMedoids = KMedoids(numbins, c.inputs)
	c.outBins, c.outMedoids = KMedoids(numbins, c.outputs)
	fmt.Printf("Some input medoids: %v\n", c.inMedoids)
	fmt.Printf("Some input clusters: %v\n", c.inBins)
}
