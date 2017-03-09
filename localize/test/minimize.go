package test

import (
	"sync"
	"math/rand"
	"runtime"
)

import (
	"github.com/timtadh/data-structures/errors"
)

import (
	"github.com/timtadh/dynagrok/localize/lattice"
	"github.com/timtadh/dynagrok/localize/lattice/subgraph"
)

func (t *Testcase) Minimize(lat *lattice.Lattice, sg *subgraph.SubGraph) (*Testcase, error){
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
	errors.Logf("DEBUG", "trim blocks of lines")
	t, err = t.minimizeWith(lat, sg, func(test *Testcase)[]*Mutant {
		return atMost(50, test.LineBlockTrimmingMuts())
	})
	if err != nil {
		return nil, err
	}
	errors.Logf("DEBUG", "trim lines")
	t, err = t.minimizeWith(lat, sg, func(test *Testcase)[]*Mutant {
		return atMost(150, test.LineTrimmingMuts())
	})
	if err != nil {
		return nil, err
	}
	errors.Logf("DEBUG", "trim blocks of lines")
	t, err = t.minimizeWith(lat, sg, func(test *Testcase)[]*Mutant {
		return atMost(50, test.LineBlockTrimmingMuts())
	})
	if err != nil {
		return nil, err
	}
	// TODO: make this configurable
	// errors.Logf("DEBUG", "trim blocks")
	// t, err = t.minimizeWith(lat, sg, func(test *Testcase)[]*Mutant {
	// 	return atMost(150, test.BlockTrimmingMuts())
	// })
	// if err != nil {
	// 	return nil, err
	// }
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

func (t *Testcase) minimizeWith(lat *lattice.Lattice, sg *subgraph.SubGraph, f func(*Testcase)[]*Mutant) (*Testcase, error) {
	type testOrError struct {
		test *Testcase
		err  error
	}
	gen := func(muts []*Mutant, out chan<- *Mutant, done <-chan bool) {
		for len(muts) > 0 {
			var mut *Mutant
			muts, mut = uniform(muts)
			if mut == nil {
				break
			}
			select {
			case out<-mut:
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
				out<-testOrError{err:err}
				break
			}
			if !test.Usable() {
				// errors.Logf("DEBUG", "not usable")
				continue
			}
			p, err := test.Digraph(lat)
			if err != nil {
				errors.Logf("ERROR", "could not load: %v", err)
				out<-testOrError{err:err}
				break
			}
			if !sg.EmbeddedIn(p) {
				// errors.Logf("DEBUG", "didn't contain subgraph")
				continue
			}
			out<-testOrError{test:test}
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
		go gen(muts, mutsCh, done)
		wg.Add(workers)
		for w := 0; w < workers; w++ {
			go exec(&wg, mutsCh, tests)
		}
		go func() {
			wg.Wait()
			close(tests)
		}()
		te, ok := <-tests
		close(done)
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

func atMost(amt int, muts []*Mutant) []*Mutant {
	for len(muts) > amt {
		muts = percent(.9, muts)
	}
	return muts
}

func percent(p float64, slice []*Mutant) ([]*Mutant) {
	amt := int(p * float64(len(slice)))
	ten := make([]*Mutant, 0, amt)
	for i := 0; i < amt; i++ {
		var x *Mutant
		slice, x = uniform(slice)
		if x != nil {
			ten = append(ten, x)
		}
	}
	return ten
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

