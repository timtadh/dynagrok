package mutate

import (
	"fmt"
	"strings"
	"go/ast"
	"go/token"
	"go/types"
	"runtime"
	"math/rand"
	"go/printer"
	"bytes"
	"os"
	"encoding/binary"
)

import (
	"github.com/timtadh/data-structures/errors"
	"golang.org/x/tools/go/loader"
)

import (
	"github.com/timtadh/dynagrok/dgruntime/excludes"
	"github.com/timtadh/dynagrok/instrument"
)


func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	if urandom, err := os.Open("/dev/urandom"); err != nil {
		panic(err)
	} else {
		seed := make([]byte, 8)
		if _, err := urandom.Read(seed); err == nil {
			rand.Seed(int64(binary.BigEndian.Uint64(seed)))
		}
		urandom.Close()
	}
}

type mutator struct {
	program *loader.Program
	entry string
	only bool
}

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

func Mutate(mutate float64, only bool, entryPkgName string, program *loader.Program) (err error) {
	entry := program.Package(entryPkgName)
	if entry == nil {
		return errors.Errorf("The entry package was not found in the loaded program")
	}
	if entry.Pkg.Name() != "main" {
		return errors.Errorf("The entry package was not main")
	}
	m := &mutator{
		program: program,
		entry: entryPkgName,
		only: only,
	}
	muts, err := m.collect()
	if err != nil {
		return err
	}
	if len(muts) <= 0 {
		return errors.Errorf("Can't mutate this program, there are no mutation points")
	}
	for int(float64(len(muts)) * mutate) <= 0 {
		mutate *= 1.01
		if mutate > 1 {
			mutate = 1
			break
		}
	}
	mutations := muts.Sample(int(float64(len(muts))*mutate))
	errors.Logf("INFO", "mutating %v points out of %v potential points", len(mutations), len(muts))
	fmt.Println(mutations)
	mutations.Mutate()
	return nil
}

func (m *mutator) pkgAllowed(pkg *loader.PackageInfo) bool {
	if len(pkg.BuildPackage.CgoFiles) > 0 {
		return false
	}
	if excludes.ExcludedPkg(pkg.Pkg.Path()) {
		return false
	}
	if m.only && pkg.Pkg.Path() != m.entry {
		return false
	}
	return true
}

func (m *mutator) collect() (muts Mutations, err error) {
	muts = make(Mutations, 0, 10)
	for _, pkg := range m.program.AllPackages {
		if !m.pkgAllowed(pkg) {
			continue
		}
		for _, fileAst := range pkg.Files {
			err = instrument.Functions(pkg, fileAst, func(fn ast.Node, fnName string) error {
				var body *[]ast.Stmt
				switch x := fn.(type) {
				case *ast.FuncDecl:
					if x.Body == nil {
						return nil
					}
					body = &x.Body.List
				case *ast.FuncLit:
					if x.Body == nil {
						return nil
					}
					body = &x.Body.List
				default:
					return errors.Errorf("unexpected type %T", x)
				}
				bodyMuts, err := m.fnBodyCollect(pkg, fnName, body)
				if err != nil {
					return err
				}
				muts = append(muts, bodyMuts...)
				return nil
			})
			if err != nil {
				return nil, err
			}
		}
	}
	return muts, nil
}

func (m *mutator) fnBodyCollect(pkg *loader.PackageInfo, fnName string, fnBody *[]ast.Stmt) (Mutations, error) {
	muts := make(Mutations, 0, 10)
	return muts, instrument.Blocks(fnBody, nil, func(blk *[]ast.Stmt, id int) error {
		for j := 0; j < len(*blk); j++ {
			p := m.program.Fset.Position((*blk)[j].Pos())
			switch stmt := (*blk)[j].(type) {
			case *ast.ForStmt:
				muts = append(muts, &BranchMutation{mutator:m, cond: &stmt.Cond, p: p })
			case *ast.IfStmt:
				muts = append(muts, &BranchMutation{mutator:m, cond: &stmt.Cond, p: p })
			}
			exprs := make([]ast.Expr, 0, 10)
			err := instrument.Exprs((*blk)[j], func(e ast.Expr) error {
				exprs = append(exprs, e)
				return nil
			})
			if err != nil {
				return err
			}
			for _, e := range exprs {
				switch expr := e.(type) {
				case *ast.BinaryExpr:
					muts = m.exprCollect(muts, pkg, &expr.X)
					muts = m.exprCollect(muts, pkg, &expr.Y)
				case *ast.UnaryExpr:
					muts = m.exprCollect(muts, pkg, &expr.X)
				}
			}
		}
		return nil
	})
}

func (m *mutator) exprCollect(muts Mutations, pkg *loader.PackageInfo, expr *ast.Expr) Mutations {
	p := m.program.Fset.Position((*expr).Pos())
	exprType := pkg.Info.TypeOf(*expr)
	switch eT := exprType.(type) {
	case *types.Basic:
		i := eT.Info()
		if (i & types.IsInteger) != 0 {
			muts = append(muts, &IncrementMutation{
				mutator: m,
				expr: expr,
				tokType: token.INT,
				p: p,
			})
		} else if (i & types.IsFloat) != 0 {
			muts = append(muts, &IncrementMutation{
				mutator: m,
				expr: expr,
				tokType: token.FLOAT,
				p: p,
			})
		}
	}
	return muts
}

func (m *mutator) stringNode(n ast.Node) string {
	var buf bytes.Buffer
	printer.Fprint(&buf, m.program.Fset, n)
	return buf.String()
}
