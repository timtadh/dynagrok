package objectstate

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
	"strings"
)

import (
	"github.com/timtadh/data-structures/errors"
	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/loader"
)

import (
	"github.com/timtadh/dynagrok/analysis"
	"github.com/timtadh/dynagrok/dgruntime/excludes"
)

type instrumenter struct {
	program     *loader.Program
	entry       string
	method      string
	currentFile *ast.File
}

func Instrument(entryPkgName string, methodName string, program *loader.Program) (err error) {
	entry := program.Package(entryPkgName)
	if entry == nil {
		return errors.Errorf("The entry package was not found in the loaded program")
	}
	if entry.Pkg.Name() != "main" {
		return errors.Errorf("The entry package was not main")
	}
	i := &instrumenter{
		program: program,
		entry:   entryPkgName,
		method:  methodName,
	}
	return i.instrument()
}

// methodCallLoc is a global map which allows blocks to be instrumented
// in two passes
var (
	methodCallLoc = make(map[token.Pos]ast.Stmt)
)

func (i *instrumenter) instrument() (err error) {
	for _, pkg := range i.program.AllPackages {
		//if len(pkg.BuildPackage.CgoFiles) > 0 {
		//	continue
		//}
		if excludes.ExcludedPkg(pkg.Pkg.Path()) {
			continue
		}
		for _, fileAst := range pkg.Files {
			i.currentFile = fileAst
			hadFunc := false
			err = analysis.Functions(pkg, fileAst, func(fn ast.Node, fnName string) error {
				if i.method != "" && !strings.Contains(fnName, i.method) {
					fmt.Printf("%v != %v\n", i.method, fnName)
					return nil
				}
				hadFunc = true
				switch x := fn.(type) {
				case *ast.FuncDecl:
					if x.Body == nil {
						return nil
					}
					return i.fnBody(pkg, fnName, fn, &x.Body.List)
				case *ast.FuncLit:
					if x.Body == nil {
						return nil
					}
					return i.fnBody(pkg, fnName, fn, &x.Body.List)
				default:
					return errors.Errorf("unexpected type %T", x)
				}
			})
			if err != nil {
				return err
			}
			// imports dgruntime package into the files that have
			// instrumentation added
			if hadFunc {
				astutil.AddImport(i.program.Fset, fileAst, "dgruntime")
			}
		}
	}
	return nil
}

// exprFuncGenerator defines a function to be called on ast.Expressions
// if relevant, the returned function inserts a dgruntime.mkMethodCall into the
// source line directly after the passed ast.Expression
func (i instrumenter) exprFuncGenerator(pkg *loader.PackageInfo, blk *[]ast.Stmt, pos token.Pos) func(ast.Expr) error {
	return func(e ast.Expr) error {
		switch expr := e.(type) {
		case *ast.SelectorExpr:
			selExpr := expr
			if ident, ok := selExpr.X.(*ast.Ident); ok {
				if ident.Name == "dgruntime" {
					return nil
				}
				for pkgName := range i.program.AllPackages {
					if pkgName.Name() == ident.Name {
						return nil
					}
				}
				for _, imports := range astutil.Imports(i.program.Fset, i.currentFile) {
					for _, importSpec := range imports {
						if importSpec.Name != nil && importSpec.Name.Name == ident.Name {
							return nil
						}
					}
				}
				callName := selExpr.Sel.Name
				stmt, _ := i.mkMethodCall(pos, ident.Name, callName)
				methodCallLoc[pos] = stmt
			}
		default:
			return errors.Errorf("Unexpected type %v, %T", e, e)
		}
		return nil
	}
}

func (i *instrumenter) fnBody(pkg *loader.PackageInfo, fnName string, fnAst ast.Node, fnBody *[]ast.Stmt) error {
	cfg := analysis.BuildCFG(i.program.Fset, fnName, fnAst, fnBody)
	if true {
		// first collect the instrumentation points (IPs)
		// build a map from lexical blocks to a sequence of IPs
		// The IPs are basic blocks from the CFG
		instr := make(map[*[]ast.Stmt][]*analysis.Block)
		for _, b := range cfg.Blocks {
			// if the block doesn't have a body don't instrument it
			if b.Body == nil {
				continue
			}
			// associate the basic block with the lexical block
			instr[b.Body] = append(instr[b.Body], b)
		}
		// Now instrument each lexical block
		for body, blks := range instr {
			// First stort the IPs (Basic Blocks) in reverse order according
			// to where they start in the lexical block. That way we can
			// safely insert the instrumentation points (by doing it in
			// reverse
			sort.Slice(blks, func(i, j int) bool {
				return blks[i].StartsAt > blks[j].StartsAt
			})
			// Now we insert object-state instrumentation
			// This second bottom-up pass performs the insertions without any
			// strange recursive issues.
			for _, b := range blks {
				for _, stmt := range b.Stmts {
					switch x := (*stmt).(type) {
					case *ast.IfStmt, *ast.ForStmt, *ast.SelectStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.RangeStmt, *ast.LabeledStmt:
					default:
						statement(&x, i.exprFuncGenerator(pkg, body, x.Pos()))
					}
				}
				for j := len(b.Stmts) - 1; j >= 0; j-- {
					var stmt ast.Stmt = *((b.Stmts)[j])
					pos := stmt.Pos()
					if stmt, has := methodCallLoc[pos]; has {
						*body = Insert(cfg, b, *body, j+1, stmt)
						delete(methodCallLoc, pos)
					}
				}
			}
		}
	}
	return nil
}

func Insert(cfg *analysis.CFG, cfgBlk *analysis.Block, blk []ast.Stmt, j int, stmt ast.Stmt) []ast.Stmt {
	if cfgBlk == nil {
		if len(blk) == 0 {
			cfgBlk = nil
		} else if j >= len(blk) {
			j = len(blk)
			cfgBlk = cfg.GetClosestBlk(len(blk)-1, blk, blk[len(blk)-1])
		} else if j < 0 {
			j = 0
			cfgBlk = cfg.GetClosestBlk(0, blk, blk[0])
		} else if j == len(blk) {
			cfgBlk = cfg.GetClosestBlk(j-1, blk, blk[j-1])
		} else {
			cfgBlk = cfg.GetClosestBlk(j, blk, blk[j])
		}
		if cfgBlk == nil {
			p := cfg.FSet.Position(stmt.Pos())
			fmt.Printf("nil cfg-blk %T %v %v \n", stmt, analysis.FmtNode(cfg.FSet, stmt), p)
			// panic(fmt.Errorf("nil cfgBlk"))
		}
	}
	if cfgBlk != nil {
		cfg.AddAllToBlk(cfgBlk, stmt)
	}
	if cap(blk) <= len(blk)+1 {
		nblk := make([]ast.Stmt, len(blk), (cap(blk)+1)*2)
		copy(nblk, blk)
		blk = nblk
	}
	blk = blk[:len(blk)+1]
	for i := len(blk) - 1; i > 0; i-- {
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

func (i instrumenter) mkMethodCall(pos token.Pos, name string, callName string) (ast.Stmt, string) {
	p := i.program.Fset.Position(pos)
	s := fmt.Sprintf("dgruntime.MethodCall(\"%s\", %s, %s)", callName, strconv.Quote(p.String()), name)
	e, err := parser.ParseExprFrom(i.program.Fset, i.program.Fset.File(pos).Name(), s, parser.Mode(0))
	if err != nil {
		panic(fmt.Errorf("mkMethodCall (%v) error: %v", s, err))
	}
	return &ast.ExprStmt{e}, s
}
