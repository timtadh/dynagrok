package locavore

import (
	"fmt"
	"sort"
)

func printScores(suspiciousness map[string]float64) {
	pl := sortByScore(suspiciousness)
	for i, pair := range pl {
		fmt.Printf("%d. %s: %.3f\n", i+1, pair.Key, pair.Value)
	}
}

func sortByScore(suspiciousness map[string]float64) PairList {
	pl := make(PairList, len(suspiciousness))
	i := 0
	for k, v := range suspiciousness {
		pl[i] = Pair{k, v}
		i++
	}
	sort.Sort(sort.Reverse(pl))
	return pl
}

type Pair struct {
	Key   string
	Value float64
}

type PairList []Pair

func (p PairList) Len() int           { return len(p) }
func (p PairList) Less(i, j int) bool { return p[i].Value < p[j].Value }
func (p PairList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
