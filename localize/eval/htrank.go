package eval

import (
	"bytes"
	crand "crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"sort"
	"strconv"

	"github.com/timtadh/data-structures/errors"
	"github.com/timtadh/dynagrok/localize/mine"
	matrix "github.com/timtadh/go.matrix"
)

func (e *Evaluator) HTRank(methodName, scoreName, chainName string, colorStates map[int][]int, P [][]float64) (results EvalResults) {
	group := func(order []int, scores map[int]float64) [][]int {
		sort.Slice(order, func(i, j int) bool {
			return scores[order[i]] < scores[order[j]]
		})
		groups := make([][]int, 0, 10)
		for _, n := range order {
			lg := len(groups)
			if lg > 0 && scores[n] == scores[groups[lg-1][0]] {
				groups[lg-1] = append(groups[lg-1], n)
			} else {
				groups = append(groups, make([]int, 0, 10))
				groups[lg] = append(groups[lg], n)
			}
		}
		return groups
	}
	faultColors := make(map[int]bool)
	for color := range e.lattice.Labels.Labels() {
		if e.Fault(color) != nil {
			faultColors[color] = true
		}
	}
	if len(faultColors) <= 0 {
		return nil
	}
	found := false
	for f := range faultColors {
		if len(colorStates[f]) > 0 {
			found = true
		}
	}
	if !found {
		return nil
	}
	scores := e.getHitScores(colorStates, P)

	order := make([]int, 0, len(colorStates))
	for color := range colorStates {
		order = append(order, color)
	}
	grouped := group(order, scores)
	ranks := make(map[int]float64)
	total := 0
	for _, group := range grouped {
		count := 0
		for _, color := range group {
			count++
			ranks[color] = float64(total) + float64(len(group))/2
		}
		total += len(group)
	}
	var min *MarkovEvalResult
	for color, score := range scores {
		if f := e.Fault(color); f != nil {
			label := e.lattice.Labels.Label(color)
			b, fn, pos := e.lattice.Info.Get(color)
			r := &MarkovEvalResult{
				MethodName:    methodName,
				ScoreName:     scoreName,
				ChainName:     chainName,
				HT_Rank:       ranks[color],
				HT_RankMethod: e.HTRankMethod(len(P)),
				HittingTime:   score,
				fault:         f,
				loc: &mine.Location{
					Color:        color,
					BasicBlockId: b,
					FnName:       fn,
					Position:     pos,
					Label:        label,
				},
			}
			if min == nil || r.HT_Rank < min.HT_Rank {
				min = r
			}
		}
	}
	return EvalResults{min}
}

func (e *Evaluator) getHitScores(colorStates map[int][]int, P [][]float64) map[int]float64 {
	scores := make(map[int]float64)
	if e.ExactHTRank(len(P)) {
		errors.Logf("INFO", "computing exact HTrank, P: %vx%v\n", len(P), len(P))
		states := make([]int, 0, len(colorStates))
		order := make([]int, 0, len(colorStates))
		for color, group := range colorStates {
			order = append(order, color)
			for _, state := range group {
				states = append(states, state)
			}
		}
		hittingTimes, err := ParPyEHT(e.Workers(), 0, states, P)
		if err != nil {
			fmt.Println(err)
			errors.Logf("WARNING", "falling back on go implementation of hittingTime computation")
			for color, states := range colorStates {
				for _, state := range states {
					hit := ExpectedHittingTime(0, state, P)
					if min, has := scores[color]; !has || hit < min {
						scores[color] = hit
					}
				}
			}
		} else {
			for color, states := range colorStates {
				for _, state := range states {
					hit, has := hittingTimes[state]
					if !has {
						continue
					}
					if min, has := scores[color]; !has || hit < min {
						scores[color] = hit
					}
				}
			}
		}
	} else {
		errors.Logf("INFO", "estimating HTrank, P: %vx%v\n", len(P), len(P))
		hittingTimes := EsimateEspectedHittingTimes(e.Workers(), 500, 0, 1000000, P)
		for color, states := range colorStates {
			for _, state := range states {
				if state < len(hittingTimes) {
					hit := hittingTimes[state]
					if min, has := scores[color]; !has || hit < min {
						scores[color] = hit
					}
				}
			}
		}
	}
	for color, hit := range scores {
		if hit <= 0 {
			scores[color] = math.Inf(1)
		} else {
			scores[color] = math.Round(hit)
		}
	}
	return scores
}

func ExpectedHittingTimes(transitions [][]float64) [][]float64 {
	P := matrix.MakeDenseMatrixStacked(transitions)
	M := matrix.Zeros(P.Rows(), P.Cols())
	for i := 0; i < P.Rows()*P.Rows(); i++ {
		prevM := M.Copy()
		for t := 0; t < M.Rows(); t++ {
			for s := 0; s < M.Cols(); s++ {
				if t == s {
					M.Set(t, s, 0.0)
				} else {
					sum := 0.0
					for k := 0; k < P.Rows(); k++ {
						if k != s {
							sum += P.Get(t, k) * (M.Get(k, s) + 1)
						}
					}
					M.Set(t, s, P.Get(t, s)+sum)
				}
			}
		}
		diff, err := M.Minus(prevM)
		if err != nil {
			panic(err)
		}
		if diff.DenseMatrix().TwoNorm() < .01 {
			break
		}
	}
	return M.Arrays()
}

func ExpectedHittingTime(x, y int, transitions [][]float64) float64 {
	P := matrix.MakeDenseMatrixStacked(transitions)
	for s := 0; s < P.Cols(); s++ {
		P.Set(y, s, 0)
	}
	P.Set(y, y, 1)
	last := P.Rows()
	P.SwapRows(y, last-1)
	P = P.Transpose()
	P.SwapRows(y, last-1)
	P = P.Transpose()
	Q := P.GetMatrix(0, 0, last-1, last-1)
	I := matrix.Eye(Q.Rows())
	c := matrix.Ones(Q.Rows(), 1)
	IQ, err := I.Minus(Q)
	if err != nil {
		panic(err)
	}
	N, err := IQ.DenseMatrix().Inverse()
	if err != nil {
		panic(err)
	}
	Nc, err := N.Times(c)
	if err != nil {
		panic(err)
	}
	// fmt.Println(x, y, Nc.Get(x,0))
	return Nc.Get(x, 0)
}

func ParPyEHT(cpus, start int, states []int, transitions [][]float64) (map[int]float64, error) {
	if states == nil {
		panic("states is nil")
	}
	errors.Logf("INFO", "total states to compute: %v, workers: %v", len(states), cpus)
	work := make([][]int, cpus)
	for i, state := range states {
		w := i % len(work)
		work[w] = append(work[w], state)
	}
	type result struct {
		hits map[int]float64
		err  error
	}
	hits := make(map[int]float64, len(states))
	results := make(chan result)
	expected := 0
	for w := range work {
		if len(work[w]) > 0 {
			expected++
			go func(nodeId int, mine []int) {
				hits, err := PyExpectedHittingTimes(nodeId, start, mine, transitions)
				results <- result{hits, err}
			}(w, work[w])
		}
	}
	var err error
	for i := 0; i < expected; i++ {
		r := <-results
		if r.err != nil {
			err = r.err
			continue
		}
		for state, time := range r.hits {
			hits[state] = time
		}
	}
	if err != nil {
		return nil, err
	}
	return hits, nil
}

func PyExpectedHittingTimes(nodeId int, start int, states []int, transitions [][]float64) (map[int]float64, error) {
	if states == nil {
		panic("states is nil")
	}
	type data struct {
		Start       int
		States      []int
		Transitions [][]float64
	}
	encoded, err := json.Marshal(data{start, states, transitions})
	if err != nil {
		return nil, err
	}
	var outbuf, errbuf bytes.Buffer
	inbuf := bytes.NewBuffer(encoded)
	c := exec.Command("hitting-times", fmt.Sprintf("%v", nodeId))
	c.Stdin = inbuf
	c.Stdout = &outbuf
	c.Stderr = io.MultiWriter(os.Stderr, &errbuf)
	err = c.Start()
	if err != nil {
		return nil, err
	}
	err = c.Wait()
	if err != nil {
		return nil, fmt.Errorf("py hitting time err: %v\n`%v`\n`%v`", err, errbuf.String(), outbuf.String())
	}
	if !c.ProcessState.Success() {
		return nil, fmt.Errorf("failed to have python compute hitting times: %v\n %v", errbuf.String(), outbuf.String())
	}
	var times map[string]float64
	err = json.Unmarshal(outbuf.Bytes(), &times)
	if err != nil {
		return nil, fmt.Errorf("py hitting time err, could not unmarshall: %v\n`%v`", err, outbuf.String())
	}
	hits := make(map[int]float64)
	for sState, time := range times {
		state, err := strconv.Atoi(sState)
		if err != nil {
			return nil, fmt.Errorf("py hitting time err, could not unmarshall: %v\n`%v`", err, outbuf.String())
		}
		hits[state] = time
	}
	return hits, nil
}

func EsimateEspectedHittingTimes(workers, walks, start, maxLength int, transitions [][]float64) []float64 {
	estimates := make([]float64, 0, len(transitions))
	samples := RandomWalksForHittingTimes(workers, walks, start, maxLength, transitions, reachable(start, transitions))
	fmt.Println("sample count", len(samples))
	distributions := transpose(samples)
	for _, distribution := range distributions {
		estimates = append(estimates, estExpectedTime(distribution))
	}
	return estimates
}

func reachable(start int, transitions [][]float64) []int {
	found := make(map[int]bool)
	stack := make([]int, 0, len(transitions))
	stack = append(stack, start)
	for len(stack) > 0 {
		var i int
		i, stack = stack[len(stack)-1], stack[:len(stack)-1]
		if found[i] {
			continue
		}
		found[i] = true
		for j, p := range transitions[i] {
			if p > 0 {
				stack = append(stack, j)
			}
		}
	}
	reachable := make([]int, 0, len(found))
	for i := range found {
		reachable = append(reachable, i)
	}
	return reachable
}

func transpose(samples [][]uint64) (distributions [][]uint64) {
	distributions = make([][]uint64, len(samples[0]))
	for i := range distributions {
		distributions[i] = make([]uint64, len(samples))
	}
	for i, sample := range samples {
		for j, value := range sample {
			distributions[j][i] = value
		}
	}
	return distributions
}

func estExpectedTime(distribution []uint64) float64 {
	sort.Slice(distribution, func(i, j int) bool {
		return distribution[i] < distribution[j]
	})
	total := 0.0
	for _, s := range distribution {
		total += float64(s)
	}
	maxTime := int(distribution[len(distribution)-1])
	cumulative := func(t int) float64 {
		sum := 0.0
		for i := 0; i < len(distribution); i++ {
			if distribution[i] < uint64(t) {
				sum += 1
			}
		}
		return (1 / float64(len(distribution))) * sum
	}
	var sum float64
	for i := 0; i < len(distribution)-1; i++ {
		sum += float64(distribution[i+1]-distribution[i]) * cumulative(int(distribution[i]))
	}
	est := float64(maxTime) - sum
	return est
}

func RandomWalksForHittingTimes(cpus, walks, start, maxLength int, transitions [][]float64, reachable []int) [][]uint64 {
	results := make(chan []uint64, walks)
	count := 0
	for count < walks {
		prev := count
		count += walks / cpus
		if count >= walks {
			count = walks
		}
		go func(mywalks int) {
			seed := make([]byte, 8)
			if _, err := crand.Read(seed); err != nil {
				panic(err)
			}
			random := rand.New(rand.NewSource(int64(binary.BigEndian.Uint64(seed))))
			for w := 0; w < mywalks; w++ {
				results <- RandomWalkHittingTime(random, start, maxLength, transitions, reachable)
			}
		}(count - prev)
	}
	var distribution [][]uint64
	for i := 0; i < count; i++ {
		distribution = append(distribution, <-results)
	}
	return distribution
}

func RandomWalkHittingTime(random *rand.Rand, start int, maxLength int, transitions [][]float64, reachable []int) []uint64 {
	c := start
	found := make(map[int]bool)
	times := make([]uint64, len(transitions))
	for i := uint64(0); i < uint64(maxLength); i++ {
		if len(found) >= len(reachable) {
			break
		}
		if !found[c] {
			times[c] = i
			found[c] = true
		}
		c = weightedSample(random, transitions[c])
	}
	if len(found) != len(transitions) {
		for c := range times {
			if !found[c] {
				times[c] = uint64(maxLength)
			}
		}
	}
	return times
}

func weightedSample(random *rand.Rand, weights []float64) int {
	var total float64
	for _, w := range weights {
		total += w
	}
	i := 0
	r := total * random.Float64()
	for ; i < len(weights)-1 && r > weights[i]; i++ {
		r -= weights[i]
	}
	return i
}
