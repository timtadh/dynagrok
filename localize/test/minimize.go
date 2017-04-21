package test

import (
	"math/rand"
	"runtime"
	"sync"
)

import (
	"github.com/timtadh/data-structures/errors"
)

import (
	"github.com/timtadh/dynagrok/localize/lattice"
	"github.com/timtadh/dynagrok/localize/lattice/subgraph"
)

func (t *Testcase) CanMinimize(lat *lattice.Lattice, sg *subgraph.SubGraph) (bool, error) {
	err := t.Execute()
	if err != nil {
		return false, err
	}
	if !t.Usable() {
		return false, errors.Errorf("unusable test case")
	}
	p, err := t.Digraph(lat)
	if err != nil {
		return false, err
	}
	if !sg.EmbeddedIn(p) {
		return false, nil
	}
	return true, nil
}

func (t *Testcase) Minimize(lat *lattice.Lattice, sg *subgraph.SubGraph) (*Testcase, error) {
	err := t.Execute()
	if err != nil {
		return nil, err
	}
	if !t.Usable() {
		return nil, errors.Errorf("unusable test case")
	}
	p, err := t.Digraph(lat)
	if err != nil {
		errors.Logf("ERROR", "could not load: %v", err)
		return nil, err
	}
	if !sg.EmbeddedIn(p) {
		errors.Logf("DEBUG", "cannot minimize %v with %v bytes and %v lines", t.From, len(t.Case), len(t.Lines()))
		return nil, nil
	}
	errors.Logf("DEBUG", "minimizing %v with %v bytes and %v lines", t.From, len(t.Case), len(t.Lines()))
	// errors.Logf("DEBUG", "trim suffixes")
	// t, err = t.minimizeWith(lat, sg, func(test *Testcase)[]*Mutant {
	// 	return atMost(200, test.LineEndTrimmingMuts())
	// })
	// if err != nil {
	// 	return nil, err
	// }
	// errors.Logf("DEBUG", "trim prefixes")
	// t, err = t.minimizeWith(lat, sg, func(test *Testcase)[]*Mutant {
	// 	return atMost(200, test.LineStartTrimmingMuts())
	// })
	// if err != nil {
	// 	return nil, err
	// }
	errors.Logf("DEBUG", "trim curly blocks")
	t, err = t.minimizeWith(lat, sg, atMost(50), func(test *Testcase) []*Mutant {
		return test.CurlyBlocks()
	})
	if err != nil {
		return nil, err
	}
	errors.Logf("DEBUG", "trim lines with matched curly blocks")
	t, err = t.minimizeWith(lat, sg, atMost(50), func(test *Testcase) []*Mutant {
		return test.LineCurlyBlocks()
	})
	if err != nil {
		return nil, err
	}
	errors.Logf("DEBUG", "trim blocks of lines")
	t, err = t.minimizeWith(lat, sg, atMost(50), func(test *Testcase) []*Mutant {
		return test.LineBlockTrimmingMuts()
	})
	if err != nil {
		return nil, err
	}
	errors.Logf("DEBUG", "trim lines")
	t, err = t.minimizeWith(lat, sg, atMost(150), func(test *Testcase) []*Mutant {
		return test.LineTrimmingMuts()
	})
	if err != nil {
		return nil, err
	}
	// errors.Logf("DEBUG", "trim blocks of lines")
	// t, err = t.minimizeWith(lat, sg, atMost(50), func(test *Testcase)[]*Mutant {
	// 	return test.LineBlockTrimmingMuts()
	// })
	// if err != nil {
	// 	return nil, err
	// }
	// TODO: make this configurable
	errors.Logf("DEBUG", "trim blocks")
	t, err = t.minimizeWith(lat, sg, atMost(50), func(test *Testcase) []*Mutant {
		return test.BlockTrimmingMuts()
	})
	if err != nil {
		return nil, err
	}
	// errors.Logf("DEBUG", "trim lines")
	// t, err = t.minimizeWith(lat, sg, func(test *Testcase)[]*Mutant {
	// 	return atMost(200, test.LineTrimmingMuts())
	// })
	// if err != nil {
	// 	return nil, err
	// }
	p, err = t.Digraph(lat)
	if err != nil {
		errors.Logf("ERROR", "could not load: %v", err)
		return nil, err
	}
	if !sg.EmbeddedIn(p) {
		// errors.Logf("DEBUG", "didn't contain subgraph")
		return nil, errors.Errorf("Minimized test didn't contain subgraph. %v", t)
	}
	return t, nil
}

type chooseMaker func() func([]*Mutant) ([]*Mutant, *Mutant)

func (t *Testcase) minimizeWith(lat *lattice.Lattice, sg *subgraph.SubGraph, mkChoose chooseMaker, f func(*Testcase) []*Mutant) (*Testcase, error) {
	type testOrError struct {
		test *Testcase
		err  error
	}
	gen := func(muts []*Mutant, out chan<- *Mutant, done <-chan bool) {
		choose := mkChoose()
		for len(muts) > 0 {
			var mut *Mutant
			muts, mut = choose(muts)
			if mut == nil {
				break
			}
			select {
			case out <- mut:
			case <-done:
				break
			}
		}
		close(out)
	}
	exec := func(wg *sync.WaitGroup, in <-chan *Mutant, out chan<- testOrError) {
		for mut := range in {
			test := mut.Testcase()
			err := test.Execute()
			if err != nil {
				errors.Logf("ERROR", "could not execute: %v", err)
				out <- testOrError{err: err}
				break
			}
			if !test.Usable() {
				// errors.Logf("DEBUG", "not usable")
				continue
			}
			p, err := test.Digraph(lat)
			if err != nil {
				errors.Logf("ERROR", "could not load: %v", err)
				out <- testOrError{err: err}
				break
			}
			if !sg.EmbeddedIn(p) {
				// errors.Logf("DEBUG", "didn't contain subgraph")
				continue
			}
			out <- testOrError{test: test}
			break
		}
		wg.Done()
	}
	workers := runtime.NumCPU()
	cur := t
	prev := cur
	muts := f(cur)
	errors.Logf("DEBUG", "cur %d %d %v", len(cur.Case), len(muts), cur.Failed())
	for cur != nil {
		var wg sync.WaitGroup
		done := make(chan bool)
		mutsCh := make(chan *Mutant)
		tests := make(chan testOrError)
		wg.Add(workers)
		for w := 0; w < workers; w++ {
			go exec(&wg, mutsCh, tests)
		}
		go func() {
			wg.Wait()
			close(tests)
		}()
		go gen(muts, mutsCh, done)
		te, ok := <-tests
		drain := make(chan bool)
		go func() {
			for range tests {
			}
			drain <- true
		}()
		close(done)
		<-drain
		if !ok {
			break
		}
		if te.err != nil {
			return nil, te.err
		}
		test := te.test
		prev = cur
		cur = test
		muts = f(cur)
		errors.Logf("DEBUG", "cur %d %d %v", len(cur.Case), len(muts), cur.Failed())
	}
	if cur != nil {
		return cur, nil
	}
	return prev, nil
}

func atMost(amt int) chooseMaker {
	return func() func(muts []*Mutant) ([]*Mutant, *Mutant) {
		chosen := 0
		return func(muts []*Mutant) ([]*Mutant, *Mutant) {
			if chosen >= amt {
				return nil, nil
			}
			chosen++
			return uniform(muts)
		}
	}
}

func uniform(slice []*Mutant) ([]*Mutant, *Mutant) {
	if len(slice) <= 0 {
		return nil, nil
	}
	i := rand.Intn(len(slice))
	item := slice[i]
	dst := slice[i : len(slice)-1]
	src := slice[i+1 : len(slice)]
	copy(dst, src)
	return slice[:len(slice)-1], item
}
