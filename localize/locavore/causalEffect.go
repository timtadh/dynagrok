package locavore

import (
	"fmt"
	"github.com/timtadh/dynagrok/dgruntime/dgtypes"
	"log"
	"os"
)

type CausalEstimator struct {
	// defined at init
	ok    []dgtypes.Clusterable
	fail  []dgtypes.Clusterable
	profs []dgtypes.Clusterable // ok appended to fail
	// defined after C.bin()
	inBins     [][]dgtypes.Clusterable
	inMedoids  []dgtypes.Clusterable
	outBins    [][]dgtypes.Clusterable
	outMedoids []dgtypes.Clusterable
	// defined after C.match()
	simMatrix [][]float64
}

// An Individual is one data point
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

func covDissimilar(ind dgtypes.Clusterable, o dgtypes.Clusterable) float64 {
	if i, ok := ind.(*Individual); ok {
		if other, ok := o.(*Individual); ok {
			return i.cov.Dissimilar(other.cov)
		}
	}
	panic("Expected another *Individual to be passed to Dissimilar")
}

func (i *Individual) String() string {
	return fmt.Sprintf("{In: %v, Out: %v, Outcome: %v}", i.cov, i.treatment, i.outcome)
}

func CausalEffect(okf dgtypes.FuncProfile, failf dgtypes.FuncProfile, numbins int) ([]dgtypes.Clusterable, [][]float64, error) {
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
	if len(profs) < 3 {
		log.Printf("Skipping profiles of %v (not enough data)...\n", okf.FuncName)
		return make([]dgtypes.Clusterable, 0), make([][]float64, 0), error(fmt.Errorf("Not enough data"))
	}

	log.Printf("Clustering profiles of %v...\n", okf.FuncName)

	// Step 1:   Bin the inputs
	// Step 1.5: Bin the outputs
	C := CausalEstimator{
		ok:    ok,
		fail:  fail,
		profs: profs,
	}
	C.bin(numbins)
	// Step 2: {optional} Propensity scoring
	// Step 3: Matching outputs with different outcomes, based on covariant
	//	similarity
	log.Printf("Matching profiles of %v...\n", okf.FuncName)
	C.match()
	log.Printf("Matrix of causal-effect pairs for %v...\n", okf.FuncName)
	printMatrix(C.simMatrix)

	return C.outMedoids, C.simMatrix, nil
	// Step 4: ??
}

func (c *CausalEstimator) bin(numbins int) {
	c.inBins, c.inMedoids = KMedoidsFunc(numbins, c.profs, covDissimilar)
	c.outBins, c.outMedoids = KMedoids(numbins, c.profs)
	//for i := range c.outBins {
	//	fmt.Printf("Medoid: %v\n", c.outMedoids[0])
	//	for j := range c.outBins[i] {
	//		fmt.Printf("\t%d: %v\n", j, c.outBins[i][j])
	//	}
	//}
	log.Printf("Adding medoids to their respective clusters...")
	for i, j := range c.inMedoids {
		c.inBins[i] = append(c.inBins[i], j)
	}
	for i, j := range c.outMedoids {
		c.outBins[i] = append(c.inBins[i], j)
	}
	//	fmt.Printf("%v ouput medoids: %v\n", len(c.outMedoids), c.outMedoids)
	//fmt.Printf("%v output clusters: %v\n", len(c.outBins), c.outBins)
}

func (c *CausalEstimator) match() {
	// Yang - P Score Matching
	// Computer pairwise Causal Effects for each treatment
	c.simMatrix = make([][]float64, len(c.outMedoids))
	for treatment1 := range c.outMedoids {
		c.simMatrix[treatment1] = make([]float64, len(c.outMedoids))
		for treatment2 := range c.outMedoids {
			if treatment1 >= treatment2 {
				c.simMatrix[treatment1][treatment2] = -1 * c.simMatrix[treatment2][treatment1]
			} else {
				c.simMatrix[treatment1][treatment2] = c.pairwiseMatch(treatment1, treatment2)
			}
		}
	}
}

// pairwiseMatch determines the causal effect of treatment t1
// on treatment t2. t1 and t2 are the index of the c.outBins
// and c.outMedoids
func (c *CausalEstimator) pairwiseMatch(t1, t2 int) float64 {
	effect := 0.0
	for _, i := range c.profs {
		y1 := c.outcome(t1, c.mCov(i, t1))
		y2 := c.outcome(t2, c.mCov(i, t2))
		effect += (y1 - y2) / float64(len(c.profs))
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

func printMatrix(matrix [][]float64) {
	for r := range matrix {
		for c := range matrix[r] {
			fmt.Fprintf(os.Stderr, "%.3f ", matrix[r][c])
		}
		fmt.Fprintf(os.Stderr, "\n")
	}
}
