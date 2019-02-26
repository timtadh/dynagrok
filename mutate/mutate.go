package mutate

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"go/types"
	"math/rand"
	"os"
)

import (
	"github.com/timtadh/data-structures/errors"
	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/loader"
)

import (
	"github.com/timtadh/dynagrok/analysis"
	"github.com/timtadh/dynagrok/dgruntime/excludes"
	"github.com/timtadh/dynagrok/instrument"
)

func init() {
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
	program       *loader.Program
	entry         string
	only          map[string]bool
	instrumenting bool
}

func Mutate(total int, mutate float64, only, allowedMuts map[string]bool, instrumenting bool, entryPkgName string, program *loader.Program) (mutants []*ExportedMut, err error) {
	entry := program.Package(entryPkgName)
	if entry == nil {
		return nil, errors.Errorf("The entry package was not found in the loaded program")
	}
	if entry.Pkg.Name() != "main" {
		return nil, errors.Errorf("The entry package was not main")
	}
	m := &mutator{
		program:       program,
		entry:         entryPkgName,
		only:          only,
		instrumenting: instrumenting,
	}
	muts, err := m.collect()
	if err != nil {
		return nil, err
	}
	muts = muts.Filter(allowedMuts)
	if len(muts) <= 0 {
		return nil, errors.Errorf("Can't mutate this program, there are no mutation points")
	}
	var mutations Mutations
	if total > 0 {
		mutations = muts.Sample(total)
	} else {
		for int(float64(len(muts))*mutate) <= 0 {
			mutate *= 1.01
			if mutate > 1 {
				mutate = 1
				break
			}
		}
		mutations = muts.Sample(int(float64(len(muts)) * mutate))
		errors.Logf("INFO", "mutating %v points out of %v potential points", len(mutations), len(muts))
	}
	for _, m := range mutations {
		mutants = append(mutants, m.Export())
	}
	mutations.Mutate()
	return mutants, nil
}

func (m *mutator) pkgAllowed(pkg *loader.PackageInfo) bool {
	// if pkg.Cgo {
	// 	return false
	// }
	if excludes.ExcludedPkg(pkg.Pkg.Path()) {
		return false
	}
	if len(m.only) > 0 && !m.only[pkg.Pkg.Path()] {
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
			err = analysis.Functions(pkg, fileAst, func(fn ast.Node, fnName string) error {
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
				bodyMuts, err := m.fnBodyCollect(pkg, fileAst, fnName, fn, body)
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

func (m *mutator) fnBodyCollect(pkg *loader.PackageInfo, file *ast.File, fnName string, fnAst ast.Node, fnBody *[]ast.Stmt) (Mutations, error) {
	cfg := analysis.BuildCFG(m.program.Fset, fnName, fnAst, fnBody)
	muts := make(Mutations, 0, 10)
	for _, blk := range cfg.Blocks {
		for sid, s := range blk.Stmts {
			switch stmt := (*s).(type) {
			case *ast.ForStmt:
				if stmt.Cond != nil {
					p := m.program.Fset.Position(stmt.Cond.Pos())
					muts = append(muts, &BranchMutation{
						mutator: m,
						cond:    &stmt.Cond,
						p:       p,
						fileAst: file,
						fnName:  fnName,
						bbid:    blk.Id,
						sid:     sid,
					})
				}
			case *ast.IfStmt:
				if stmt.Cond != nil {
					p := m.program.Fset.Position(stmt.Cond.Pos())
					muts = append(muts, &BranchMutation{
						mutator: m,
						cond:    &stmt.Cond,
						p:       p,
						fileAst: file,
						fnName:  fnName,
						bbid:    blk.Id,
						sid:     sid,
					})
				}
			case *ast.SendStmt:
				muts = m.exprCollect(muts, pkg, file, fnName, blk, sid, &stmt.Value)
			case *ast.ReturnStmt:
				for i := range stmt.Results {
					muts = m.exprCollect(muts, pkg, file, fnName, blk, sid, &stmt.Results[i])
				}
			case *ast.AssignStmt:
				for i := range stmt.Rhs {
					muts = m.exprCollect(muts, pkg, file, fnName, blk, sid, &stmt.Rhs[i])
				}
			}
			exprs := make([]ast.Expr, 0, 10)
			Exprs(*s, func(e ast.Expr) {
				exprs = append(exprs, e)
			})
			for _, e := range exprs {
				switch expr := e.(type) {
				case *ast.BinaryExpr:
					muts = m.exprCollect(muts, pkg, file, fnName, blk, sid, &expr.X)
					muts = m.exprCollect(muts, pkg, file, fnName, blk, sid, &expr.Y)
				case *ast.UnaryExpr:
					// cannot mutate things which are having their addresses
					// taken
					if expr.Op != token.AND {
						muts = m.exprCollect(muts, pkg, file, fnName, blk, sid, &expr.X)
					}
				case *ast.ParenExpr:
					muts = m.exprCollect(muts, pkg, file, fnName, blk, sid, &expr.X)
				case *ast.CallExpr:
					for idx := range expr.Args {
						muts = m.exprCollect(muts, pkg, file, fnName, blk, sid, &expr.Args[idx])
					}
				case *ast.IndexExpr:
					// Cannot mutate the index clause in the case of a fixed
					// size array with out extra checking.
				case *ast.KeyValueExpr:
					muts = m.exprCollect(muts, pkg, file, fnName, blk, sid, &expr.Value)
				}
			}
		}
	}
	if !m.instrumenting && pkg.Pkg.Path() == m.entry && fnName == fmt.Sprintf("%v.main", pkg.Pkg.Path()) {
		astutil.AddImport(m.program.Fset, file, "dgruntime")
		*fnBody = instrument.Insert(cfg, cfg.Blocks[0], *fnBody, 0, m.mkShutdown(fnAst.Pos()))
	}
	return muts, nil
}

func (m *mutator) exprCollect(muts Mutations, pkg *loader.PackageInfo, file *ast.File, fnName string, blk *analysis.Block, sid int, expr *ast.Expr) Mutations {
	p := m.program.Fset.Position((*expr).Pos())
	exprType := pkg.Info.TypeOf(*expr)
	switch eT := exprType.(type) {
	case *types.Basic:
		i := eT.Info()
		if (i & types.IsInteger) != 0 {
			muts = append(muts, &IncrementMutation{
				mutator: m,
				expr:    expr,
				tokType: token.INT,
				p:       p,
				kind:    eT.Kind(),
				fileAst: file,
				fnName:  fnName,
				bbid:    blk.Id,
				sid:     sid,
			})
		} else if (i & types.IsFloat) != 0 {
			muts = append(muts, &IncrementMutation{
				mutator: m,
				expr:    expr,
				tokType: token.FLOAT,
				p:       p,
				kind:    eT.Kind(),
				fileAst: file,
				fnName:  fnName,
				bbid:    blk.Id,
				sid:     sid,
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

func (m *mutator) mkShutdown(pos token.Pos) ast.Stmt {
	s := "func() { dgruntime.Shutdown() }()"
	e, err := parser.ParseExprFrom(m.program.Fset, m.program.Fset.File(pos).Name(), s, parser.Mode(0))
	if err != nil {
		panic(fmt.Errorf("mkShutdown (%v) error: %v", s, err))
	}
	return &ast.DeferStmt{Call: e.(*ast.CallExpr)}
}
