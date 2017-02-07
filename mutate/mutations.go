package mutate

import (
	"fmt"
	"strings"
	"go/ast"
	"go/token"
	"go/types"
	"go/parser"
	"math/rand"
	"strconv"
)

import (
	"github.com/timtadh/data-structures/errors"
	"golang.org/x/tools/go/ast/astutil"
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
	fileAst *ast.File
}

func (m *BranchMutation) Type() string {
	return "branch-mutation"
}

func (m *BranchMutation) String() string {
	return fmt.Sprintf("%v ---> %v @ %v", m.mutator.stringNode(*m.cond), m.mutator.stringNode(m.negate()), m.p)
}

func (m *BranchMutation) Mutate() {
	(*m.cond) = m.mutate()
}

func (m *BranchMutation) mutate() ast.Expr {
	report := fmt.Sprintf("dgruntime.ReportFailBool(%v)", strconv.Quote(m.p.String()))
	pos := (*m.cond).Pos()
	failReport, err := parser.ParseExprFrom(m.mutator.program.Fset, m.mutator.program.Fset.File(pos).Name(), report, parser.Mode(0))
	if err != nil {
		panic(err)
	}
	astutil.AddImport(m.mutator.program.Fset, m.fileAst, "dgruntime")
	return &ast.BinaryExpr{
		X: failReport,
		Y: m.negate(),
		Op: token.LAND,
		OpPos: pos,
	}
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
	kind    types.BasicKind
	fileAst *ast.File
}

func (m *IncrementMutation) Type() string {
	return "increment-mutation"
}

func (m *IncrementMutation) String() string {
	return fmt.Sprintf("%v ---> %v @ %v", m.mutator.stringNode(*m.expr), m.mutator.stringNode(m.increment()), m.p)
}

func (m *IncrementMutation) Mutate() {
	(*m.expr) = m.mutate()
}

func (m *IncrementMutation) mutate() ast.Expr {
	var report string
	if m.tokType == token.INT {
		var cast string
		switch m.kind {
		case types.Int:    cast = "int"
		case types.Int8:   cast = "int8"
		case types.Int16:  cast = "int16"
		case types.Int32:  cast = "int32"
		case types.Int64:  cast = "int64"
		case types.Uint:   cast = "uint"
		case types.Uint8:  cast = "uint8"
		case types.Uint16: cast = "uint16"
		case types.Uint32: cast = "uint32"
		case types.Uint64: cast = "uint64"
		default:
			panic(fmt.Errorf("unexpected kind %v", m.kind))
		}
		report = fmt.Sprintf("%v(dgruntime.ReportFailInt(%v))", cast, strconv.Quote(m.p.String()))
	} else if m.tokType == token.FLOAT {
		var cast string
		switch m.kind {
		case types.Float32: cast = "float32"
		case types.Float64: cast = "float64"
		default:
			panic(fmt.Errorf("unexpected kind %v", m.kind))
		}
		report = fmt.Sprintf("%v(dgruntime.ReportFailFloat(%v))", cast, strconv.Quote(m.p.String()))
	} else {
		panic(fmt.Errorf("unexpected tokType %v", m.tokType))
	}
	pos := (*m.expr).Pos()
	failReport, err := parser.ParseExprFrom(m.mutator.program.Fset, m.mutator.program.Fset.File(pos).Name(), report, parser.Mode(0))
	if err != nil {
		panic(err)
	}
	astutil.AddImport(m.mutator.program.Fset, m.fileAst, "dgruntime")
	return &ast.BinaryExpr{
		X: m.increment(),
		Y: failReport,
		Op: token.ADD,
		OpPos: pos,
	}
}

func (m *IncrementMutation) increment() ast.Expr {
	pos := (*m.expr).Pos()
	return &ast.BinaryExpr{
		X: (*m.expr),
		Y: &ast.BasicLit{
			ValuePos: pos,
			Kind: m.tokType,
			Value: "1",
		},
		Op: token.ADD,
		OpPos: pos,
	}
}
