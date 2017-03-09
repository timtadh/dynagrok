package locavore

import (
	//"github.com/mdesenfants/gokmeans"
	"github.com/timtadh/dynagrok/dgruntime/dgtypes"
)

type Profile interface {
	Vector() []float64
}

type Localizer struct {
	ok        []Profile
	fail      []Profile
	profs     []Profile
	bins      [][]Profile
	centroids []Profile
}

func Localize(okf []dgtypes.FuncProfile, failf []dgtypes.FuncProfile, numbins int) {
	var ok, fail []Profile = make([]Profile, 0), make([]Profile, 0)
	for _, prof := range okf {
		ok = append(ok, prof)
	}
	for _, prof := range failf {
		fail = append(fail, prof)
	}

	// Step 1: Bin the profiles
	bins := make([][]Profile, numbins)
	for i := range bins {
		bins[i] = make([]Profile, 0)
	}
	profs := append(ok, fail...)

	l := Localizer{ok: ok, fail: fail, profs: profs, bins: bins}
	l.bin(numbins)
	// Step 2: Propensity scoring
	// Step 3: Matching
	// Step 4: ??
}

func (l Localizer) bin(numbins int) {
	//	profTable := make(map[gokmeans.Node][]Profile)
	//	observations := make([][]gokmeans.Node, 0)
	//
	//	// Fill out the vector-profs map
	//	// Fill out the observations list
	//	for i := 0; i < len(profs); i++ {
	//		vector := p.Vector()
	//		if ok := profTable[vector]; ok {
	//			profTable[vector] = append(profTable[vector], profs[i])
	//		} else {
	//			profTable[vector] = []Profile{profs[i]}
	//		}
	//		observations = append(observations, gokmeans.Node{vector})
	//	}
	//
	//	if success, centroids := gokmeans.Train(observations, numbins, 50); success {
	//		// Record the centroids
	//		for i, centroid := range centroids {
	//			l.centroids[i] = profTable[centroid]
	//		}
	//		// Record the clusters
	//		for _, observation := range observations {
	//			index := gokmeans.Nearest(observation, centroids)
	//			l.bins[index] = append(l.bins[index], profTable[observation])
	//		}
	//	}
}
