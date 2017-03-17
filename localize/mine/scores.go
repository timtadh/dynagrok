package mine

import (
	"math"
)

var ScoreAbbrvs map[string]string
var ScoreNames map[string][]string

func init() {
	ScoreAbbrvs = map[string]string{
		"rp":        "RelativePrecision",
		"rf1":       "RelativeF1",
		"rj":        "RelativeJaccard",
		"ro":        "RelativeOchiai",
		"precision": "Precision",
		"p":         "Precision",
		"f1":        "F1",
		"jaccard":   "Jaccard",
		"j":         "Jaccard",
		"o":         "Ochiai",
		"o2":        "OchiaiSquared",
		"och":       "Ochiai",
		"ochiai":    "Ochiai",
		"c":         "Contrast",
		"ar":        "AssociationalRisk",
		"ig":        "InformationGain",
	}
	ScoreNames = make(map[string][]string)
	for abbrv, name := range ScoreAbbrvs {
		ScoreNames[name] = append(ScoreNames[name], abbrv)
	}
}

var Scores = map[string]ScoreFunc{
	"RelativePrecision": func(prF, prFandNode, prO, prOandNode float64) float64 {
		prf := prFandNode
		pro := prOandNode
		a := prf / (prf + pro)
		b := prF / (prF + prO)
		s := (a - b)
		return s
	},
	"RelativeF1": func(prF, prFandNode, prO, prOandNode float64) float64 {
		prf := prFandNode
		pro := prOandNode
		a := prf / (prf + pro)
		b := prF / (prF + prO)
		prt := prf + pro
		s := 2 * (prt / (prF + prt)) * (a - b)
		return s
	},
	"RelativeJaccard": func(prF, prFandNode, prO, prOandNode float64) float64 {
		prf := prFandNode
		pro := prOandNode
		b := prF / (prF + prO)
		s := ((prf / (prF + pro)) - b)
		return s
	},
	"RelativeOchiai": func(prF, prFandNode, prO, prOandNode float64) float64 {
		// Only works when prF < (prf/(prf + pro))
		prf := prFandNode
		pro := prOandNode
		prt := prf + pro
		a := prf / (prf + pro)
		b := prF / (prF + prO)
		s := math.Sqrt((prt / prF)) * (a - b)
		return s
	},
	"Precision": func(prF, prFandNode, prO, prOandNode float64) float64 {
		prf := prFandNode
		pro := prOandNode
		return prf / (prf + pro)
	},
	"F1": func(prF, prFandNode, prO, prOandNode float64) float64 {
		prf := prFandNode
		pro := prOandNode
		a := prf / (prf + pro)
		prt := prf + pro
		s := 2 * (prt / (prF + prt)) * (a)
		return s
	},
	"Jaccard": func(prF, prFandNode, prO, prOandNode float64) float64 {
		prf := prFandNode
		pro := prOandNode
		s := prf / (prF + pro)
		return s
	},
	"OchiaiSquared": func(prF, prFandNode, prO, prOandNode float64) float64 {
		prf := prFandNode
		pro := prOandNode
		prt := prf + pro
		s := (prf / prF) * (prf / prt)
		return s
	},
	"Ochiai": func(prF, prFandNode, prO, prOandNode float64) float64 {
		prf := prFandNode
		pro := prOandNode
		prt := prf + pro
		s := math.Sqrt((prf / prF) * (prf / prt))
		return s
	},
	"Contrast": func(prF, prFandNode, prO, prOandNode float64) float64 {
		prf := prFandNode
		pro := prOandNode
		return prf - pro
	},
	"expr": func(prF, prFandNode, prO, prOandNode float64) float64 {
		prf := prFandNode
		pro := prOandNode
		return (prf - pro) / (prF + pro)
	},
	"AssociationalRisk": func(prF, prFandNode, prO, prOandNode float64) float64 {
		prf := prFandNode
		pro := prOandNode
		c, x, y := prF, prf, pro
		top := x - c*x - c*y
		bot := (x + y + .00001) - ((x + y) * (x + y))
		return top / bot
	},
	"InformationGain": func(prF, prFandNode, prO, prOandNode float64) float64 {
		lg := func(x float64) float64 {
			if x == 0 {
				return 0
			}
			return math.Log2(x)
		}
		prf := prFandNode
		pro := prOandNode
		prt := prf + pro
		HF := prF*lg(prF) + prO*lg(prO)
		HFn := (prf/prt)*lg(prf/prt) + (pro/prt)*lg(pro/prt)
		return HFn - HF
	},
}
