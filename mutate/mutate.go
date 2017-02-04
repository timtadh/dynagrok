package mutate

import (
	"fmt"
	"go/ast"
	"go/types"
	"go/token"
	"runtime"
	"math/rand"
	"os"
	"encoding/binary"
)

import (
	"github.com/timtadh/data-structures/errors"
	"golang.org/x/tools/go/loader"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
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
	ssa *ssa.Program
	entry string
	only bool
}

type counts struct {
	packages int
	functions int
	files int
	branches int
}

func (c *counts) String() string {
	return fmt.Sprintf(`counts {
	packages: %v,
	functions: %v,
	files: %v,
	branches: %v,
}`, c.packages, c.functions, c.files, c.branches)
}

func buildSSA(program *loader.Program) *ssa.Program {
	sp := ssautil.CreateProgram(program, ssa.GlobalDebug)
	sp.Build()
	return sp
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
		ssa: buildSSA(program),
		entry: entryPkgName,
		only: only,
	}
	c, err := m.count()
	if err != nil {
		return err
	}
	if c.branches <= 0 {
		errors.Logf("ERROR", "%v", c)
		return errors.Errorf("Can't mutate this program, there are no branches")
	}
	for int(float64(c.branches) * m.mutateRate) <= 0 {
		m.mutateRate *= 1.01
		if m.mutateRate > 1 {
			m.mutateRate = 1
			break
		}
	}
	errors.Logf("INFO", "mutating %v branches", float64(c.branches)*m.mutateRate)
	return m.mutate()
}

func (m *mutator) count() (c *counts, err error) {
	c = &counts{}
	for _, pkg := range m.program.AllPackages {
		if len(pkg.BuildPackage.CgoFiles) > 0 {
			continue
		}
		if excludes.ExcludedPkg(pkg.Pkg.Path()) {
			continue
		}
		if m.only && pkg.Pkg.Path() != m.entry {
			errors.Logf("DEBUG", "skipping %v", pkg.Pkg.Path())
			continue
		}
		c.packages++
		for _, fileAst := range pkg.Files {
			c.files++
			err = instrument.Functions(fileAst, func(fn ast.Node, parent *ast.FuncDecl, count int) error {
				c.functions++
				switch x := fn.(type) {
				case *ast.FuncDecl:
					if x.Body == nil {
						return nil
					}
					fnName := instrument.FuncName(pkg.Pkg, pkg.Info.TypeOf(x.Name).(*types.Signature), x)
					return m.fnBodyCount(pkg, fnName, &x.Body.List, c)
				case *ast.FuncLit:
					if x.Body == nil {
						return nil
					}
					parentName := pkg.Pkg.Path()
					if parent != nil {
						parentType := pkg.Info.TypeOf(parent.Name)
						if parentType != nil {
							parentName = instrument.FuncName(pkg.Pkg, parentType.(*types.Signature), parent)
						}
					}
					fnName := fmt.Sprintf("%v$%d", parentName, count)
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
			switch (*blk)[j].(type) {
			case *ast.IfStmt, *ast.ForStmt:
				c.branches++
			}
		}
		return nil
	})
}

func (m *mutator) mutate() (err error) {
	for _, pkg := range m.program.AllPackages {
		if len(pkg.BuildPackage.CgoFiles) > 0 {
			continue
		}
		if excludes.ExcludedPkg(pkg.Pkg.Path()) {
			continue
		}
		if m.only && pkg.Pkg.Path() != m.entry {
			errors.Logf("DEBUG", "skipping %v", pkg.Pkg.Path())
			continue
		}
		for _, fileAst := range pkg.Files {
			err = instrument.Functions(fileAst, func(fn ast.Node, parent *ast.FuncDecl, count int) error {
				switch x := fn.(type) {
				case *ast.FuncDecl:
					if x.Body == nil {
						return nil
					}
					fnName := instrument.FuncName(pkg.Pkg, pkg.Info.TypeOf(x.Name).(*types.Signature), x)
					return m.fnBodyMutate(pkg, fnName, &x.Body.List)
				case *ast.FuncLit:
					if x.Body == nil {
						return nil
					}
					parentName := pkg.Pkg.Path()
					if parent != nil {
						parentType := pkg.Info.TypeOf(parent.Name)
						if parentType != nil {
							parentName = instrument.FuncName(pkg.Pkg, parentType.(*types.Signature), parent)
						}
					}
					fnName := fmt.Sprintf("%v$%d", parentName)
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
			pos := (*blk)[j].Pos()
			switch stmt := (*blk)[j].(type) {
			case *ast.ForStmt:
				p := m.program.Fset.Position(pos)
				if rand.Float64() < m.mutateRate {
					stmt.Cond = m.negate(stmt.Cond)
					errors.Logf("DEBUG", "branch %T %v, neg %v", stmt, p, stmt.Cond)
				}
			case *ast.IfStmt:
				p := m.program.Fset.Position(pos)
				if rand.Float64() < m.mutateRate {
					stmt.Cond = m.negate(stmt.Cond)
					errors.Logf("DEBUG", "branch %T %v, neg %v", stmt, p, stmt.Cond)
				}
			}
		}
		return nil
	})
}

func (m *mutator) negate(cond ast.Expr) (ast.Expr) {
	return &ast.UnaryExpr{
		Op: token.NOT,
		X: cond,
	}
}
