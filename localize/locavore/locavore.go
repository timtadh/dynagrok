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

type Individual struct {
	cov       dgtypes.Clusterable
	treatment dgtypes.Clusterable
	outcome   bool // true if pass, false is fail
}

func (i *Individual) Dissimilar(o dgtypes.Clusterable) float64 {
	if other, ok := o.(*Individual); ok {
		return i.treatment.Dissimilar(other.treatment)
	}
	panic("Expected another *Individual to be passed to Dissimilar")
}
func (i *Individual) String() string {
	return fmt.Sprintf("{In: %v, Out: %v, Outcome: %v}", i.cov, i.treatment, i.outcome)
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
	for i := range okf.In {
		if len(okf.Out) != 0 {
			ok = append(ok, &Individual{cov: okf.In[i], treatment: okf.Out[i], outcome: true})
		}
	}
	for i := range failf.In {
		if len(failf.Out) != 0 {
			fail = append(fail, &Individual{cov: failf.In[i], treatment: failf.Out[i], outcome: false})
		}
	}
	profs := append(ok, fail...)
	if len(profs) == 0 {
		log.Printf("Skipping profiles of %v (not enough data)...\n", okf.FuncName)
		return
	}

	log.Printf("Clustering profiles of %v...\n", okf.FuncName)

	// Step 1:   Bin the inputs
	// Step 1.5: Bin the outputs
	C := CausalEstimator{
		ok:    ok,
		fail:  fail,
		profs: append(ok, fail...),
	}
	C.bin(numbins)
	// Step 2: {optional} Propensity scoring
	// Step 3: Matching outputs with different outcomes, based on covariant
	//	similarity
	C.match()

	// Step 4: ??
}

func (c *CausalEstimator) bin(numbins int) {
	c.inBins, c.inMedoids = KMedoidsFunc(numbins, c.profs,
		func(this dgtypes.Clusterable, other dgtypes.Clusterable) float64 {
			if obj, ok := this.(*Individual); ok {
				if o, ok := other.(*Individual); ok {
					return obj.cov.Dissimilar(o.cov)
				}
			}
			panic("Expected *Individual")
		})
	c.outBins, c.outMedoids = KMedoids(numbins, c.profs)
	fmt.Printf("Some input medoids: %v\n", c.inMedoids)
	fmt.Printf("Some input clusters: %v\n", c.inBins)
}

func (c *CausalEstimator) match() {
	// Yang - P Score Matching
	// Computer pairwise Causal Effects for each treatment
	// TODO : Compute only one triangle of this matrix
	// TODO : Store this somewhere
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
	if x, ok := c.outBins[treatment][ind].(*Individual); ok {
		if x.outcome == false {
			return 0.0
		} else {
			return 1.0
		}
	} else {
		panic("Expected Individual")
	}
}

// gets the covariates of the individual referred to by i
// in treatment level t
func (c *CausalEstimator) cov(i, t int) dgtypes.Clusterable {
	if x, ok := c.outBins[t][i].(*Individual); ok {
		return x.cov
	} else {
		panic("Expected Individual")
	}
}

// mCov is the covariate matching function mentioned in Yang
// it returns the argmin over elements j in tlevel of ||individual - j||
// in terms of their covariates
func (c *CausalEstimator) mCov(individual dgtypes.Clusterable, tlevel int) int {
	if ind, ok := individual.(*Individual); ok {
		min := 2.0
		matchindex := -1
		for i := range c.outBins[tlevel] {
			dist := c.cov(i, tlevel).Dissimilar(ind.cov)
			if dist < min {
				min = dist
				matchindex = i
			}
		}
		return matchindex
	} else {
		panic("Expected Individual")
	}
}
