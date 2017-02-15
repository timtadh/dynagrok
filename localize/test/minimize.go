package test

import (
	"bytes"
	"math/rand"
)

import (
	"github.com/timtadh/data-structures/errors"
)

import (
	"github.com/timtadh/dynagrok/localize/lattice"
	"github.com/timtadh/dynagrok/localize/lattice/digraph"
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
	cur := t
	prev := cur
	muts := tenPercent(cur.MinimizingMuts())
	errors.Logf("DEBUG", "cur %d %d", len(cur.Case), len(muts))
	for cur != nil {
		var mf func()*Testcase
		muts, mf = uniform(muts)
		if mf == nil {
			errors.Logf("ERROR", "no more muts")
			break
		}
		mut := mf()
		err := mut.Execute()
		if err != nil {
			errors.Logf("ERROR", "could not execute: %v", err)
			return nil, err
		}
		if !mut.Usable() {
			// errors.Logf("DEBUG", "not usable")
			continue
		}
		p, err := mut.Digraph(lat)
		if err != nil {
			errors.Logf("ERROR", "could not load: %v", err)
			return nil, err
		}
		if !sg.EmbeddedIn(p) {
			// errors.Logf("DEBUG", "didn't contain subgraph")
			continue
		}
		prev = cur
		cur = mut
		muts = tenPercent(cur.MinimizingMuts())
		errors.Logf("DEBUG", "cur %d %d", len(cur.Case), len(muts))
	}
	return prev, nil
}

func (t *Testcase) Digraph(l *lattice.Lattice) (*digraph.Indices, error) {
	var buf bytes.Buffer
	_, err := buf.Write(t.Profile())
	if err != nil {
		return nil, err
	}
	return digraph.LoadDot(l.Positions, l.FnNames, l.BBIds, l.Labels, &buf)
}

func tenPercent(slice []func()*Testcase) ([]func()*Testcase) {
	amt := int(.1 * float64(len(slice)))
	ten := make([]func()*Testcase, 0, amt)
	for i := 0; i < amt; i++ {
		var x func()*Testcase
		slice, x = uniform(slice)
		ten = append(ten, x)
	}
	return ten
}

func uniform(slice []func()*Testcase) ([]func()*Testcase, func()*Testcase) {
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

