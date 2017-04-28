package locavore

import (
	"fmt"
	"log"
	"math"

	"github.com/timtadh/dynagrok/dgruntime/dgtypes"
)

// Compile takes a list of passing FuncProfiles and a list of failing
// FuncProfiles. It returns these lists, after appending together profiles which
// are defined on the same funcName.
func Collate(okf []dgtypes.FuncProfile, failf []dgtypes.FuncProfile) ([]dgtypes.FuncProfile, []dgtypes.FuncProfile) {
	return collateProf(okf), collateProf(failf)
}
func collateProf(profiles []dgtypes.FuncProfile) []dgtypes.FuncProfile {
	var ret []dgtypes.FuncProfile = make([]dgtypes.FuncProfile, 0)
	for _, prof := range profiles {
		contains := false
		for i := range ret {
			if ret[i].FuncName == prof.FuncName {
				contains = true
				if len(prof.In) != len(prof.Out) {
					fmt.Printf("In (%d):\n%v\n\nOut (%d):\n%v\n\n", len(prof.In), prof.In, len(prof.Out), prof.Out)
				}
				ret[i].In = append(ret[i].In, prof.In...)
				ret[i].Out = append(ret[i].Out, prof.Out...)
				break
			}
		}
		if !contains {
			ret = append(ret, prof)
		}
	}
	return ret
}

func Localize(okf []dgtypes.FuncProfile, failf []dgtypes.FuncProfile, types []dgtypes.Type, numbins int) {
	suspiciousness := make(map[string]float64)
	okf, failf = Collate(okf, failf)
	for _, okprof := range okf {
		for _, failprof := range failf {
			if okprof.FuncName == failprof.FuncName {
				fmt.Println("")
				log.Printf("--- Attempting to calculate causal effect for %s ---", okprof.FuncName)
				treatments, pairwiseEffects, err := CausalEffect(okprof, failprof, numbins)
				if err == nil {
					log.Printf("--- Succesfully calculated causal effect for %s ---", okprof.FuncName)
				}
				suspiciousness[okprof.FuncName] = Suspiciousness(pairwiseEffects)
				_ = treatments
			}
		}
	}
	// TODO print Max-value suspiciousness as well. That will work better for
	// certain faults
	// TODO create test driver for some empty-list bug fault.
	fmt.Println("")
	log.Printf("--- Finished calculating causal effect ---")
	log.Printf("Printing scores...")
	printScores(suspiciousness)

}

// Suspiciousness takes a matrix of causal effect pairs and computes some metric
// to determine the overal suspiciousness of the associated function.
func Suspiciousness(matrix [][]float64) float64 {
	return math.Abs(Average(matrix))
}

func Average(matrix [][]float64) float64 {
	avg := 0.0
	for i := range matrix {
		for j := range matrix {
			if i >= j {
				continue
			}
			avg += matrix[i][j] / float64(len(matrix))
		}
	}
	return avg
}
