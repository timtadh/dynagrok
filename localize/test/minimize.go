package test

import (
	"math/rand"
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
	t, err = t.minimizeWith(lat, sg, func(test *Testcase)[]*Mutant {
		return test.LineEndTrimmingMuts()
	})
	if err != nil {
		return nil, err
	}
	t, err = t.minimizeWith(lat, sg, func(test *Testcase)[]*Mutant {
		return test.LineStartTrimmingMuts()
	})
	if err != nil {
		return nil, err
	}
	t, err = t.minimizeWith(lat, sg, func(test *Testcase)[]*Mutant {
		return percent(.5, test.LineBlockTrimmingMuts())
	})
	if err != nil {
		return nil, err
	}
	t, err = t.minimizeWith(lat, sg, func(test *Testcase)[]*Mutant {
		return percent(.05, test.BlockTrimmingMuts())
	})
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (t *Testcase) minimizeWith(lat *lattice.Lattice, sg *subgraph.SubGraph, f func(*Testcase)[]*Mutant) (*Testcase, error) {
	cur := t
	prev := cur
	muts := f(cur)
	errors.Logf("DEBUG", "cur %d %d", len(cur.Case), len(muts))
	for cur != nil {
		var mut *Mutant
		muts, mut = uniform(muts)
		if mut == nil {
			errors.Logf("ERROR", "no more muts")
			prev = cur
			break
		}
		test := mut.Testcase()
		err := test.Execute()
		if err != nil {
			errors.Logf("ERROR", "could not execute: %v", err)
			return nil, err
		}
		if !test.Usable() {
			// errors.Logf("DEBUG", "not usable")
			continue
		}
		p, err := test.Digraph(lat)
		if err != nil {
			errors.Logf("ERROR", "could not load: %v", err)
			return nil, err
		}
		if !sg.EmbeddedIn(p) {
			// errors.Logf("DEBUG", "didn't contain subgraph")
			continue
		}
		prev = cur
		cur = test
		muts = f(cur)
		errors.Logf("DEBUG", "cur %d %d", len(cur.Case), len(muts))
	}
	return prev, nil
}

func percent(p float64, slice []*Mutant) ([]*Mutant) {
	amt := int(p * float64(len(slice)))
	ten := make([]*Mutant, 0, amt)
	for i := 0; i < amt; i++ {
		var x *Mutant
		slice, x = uniform(slice)
		ten = append(ten, x)
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

