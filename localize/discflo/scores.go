package discflo

import (
	"math"
)

import ()

import (
	"github.com/timtadh/dynagrok/localize/lattice"
)


var scoreAbbrvs map[string]string
var scoreNames map[string][]string

func init() {
	scoreAbbrvs = map[string]string{
		// "swrp": "SizeWeightedRelativePrecision",
		// "swrf1": "SizeWeightedRelativeF1",
		// "swrj": "SizeWeightedRelativeJaccard",
		// "swro": "SizeWeightedRelativeOchiai",
		"rp": "RelativePrecision",
		"rf1": "RelativeF1",
		"rj": "RelativeJaccard",
		"ro": "RelativeOchiai",
		"precision": "Precision",
		"p": "Precision",
		"f1": "F1",
		"jaccard": "Jaccard",
		"j": "Jaccard",
		"o": "Ochiai",
		"och": "Ochiai",
		"ochiai": "Ochiai",
	}
	scoreNames = make(map[string][]string)
	for abbrv, name := range scoreAbbrvs {
		scoreNames[name] = append(scoreNames[name], abbrv)
	}
}

type Score func(lat *lattice.Lattice, n *lattice.Node) float64

func Prs(lat *lattice.Lattice, n *lattice.Node) (prF, prO, prf, pro float64) {
	F := float64(lat.Fail.G.Graphs)
	O := float64(lat.Ok.G.Graphs)
	T := F + O
	f := (float64(n.FIS()))
	if len(n.SubGraph.E) > 0 || len(n.SubGraph.V) >= 1 {
		var o float64
		for i := range n.SubGraph.E {
			count := lat.Ok.EdgeCounts[n.SubGraph.Colors(i)]
			o += float64(count)/T
		}
		for i := range n.SubGraph.V {
			count := float64(len(lat.Ok.ColorIndex[n.SubGraph.V[i].Color]))
			o += float64(count)/T
		}
		pro = o/float64(len(n.SubGraph.V) + len(n.SubGraph.E))
	} else {
		pro = O/T
	}
	return F/T, O/T, f/T, pro
}

var Scores = map[string]Score {
	// "SizeWeightedRelativePrecision": func(lat *lattice.Lattice, n *lattice.Node) float64 {
	// 	prF, prO, prf, pro := Prs(lat, n)
	// 	E := float64(len(lat.Fail.G.E))
	// 	e := float64(len(n.SubGraph.E)) + 1
	// 	a := prf/(prf + pro)
	// 	b := prF/(prF + prO)
	// 	s := ((e+1)/E) * (a - b)
	// 	return s
	// },
	// "SizeWeightedRelativeF1": func(lat *lattice.Lattice, n *lattice.Node) float64 {
	// 	prF, prO, prf, pro := Prs(lat, n)
	// 	E := float64(len(lat.Fail.G.E))
	// 	e := float64(len(n.SubGraph.E)) + 1
	// 	a := prf/(prf + pro)
	// 	b := prF/(prF + prO)
	// 	prt := prf + pro
	// 	s := ((e+1)/E) * 2 * (prt/(prF + prt)) * (a - b)
	// 	return s
	// },
	// "SizeWeightedRelativeJaccard": func(lat *lattice.Lattice, n *lattice.Node) float64 {
	// 	prF, prO, prf, pro := Prs(lat, n)
	// 	E := float64(len(lat.Fail.G.E))
	// 	e := float64(len(n.SubGraph.E)) + 1
	// 	b := prF/(prF + prO)
	// 	s := ((e+1)/E) * ((prf / (prF + pro)) - b)
	// 	return s
	// },
	// "SizeWeightedRelativeOchiai": func(lat *lattice.Lattice, n *lattice.Node) float64 {
	// 	prF, prO, prf, pro := Prs(lat, n)
	// 	E := float64(len(lat.Fail.G.E))
	// 	e := float64(len(n.SubGraph.E)) + 1
	// 	prt := prf + pro
	// 	a := prf/(prf + pro)
	// 	b := prF/(prF + prO)
	// 	s := ((e+1)/E) * math.Sqrt((prt/prF)) * (a - b)
	// 	return s
	// },
	"RelativePrecision": func(lat *lattice.Lattice, n *lattice.Node) float64 {
		prF, prO, prf, pro := Prs(lat, n)
		a := prf/(prf + pro)
		b := prF/(prF + prO)
		s := (a - b)
		return s
	},
	"RelativeF1": func(lat *lattice.Lattice, n *lattice.Node) float64 {
		prF, prO, prf, pro := Prs(lat, n)
		a := prf/(prf + pro)
		b := prF/(prF + prO)
		prt := prf + pro
		s := 2 * (prt/(prF + prt)) * (a - b)
		return s
	},
	"RelativeJaccard": func(lat *lattice.Lattice, n *lattice.Node) float64 {
		prF, prO, prf, pro := Prs(lat, n)
		b := prF/(prF + prO)
		s := ((prf / (prF + pro)) - b)
		return s
	},
	"RelativeOchiai": func(lat *lattice.Lattice, n *lattice.Node) float64 {
		prF, prO, prf, pro := Prs(lat, n)
		prt := prf + pro
		a := prf/(prf + pro)
		b := prF/(prF + prO)
		s := math.Sqrt((prt/prF)) * (a - b)
		return s
	},
	"Precision": func(lat *lattice.Lattice, n *lattice.Node) float64 {
		_, _, prf, pro := Prs(lat, n)
		a := prf/(prf + pro)
		return a
	},
	"F1": func(lat *lattice.Lattice, n *lattice.Node) float64 {
		prF, _, prf, pro := Prs(lat, n)
		a := prf/(prf + pro)
		prt := prf + pro
		s := 2 * (prt/(prF + prt)) * (a)
		return s
	},
	"Jaccard": func(lat *lattice.Lattice, n *lattice.Node) float64 {
		prF, _, prf, pro := Prs(lat, n)
		s := prf / (prF + pro)
		return s
	},
	"Ochiai": func(lat *lattice.Lattice, n *lattice.Node) float64 {
		prF, _, prf, pro := Prs(lat, n)
		prt := prf + pro
		s := math.Sqrt((prf/prF) * (prf/prt))
		return s
	},
}
