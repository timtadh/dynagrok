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
	//	similarity
	//C.match()

	// Step 4: ??
}

func (c *CausalEstimator) bin(numbins int) {
	c.inBins, c.inMedoids = KMedoids(numbins, c.inputs)
	c.outBins, c.outMedoids = KMedoids(numbins, c.outputs)
	fmt.Printf("Some input medoids: %v\n", c.inMedoids)
	fmt.Printf("Some input clusters: %v\n", c.inBins)
}

func (c *CausalEstimator) match() {
	// Yang - P Score Matching
	// Computer pairwise Causal Effects for each treatment
	// TODO : Compute only one triangle of this matrix
	for treatment1 := range c.outMedoids {
		for treatment2 := range c.outMedoids {
			if treatment1 == treatment2 {
				continue
			} else {
				c.pairwiseMatch(treatment1, treatment2)
			}
		}
	}
}

// pairwiseMatch determines the causal effect of treatment t1
// on treatment t2. t1 and t2 are the index of the c.outBins
// and c.outMedoids
func (c *CausalEstimator) pairwiseMatch(t1, t2 int) float64 {
	effect := 0.0
	for _, i := range c.outputs {
		y1 := c.outcome(t1, c.mCov(i, t1))
		y2 := c.outcome(t1, c.mCov(i, t2))
		effect = (y1 - y2) / float64(len(c.outputs))
	}
	return effect
}

func (c *CausalEstimator) outcome(treatment, ind int) float64 {
	// if c.outBins[treatment][ind] in the fail list, return 0.0;
	// else return 1.0
	return 0.0
}

// gets the covariates of the individual referred to by i
// in treatment level t
func (c *CausalEstimator) cov(i, t int) dgtypes.Clusterable {
	panic("Not Implemented")
	return nil
}

// mCov is the covariate matching function mentioned in Yang
// it returns the argmin over elements j in tlevel of ||individual - j||
// in terms of their covariates
func (c *CausalEstimator) mCov(individual dgtypes.Clusterable, tlevel int) int {
	min := 2.0
	matchindex := -1
	for i := range c.outBins[tlevel] {
		dist := c.cov(i, tlevel).Dissimilar(individual)
		if dist < min {
			min = dist
			matchindex = i
		}
	}
	return matchindex
}
