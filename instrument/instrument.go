package instrument

import (
	"fmt"
	"go/ast"
	"go/parser"
	"strconv"
	"unsafe"
)

import (
	"github.com/timtadh/data-structures/errors"
	"golang.org/x/tools/go/loader"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

import ()

type instrumenter struct {
	program *loader.Program
	ssa *ssa.Program
	entry string
	seenFn map[*ssa.Function]bool
}

type bbEntry struct {
	bb *ssa.BasicBlock
	i *ssa.DebugRef
	n ast.Expr
}

func buildSSA(program *loader.Program) *ssa.Program {
	sp := ssautil.CreateProgram(program, ssa.GlobalDebug)
	sp.Build()
	return sp
}

func Instrument(entryPkgName string, program *loader.Program) (err error) {
	entry := program.Package(entryPkgName)
	if entry == nil {
		return errors.Errorf("The entry package was not found in the loaded program")
	}
	if entry.Pkg.Name() != "main" {
		return errors.Errorf("The entry package was not main")
	}
	i := &instrumenter{
		program: program,
		ssa: buildSSA(program),
		entry: entryPkgName,
		seenFn: make(map[*ssa.Function]bool),
	}
	return i.instrument()
}


func (i *instrumenter) instrument() (err error) {
	for pkgType, _ := range i.program.AllPackages {
		ssaPkg := i.ssa.Package(pkgType)
		if ssaPkg == nil {
			return errors.Errorf("Could not find pkg %v", pkgType)
		}
		err := i.functions(ssaPkg, func(fn *ssa.Function) error {
			fnAst := fn.Syntax()
			if fnAst == nil {
				// skipping synthetic function
				return nil
			}
			errors.Logf("INFO", "fn %v %T %v", fn, fnAst, fnAst)
			entries := i.basicBlockEntries(fn)
			switch n := fnAst.(type) {
			case *ast.FuncDecl:
				return i.funcDecl(fn, entries, n)
			case *ast.FuncLit:
				return i.funcLit(fn, entries, n)
			default:
				return errors.Errorf("Unexpected ast node for %v %T, expected FuncDecl or FuncLit", fn, fnAst)
			}
		})
		if err != nil {
			return err
		}
		// for _, f := range pkgInfo.Files {
		// 	errors.Logf("INFO", "f %v", f)
		// }
	}
	return nil
}

func (i instrumenter) functions(pkg *ssa.Package, do func(*ssa.Function) error) error {
	var values [10]*ssa.Value
	var process func(fn *ssa.Function) error
	process = func(fn *ssa.Function) error {
		if i.seenFn[fn] {
			return nil
		}
		i.seenFn[fn] = true
		if err := do(fn); err != nil {
			return err
		}
		return nil
	}
	for _, member := range pkg.Members {
		if fn, is := member.(*ssa.Function); is {
			if err := process(fn); err != nil {
				return err
			}
		}
	}
	for _, member := range pkg.Members {
		if fn, is := member.(*ssa.Function); is {
			for _, blk := range fn.Blocks {
				for _, inst := range blk.Instrs {
					for _, op := range inst.Operands(values[:0]) {
						if innerFn, is := (*op).(*ssa.Function); is {
							if err := process(innerFn); err != nil {
								return err
							}
						}
					}
				}
			}
		}
	}
	return nil
}

func (i instrumenter) basicBlockEntries(fn *ssa.Function) []bbEntry {
	entries := make([]bbEntry, 0, len(fn.Blocks))
	for _, blk := range fn.Blocks {
		found := false
		for _, inst := range blk.Instrs {
			if debug, is := inst.(*ssa.DebugRef); is {
				entries = append(entries, bbEntry{
					bb: blk,
					i: debug,
					n: debug.Expr,
				})
				found = true
				break
			}
		}
		if !found {
			// Some blocks do not have a clear syntactic location
			// for _, inst := range blk.Instrs {
			// 	errors.Logf("INFO", "inst %T %v %v", inst, inst, i.program.Fset.Position(inst.Pos()))
			// }
			// return errors.Errorf("Not entry for %v", blk)
			entries = append(entries, bbEntry{
				bb: blk,
				i: nil,
				n: nil,
			})
		}
	}
	return entries
}

func (i instrumenter) funcDecl(fn *ssa.Function, entries []bbEntry, n *ast.FuncDecl) error {
	if n.Body == nil {
		// this is a forward declaration
		return nil
	}
	return i.fnBody(fn, entries, n.Body)
}

func (i instrumenter) funcLit(fn *ssa.Function, entries []bbEntry, n *ast.FuncLit) error {
	if n.Body == nil {
		// this is a forward declaration
		return nil
	}
	return i.fnBody(fn, entries, n.Body)
}

func (i instrumenter) fnBody(fn *ssa.Function, entries []bbEntry, blk *ast.BlockStmt) error {
	name := fmt.Sprintf("%v.%v @ %v", fn.Package().Pkg.Name(), fn.Name(), i.program.Fset.Position(fn.Syntax().Pos()))
	for _, e := range entries {
		if e.n == nil {
			continue
		}
		if len(e.bb.Succs) == 0 {
			errors.Logf("DEBUG", "%v %v", i.program.Fset.Position(e.i.Pos()), i.program.Fset.Position(e.n.Pos()))
			eBlk, err := i.findContainingBlk(&blk.List, e.n)
			if err != nil {
				return err
			}
			if eBlk == nil {
				return errors.Errorf("could not find blk")
			}
			errors.Logf("DEBUG", "FOUND BLK")
			prin, err := i.mkPrint(fn, fmt.Sprintf("exiting %v", name))
			if err != nil {
				return err
			}
			*eBlk = append([]ast.Stmt{prin}, (*eBlk)...)
		}
	}
	prin, err := i.mkPrint(fn, name)
	if err != nil {
		return err
	}
	blk.List = append([]ast.Stmt{prin}, blk.List...)
	errors.Logf("DEBUG", "instrumented %v", fn)
	return nil
}

func (i instrumenter) findContainingBlk(blk *[]ast.Stmt, e ast.Expr) (*[]ast.Stmt, error) {
	for _, stmt := range *blk {
		errors.Logf("DEBUG", "stmt %T", stmt)
		if i.isExpr(stmt, e) {
			return blk, nil
		}
		sblk, err := i.findExpr(stmt, e)
		if err != nil {
			return nil, err
		} else if sblk != nil {
			return sblk, nil
		}
	}
	return nil, nil
}

func (i instrumenter) isExpr(stmt ast.Stmt, e ast.Expr) (bool) {
	switch n := stmt.(type) {
	case *ast.ExprStmt:
		if i.sameExpr(n.X, e) {
			return true
		}
	case *ast.ReturnStmt:
		for _, res := range n.Results {
			if i.sameExpr(res, e) {
				return true
			}
		}
	case *ast.AssignStmt:
		for _, x := range n.Lhs {
			if i.sameExpr(x, e) {
				return true
			}
		}
		for _, x := range n.Rhs {
			if i.sameExpr(x, e) {
				return true
			}
		}
	case *ast.IfStmt:
		if i.isExpr(n.Init, e) {
			return true
		}
		if i.sameExpr(n.Cond, e) {
			return true
		}
	case *ast.RangeStmt:
		if i.sameExpr(n.Key, e) {
			return true
		}
		if i.sameExpr(n.Value, e) {
			return true
		}
	case *ast.SwitchStmt:
		if i.isExpr(n.Init, e) {
			return true
		}
	case *ast.ForStmt:
		if i.isExpr(n.Init, e) {
			return true
		}
		if i.sameExpr(n.Cond, e) {
			return true
		}
		if i.isExpr(n.Post, e) {
			return true
		}
	case *ast.CaseClause:
		for _, c := range n.List {
			if i.sameExpr(c, e) {
				return true
			}
		}
	case *ast.DeclStmt:
		// no action?
	case *ast.BranchStmt, nil:
		// no action
	default:
		panic(fmt.Errorf("unexpected type %T", n))
	}
	return false
}

func (i instrumenter) findExpr(stmt ast.Stmt, e ast.Expr) (*[]ast.Stmt, error) {
	switch n := stmt.(type) {
	case *ast.BlockStmt:
		return i.findContainingBlk(&n.List, e)
	case *ast.IfStmt:
		blk, err := i.findContainingBlk(&n.Body.List, e)
		if err != nil {
			return nil, err
		}
		if blk != nil {
			return blk, nil
		}
		return i.findExpr(n.Else, e)
	case *ast.SwitchStmt:
		return i.findContainingBlk(&n.Body.List, e)
	case *ast.CaseClause:
		return i.findContainingBlk(&n.Body, e)
	case *ast.ForStmt:
		return i.findContainingBlk(&n.Body.List, e)
	case *ast.RangeStmt:
		return i.findContainingBlk(&n.Body.List, e)
	case *ast.DeclStmt:
		// no action?
	case *ast.BranchStmt, *ast.ExprStmt, *ast.ReturnStmt, *ast.AssignStmt, nil:
		// no action
	default:
		panic(fmt.Errorf("unexpected type %T", n))
	}
	return nil, nil
}

func (i instrumenter) sameExpr(a, b ast.Expr) (bool) {
	if a == nil || b == nil {
		return false
	}
	if i._sameExpr(a, b) {
		return true
	}
	switch x := a.(type) {
	case *ast.Ident, *ast.BasicLit:
		return i._sameExpr(x, b)
	case *ast.BinaryExpr:
		if i.sameExpr(x.X, b) {
			return true
		}
		if i.sameExpr(x.Y, b) {
			return true
		}
		return false
	case *ast.StarExpr:
		return i.sameExpr(x.X, b)
	case *ast.UnaryExpr:
		return i.sameExpr(x.X, b)
	case *ast.ParenExpr:
		return i.sameExpr(x.X, b)
	case *ast.CompositeLit:
		if i.sameExpr(x.Type, b) {
			return true
		}
		for _, q := range x.Elts {
			if i.sameExpr(q, b) {
				return true
			}
		}
		return false
	case *ast.CallExpr:
		if i.sameExpr(x.Fun, b) {
			return true
		}
		for _, q := range x.Args {
			if i.sameExpr(q, b) {
				return true
			}
		}
		return false
	case *ast.SelectorExpr:
		if i.sameExpr(x.X, b) {
			return true
		}
		if i.sameExpr(x.Sel, b) {
			return true
		}
		return false
	case *ast.IndexExpr:
		if i.sameExpr(x.X, b) {
			return true
		}
		if i.sameExpr(x.Index, b) {
			return true
		}
		return false
	case *ast.SliceExpr:
		if i.sameExpr(x.X, b) {
			return true
		}
		if i.sameExpr(x.Low, b) {
			return true
		}
		if i.sameExpr(x.High, b) {
			return true
		}
		if i.sameExpr(x.Max, b) {
			return true
		}
		return false
	case *ast.KeyValueExpr:
		if i.sameExpr(x.Key, b) {
			return true
		}
		if i.sameExpr(x.Value, b) {
			return true
		}
		return false
	case *ast.TypeAssertExpr:
		if i.sameExpr(x.X, b) {
			return true
		}
		if i.sameExpr(x.Type, b) {
			return true
		}
		return false
	case *ast.MapType:
		if i.sameExpr(x.Key, b) {
			return true
		}
		if i.sameExpr(x.Value, b) {
			return true
		}
		return false
	case *ast.ArrayType:
		if i.sameExpr(x.Len, b) {
			return true
		}
		if i.sameExpr(x.Elt, b) {
			return true
		}
		return false
	case *ast.FuncLit:
		// TODO: confirm this is the correct action
		return false
	default:
		panic(fmt.Errorf("unexpected type %T", x))
	}
	return false
}

func (i instrumenter) _sameExpr(a, b ast.Expr) bool {
	if a == nil || b == nil {
		return false
	}
	type intr struct {
		typ uintptr
		data uintptr
	}
	x := (*intr)(unsafe.Pointer(&a)).data
	y := (*intr)(unsafe.Pointer(&b)).data
	errors.Logf("DEBUG", "a %T %v %v ?= b %T %v %v", a, a, x, b, b, y)
	return x == y
}


func (i instrumenter) mkPrint(fn *ssa.Function, data string) (ast.Stmt, error) {
	s := fmt.Sprintf("println(%v)", strconv.Quote(data))
	e, err := parser.ParseExprFrom(i.program.Fset, i.program.Fset.File(fn.Pos()).Name(), s, parser.Mode(0))
	if err != nil {
		return nil, errors.Errorf("mkPrint (%v) error: %v", s, err)
	}
	return &ast.ExprStmt{e}, nil
}
