package instrument

import (
	"fmt"
	"go/ast"
	"go/token"
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

type bbInfo struct {
	bb *ssa.BasicBlock
	ref *ssa.DebugRef
	entry *parent
	entryPos token.Pos
	exit *parent
	exitPos token.Pos
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
			errors.Logf("INFO", "fn %v", fn)
			parents := findParents(fnAst)
			blkInfos := i.basicBlocks(fn, parents)
			switch n := fnAst.(type) {
			case *ast.FuncDecl:
				return i.funcDecl(fn, blkInfos, n)
			case *ast.FuncLit:
				return i.funcLit(fn, blkInfos, n)
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

func (i instrumenter) basicBlocks(fn *ssa.Function, parents *parents) []bbInfo {
	bbInfos := make([]bbInfo, 0, len(fn.Blocks))
	errors.Logf("SSA-FN", "fn %v", fn)
	for _, blk := range fn.Blocks {
		var ref *ssa.DebugRef
		var entry token.Pos
		var exit token.Pos
		errors.Logf("SSA-BLK", "blk %v", blk.Index)
		for _, inst := range blk.Instrs {
			errors.Logf("SSA-INST", "%v", inst)
			if entry == 0 {
				entry = inst.Pos()
			}
			exit = inst.Pos()
			if debug, is := inst.(*ssa.DebugRef); is {
				if ref == nil {
					ref = debug
				}
			}
		}
		var entryParents *parent
		var exitParents *parent
		for _, v, next := parents.posParents.Range(Pos(entry), Pos(exit))(); next != nil; _, v, next = next() {
			if entryParents == nil {
				entryParents = v.(*parent)
			}
			exitParents = v.(*parent)
		}
		bbInfos = append(bbInfos, bbInfo{
			bb: blk,
			ref: ref,
			entry: entryParents,
			entryPos: entry,
			exit: exitParents,
			exitPos: exit,
		})
	}
	return bbInfos
}

func (i instrumenter) funcDecl(fn *ssa.Function, blkInfos []bbInfo, n *ast.FuncDecl) error {
	if n.Body == nil {
		// this is a forward declaration
		return nil
	}
	return i.fnBody(fn, blkInfos, n.Body)
}

func (i instrumenter) funcLit(fn *ssa.Function, blkInfos []bbInfo, n *ast.FuncLit) error {
	if n.Body == nil {
		// this is a forward declaration
		return nil
	}
	return i.fnBody(fn, blkInfos, n.Body)
}

func (i instrumenter) fnBody(fn *ssa.Function, blkInfos []bbInfo, blk *ast.BlockStmt) error {
	name := fmt.Sprintf("%v.%v @ %v", fn.Package().Pkg.Name(), fn.Name(), i.program.Fset.Position(fn.Syntax().Pos()))
	for x := range blkInfos {
		bbInfo := &blkInfos[x]
		var err error
		if len(bbInfo.bb.Preds) == 0 && len(bbInfo.bb.Succs) == 0 {
			err = i.enterBasicBlock(fn, blkInfos, x, "(single blk)")
		} else if len(bbInfo.bb.Preds) == 0 {
			err = i.enterBasicBlock(fn, blkInfos, x, "(entry blk)")
		} else if len(bbInfo.bb.Succs) == 0 {
			err = i.enterBasicBlock(fn, blkInfos, x, "(exit blk)")
		} else {
			err = i.enterBasicBlock(fn, blkInfos, x, "")
		}
		if err != nil {
			return err
		}
	}
	for x := range blkInfos {
		bbInfo := &blkInfos[x]
		var err error
		if len(bbInfo.bb.Preds) == 0 && len(bbInfo.bb.Succs) == 0 {
			err = i.exitBasicBlock(fn, blkInfos, x, "(single blk)")
		} else if len(bbInfo.bb.Preds) == 0 {
			err = i.exitBasicBlock(fn, blkInfos, x, "(entry blk)")
		} else if len(bbInfo.bb.Succs) == 0 {
			err = i.exitBasicBlock(fn, blkInfos, x, "(exit blk)")
		} else {
			err = i.exitBasicBlock(fn, blkInfos, x, "")
		}
		if err != nil {
			return err
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

func (i instrumenter) enterBasicBlock(fn *ssa.Function, bbInfos []bbInfo, idx int, prefix string) error {
	bbInfo := &bbInfos[idx]
	if bbInfo.entry == nil {
		return nil
	}
	blk := bbInfo.entry.blk
	blkIdx := findIdx(bbInfo.entry.blk, bbInfo.entry.after)
	pos := bbInfo.entryPos
	err := i.instrumentAt(fn, blk, blkIdx, pos, bbInfo.bb.Index, prefix + " entering")
	if err != nil {
		return err
	}
	for _, pred := range bbInfo.bb.Preds {
		err := i.instrumentAt(fn, blk, blkIdx, pos, pred.Index, prefix + " exiting")
		if err != nil {
			return err
		}
	}
	return nil
}

func (i instrumenter) exitBasicBlock(fn *ssa.Function, bbInfos []bbInfo, idx int, prefix string) error {
	bbInfo := &bbInfos[idx]
	if bbInfo.exit == nil {
		return nil
	}
	if len(bbInfo.bb.Succs) == 0 {
		blk := bbInfo.exit.blk
		x := len(*blk)-1
		switch (*blk)[x].(type) {
		case *ast.ReturnStmt:
		default:
			x++
		}
		err := i.instrumentAt(fn, blk, x, bbInfo.exitPos, bbInfo.bb.Index, prefix + " exiting")
		if err != nil {
			return err
		}
	}
	return nil
}

func findIdx(blk *[]ast.Stmt, prev uintptr) int {
	for i, stmt := range *blk {
		if stmtPtr(stmt) == prev {
			return i
		}
	}
	return 0
}

func (i instrumenter) instrumentAt(fn *ssa.Function, blk *[]ast.Stmt, idx int, at token.Pos, blkIndex int, prefix string) error {
	// errors.Logf("DEBUG", "FOUND BLK")
	name := fmt.Sprintf("%v.%v @ %v", fn.Package().Pkg.Name(), fn.Name(), i.program.Fset.Position(at))
	blkName := fmt.Sprintf("%v blk %v", prefix, blkIndex)
	prin, err := i.mkPrint(fn, fmt.Sprintf("%v in %v", blkName, name))
	if err != nil {
		return err
	}
	*blk = insert(*blk, idx, prin)
	return nil
}

func insert(blk []ast.Stmt, j int, stmt ast.Stmt) []ast.Stmt {
	if j > len(blk) {
		j = len(blk)
	}
	if j < 0 {
		j = 0
	}
	if cap(blk) <= len(blk) + 1 {
		nblk := make([]ast.Stmt, len(blk), (cap(blk)+1)*2)
		copy(nblk, blk)
		blk = nblk
	}
	blk = blk[:len(blk)+1]
	for i := len(blk)-1; i > 0; i-- {
		if j == i {
			blk[i] = stmt
			break
		}
		blk[i] = blk[i-1]
	}
	if j == 0 {
		blk[j] = stmt
	}
	return blk
}

func (i instrumenter) findNearestBlk(blk *[]ast.Stmt, at token.Pos) (*[]ast.Stmt, error) {
	return nil, nil
}

func (i instrumenter) findContainingBlk(blk *[]ast.Stmt, e ast.Expr) (int, *[]ast.Stmt, error) {
	for j, stmt := range *blk {
		// errors.Logf("DEBUG", "stmt %T", stmt)
		if i.isExpr(stmt, e) {
			return j, blk, nil
		}
		idx, sblk, err := i.findExpr(stmt, e)
		if err != nil {
			return 0, nil, err
		} else if sblk != nil {
			return idx, sblk, nil
		}
	}
	return 0, nil, nil
}

func (i instrumenter) isExpr(stmt ast.Stmt, e ast.Expr) (bool) {
	switch n := stmt.(type) {
	case *ast.ExprStmt:
		if i.sameExpr(n.X, e) {
			return true
		}
	case *ast.DeferStmt:
		if i.sameExpr(n.Call, e) {
			return true
		}
	case *ast.GoStmt:
		if i.sameExpr(n.Call, e) {
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
	case *ast.IncDecStmt:
		if i.sameExpr(n.X, e) {
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
	case *ast.TypeSwitchStmt:
		if i.isExpr(n.Init, e) {
			return true
		}
		if i.isExpr(n.Assign, e) {
			return true
		}
	case *ast.SendStmt:
		if i.sameExpr(n.Chan, e) {
			return true
		}
		if i.sameExpr(n.Value, e) {
			return true
		}
	case *ast.CaseClause:
		for _, c := range n.List {
			if i.sameExpr(c, e) {
				return true
			}
		}
	case *ast.CommClause:
		if i.isExpr(n.Comm, e) {
			return true
		}
		for _, c := range n.Body {
			if i.isExpr(c, e) {
				return true
			}
		}
	case *ast.DeclStmt:
		switch d := n.Decl.(type) {
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					if i.sameExpr(s.Name, e) {
						return true
					}
					return i.sameExpr(s.Type, e)
				case *ast.ValueSpec:
					for _, name := range s.Names {
						if i.sameExpr(name, e) {
							return true
						}
					}
					for _, val := range s.Values {
						if i.sameExpr(val, e) {
							return true
						}
					}
					return i.sameExpr(s.Type, e)
				default:
					panic(fmt.Errorf("unexpected type %T", s))
				}
			}
		default:
			panic(fmt.Errorf("unexpected type %T", d))
		}
	case *ast.LabeledStmt:
		return i.isExpr(n.Stmt, e)
	case *ast.EmptyStmt, *ast.SelectStmt, *ast.BranchStmt, nil:
		// no action
	default:
		panic(fmt.Errorf("unexpected type %T", n))
	}
	return false
}

func (i instrumenter) findExpr(stmt ast.Stmt, e ast.Expr) (int, *[]ast.Stmt, error) {
	switch n := stmt.(type) {
	case *ast.BlockStmt:
		return i.findContainingBlk(&n.List, e)
	case *ast.IfStmt:
		idx, blk, err := i.findContainingBlk(&n.Body.List, e)
		if err != nil {
			return 0, nil, err
		}
		if blk != nil {
			return idx, blk, nil
		}
		return i.findExpr(n.Else, e)
	case *ast.TypeSwitchStmt:
		return i.findContainingBlk(&n.Body.List, e)
	case *ast.SwitchStmt:
		return i.findContainingBlk(&n.Body.List, e)
	case *ast.SelectStmt:
		return i.findContainingBlk(&n.Body.List, e)
	case *ast.CaseClause:
		return i.findContainingBlk(&n.Body, e)
	case *ast.CommClause:
		return i.findContainingBlk(&n.Body, e)
	case *ast.ForStmt:
		return i.findContainingBlk(&n.Body.List, e)
	case *ast.RangeStmt:
		return i.findContainingBlk(&n.Body.List, e)
	case *ast.DeclStmt:
		switch d := n.Decl.(type) {
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch spec.(type) {
				case *ast.TypeSpec:
				case *ast.ValueSpec:
				default:
					panic(fmt.Errorf("unexpected type %T", spec))
				}
			}
		default:
			panic(fmt.Errorf("unexpected type %T", d))
		}
	case *ast.SendStmt, *ast.GoStmt:
		// no action
	case *ast.LabeledStmt, *ast.DeferStmt, *ast.IncDecStmt, *ast.BranchStmt, *ast.ExprStmt, *ast.ReturnStmt, *ast.AssignStmt, nil:
		// no action
	default:
		panic(fmt.Errorf("unexpected type %T", n))
	}
	return 0, nil, nil
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
	case *ast.ChanType:
		if i.sameExpr(x.Value, b) {
			return true
		}
		return false
	case *ast.InterfaceType:
		for _, f := range x.Methods.List {
			for _, name := range f.Names {
				if i.sameExpr(name, b) {
					return true
				}
			}
			if i.sameExpr(f.Tag, b) {
				return true
			}
			if i.sameExpr(f.Type, b) {
				return true
			}
		}
	case *ast.StructType:
		for _, f := range x.Fields.List {
			for _, name := range f.Names {
				if i.sameExpr(name, b) {
					return true
				}
			}
			if i.sameExpr(f.Tag, b) {
				return true
			}
			if i.sameExpr(f.Type, b) {
				return true
			}
		}
	case *ast.FuncLit, *ast.FuncType:
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
	// errors.Logf("DEBUG", "a %T %v %v ?= b %T %v %v", a, a, x, b, b, y)
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
