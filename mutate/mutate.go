package mutate

import (
	"fmt"
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
	mutateRate float64
	program *loader.Program
	entry string
	only bool
}

type counts struct {
	packages int
	functions int
	files int
	branches int
	exprs int
}

func (c *counts) String() string {
	return fmt.Sprintf(`counts {
	packages: %v,
	functions: %v,
	files: %v,
	branches: %v,
}`, c.packages, c.functions, c.files, c.branches)
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
		mutateRate: mutate,
		program: program,
		entry: entryPkgName,
		only: only,
	}
	c, err := m.count()
	if err != nil {
		return err
	}
	if (c.exprs + c.branches) <= 0 {
		errors.Logf("ERROR", "%v", c)
		return errors.Errorf("Can't mutate this program, there are no mutation points")
	}
	for int(float64(c.exprs + c.branches) * m.mutateRate) <= 0 {
		m.mutateRate *= 1.01
		if m.mutateRate > 1 {
			m.mutateRate = 1
			break
		}
	}
	errors.Logf("INFO", "mutating %v points", float64(c.exprs + c.branches)*m.mutateRate)
	return m.mutate()
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

func (m *mutator) count() (c *counts, err error) {
	c = &counts{}
	for _, pkg := range m.program.AllPackages {
		if !m.pkgAllowed(pkg) {
			continue
		}
		c.packages++
		for _, fileAst := range pkg.Files {
			c.files++
			err = instrument.Functions(pkg, fileAst, func(fn ast.Node, fnName string) error {
				c.functions++
				switch x := fn.(type) {
				case *ast.FuncDecl:
					if x.Body == nil {
						return nil
					}
					return m.fnBodyCount(pkg, fnName, &x.Body.List, c)
				case *ast.FuncLit:
					if x.Body == nil {
						return nil
					}
					return m.fnBodyCount(pkg, fnName, &x.Body.List, c)
				default:
					return errors.Errorf("unexpected type %T", x)
				}
			})
			if err != nil {
				return nil, err
			}
		}
	}
	return c, nil
}

func (m *mutator) fnBodyCount(pkg *loader.PackageInfo, fnName string, fnBody *[]ast.Stmt, c *counts) error {
	return instrument.Blocks(fnBody, nil, func(blk *[]ast.Stmt, id int) error {
		for j := 0; j < len(*blk); j++ {
			switch stmt := (*blk)[j].(type) {
			case *ast.ForStmt:
				if stmt.Cond != nil {
					c.branches++
				}
			case *ast.IfStmt:
				if stmt.Cond != nil {
					c.branches++
				}
			}
			err := instrument.Exprs((*blk)[j], func(e ast.Expr) error {
				switch expr := e.(type) {
				case *ast.BinaryExpr:
					m.exprCount(pkg, expr.X, c)
					m.exprCount(pkg, expr.Y, c)
				case *ast.UnaryExpr:
					m.exprCount(pkg, expr.X, c)
				}
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (m *mutator) mutate() (err error) {
	for _, pkg := range m.program.AllPackages {
		if !m.pkgAllowed(pkg) {
			continue
		}
		for _, fileAst := range pkg.Files {
			err = instrument.Functions(pkg, fileAst, func(fn ast.Node, fnName string) error {
				switch x := fn.(type) {
				case *ast.FuncDecl:
					if x.Body == nil {
						return nil
					}
					return m.fnBodyMutate(pkg, fnName, &x.Body.List)
				case *ast.FuncLit:
					if x.Body == nil {
						return nil
					}
					return m.fnBodyMutate(pkg, fnName, &x.Body.List)
				default:
					return errors.Errorf("unexpected type %T", x)
				}
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *mutator) fnBodyMutate(pkg *loader.PackageInfo, fnName string, fnBody *[]ast.Stmt) error {
	return instrument.Blocks(fnBody, nil, func(blk *[]ast.Stmt, id int) error {
		for j := 0; j < len(*blk); j++ {
			p := m.program.Fset.Position((*blk)[j].Pos())
			switch stmt := (*blk)[j].(type) {
			case *ast.ForStmt:
				if stmt.Cond != nil && rand.Float64() < m.mutateRate {
					c := m.negate(stmt.Cond)
					errors.Logf("DEBUG", "\n\t\tmutating %T %v -> %v @ %v", stmt, m.stringNode(stmt.Cond), m.stringNode(c), p)
					stmt.Cond = c
				}
			case *ast.IfStmt:
				if stmt.Cond != nil && rand.Float64() < m.mutateRate {
					c := m.negate(stmt.Cond)
					errors.Logf("DEBUG", "\n\t\tmutating %T %v -> %v @ %v", stmt, m.stringNode(stmt.Cond), m.stringNode(c), p)
					stmt.Cond = c
				}
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
					expr.X = m.exprMutate(pkg, expr.X)
					expr.Y = m.exprMutate(pkg, expr.Y)
				case *ast.UnaryExpr:
					expr.X = m.exprMutate(pkg, expr.X)
				}
			}
		}
		return nil
	})
}

func (m *mutator) exprCount(pkg *loader.PackageInfo, expr ast.Expr, c *counts) {
	exprType := pkg.Info.TypeOf(expr)
	switch eT := exprType.(type) {
	case *types.Basic:
		i := eT.Info()
		if (i & types.IsInteger) != 0 {
			c.exprs++
		} else if (i & types.IsFloat) != 0 {
			c.exprs++
		}
	}
}

func (m *mutator) exprMutate(pkg *loader.PackageInfo, expr ast.Expr) ast.Expr {
	p := m.program.Fset.Position(expr.Pos())
	exprType := pkg.Info.TypeOf(expr)
	switch eT := exprType.(type) {
	case *types.Basic:
		i := eT.Info()
		if (i & types.IsInteger) != 0 {
			if rand.Float64() < m.mutateRate {
				out := m.plusOne(expr, token.INT)
				errors.Logf("DEBUG", "\n\t\tmutating %v -> %v @ %v", m.stringNode(expr), m.stringNode(out), p)
				return out
			}
		} else if (i & types.IsFloat) != 0 {
			if rand.Float64() < m.mutateRate {
				out := m.plusOne(expr, token.FLOAT)
				errors.Logf("DEBUG", "\n\t\tmutating %v -> %v @ %v", m.stringNode(expr), m.stringNode(out), p)
				return out
			}
		}
	}
	return expr
}

func (m *mutator) negate(cond ast.Expr) (ast.Expr) {
	return &ast.UnaryExpr{
		Op: token.NOT,
		X: cond,
		OpPos: cond.Pos(),
	}
}

func (m *mutator) plusOne(numeric ast.Expr, typeTok token.Token) (ast.Expr) {
	return &ast.BinaryExpr{
		X: numeric,
		Y: &ast.BasicLit{
			ValuePos: numeric.Pos(),
			Kind: typeTok,
			Value: "1",
		},
		Op: token.ADD,
		OpPos: numeric.Pos(),
	}
}

func (m *mutator) stringNode(n ast.Node) string {
	var buf bytes.Buffer
	printer.Fprint(&buf, m.program.Fset, n)
	return buf.String()
}
