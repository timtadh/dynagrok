package mutate

import (
	"fmt"
	"strings"
	"go/ast"
	"go/token"
	"math/rand"
)

import (
	"github.com/timtadh/data-structures/errors"
)

import ()

type Mutation interface {
	Type() string
	String() string
	Mutate()
}

type Mutations []Mutation

func (muts Mutations) Filter(types map[string]bool) Mutations {
	valid := make(Mutations, 0, len(muts))
	for _, m := range muts {
		t := m.Type()
		if types[t] {
			valid = append(valid, m)
		}
	}
	return valid
}

func (muts Mutations) Sample(amt int) Mutations {
	if len(muts) < amt {
		panic(fmt.Errorf("Not enough mutation points, need %v have %v", amt, len(muts)))
	}
	s := make(Mutations, 0, amt)
	for _, i := range sample(amt, len(muts)) {
		s = append(s, muts[i])
	}
	return s
}

func (muts Mutations) Mutate() {
	for _, m := range muts {
		errors.Logf("INFO", "mutating:\n\t\t%v", m)
		m.Mutate()
	}
}

func (muts Mutations) String() string {
	parts := make([]string, 0, len(muts))
	for _, m := range muts {
		parts = append(parts, fmt.Sprintf("(%v)", m))
	}
	return fmt.Sprintf("[%v]", strings.Join(parts, ", "))
}

func sample(size, populationSize int) (sample []int) {
	if size >= populationSize {
		return srange(populationSize)
	}
	pop := func(items []int) ([]int, int) {
		i := rand.Intn(len(items))
		item := items[i]
		copy(items[i:], items[i+1:])
		return items[:len(items)-1], item
	}
	items := srange(populationSize)
	sample = make([]int, 0, size+1)
	for i := 0; i < size; i++ {
		var item int
		items, item = pop(items)
		sample = append(sample, item)
	}
	return sample
}

func srange(size int) []int {
	sample := make([]int, 0, size+1)
	for i := 0; i < size; i++ {
		sample = append(sample, i)
	}
	return sample
}

type BranchMutation struct {
	mutator *mutator
	cond *ast.Expr
	p    token.Position
}

func (m *BranchMutation) Type() string {
	return "branch-mutation"
}

func (m *BranchMutation) String() string {
	return fmt.Sprintf("%v ---> %v @ %v", m.mutator.stringNode(*m.cond), m.mutator.stringNode(m.negate()), m.p)
}

func (m *BranchMutation) Mutate() {
	(*m.cond) = m.negate()
}

func (m *BranchMutation) negate() ast.Expr {
	return &ast.UnaryExpr{
		Op: token.NOT,
		X: *m.cond,
		OpPos: (*m.cond).Pos(),
	}
}

type IncrementMutation struct {
	mutator *mutator
	expr    *ast.Expr
	tokType token.Token
	p       token.Position
}

func (m *IncrementMutation) Type() string {
	return "increment-mutation"
}

func (m *IncrementMutation) String() string {
	return fmt.Sprintf("%v ---> %v @ %v", m.mutator.stringNode(*m.expr), m.mutator.stringNode(m.increment()), m.p)
}

func (m *IncrementMutation) Mutate() {
	(*m.expr) = m.increment()
}

func (m *IncrementMutation) increment() ast.Expr {
	return &ast.BinaryExpr{
		X: (*m.expr),
		Y: &ast.BasicLit{
			ValuePos: (*m.expr).Pos(),
			Kind: m.tokType,
			Value: "1",
		},
		Op: token.ADD,
		OpPos: (*m.expr).Pos(),
	}
}

