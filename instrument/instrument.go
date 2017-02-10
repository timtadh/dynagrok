// instrument walks a source program's AST to insert instrumentation
// statements.

package instrument

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	//	"go/types"
	"strconv"
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
	currentFile *ast.File
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
		entry:   entryPkgName,
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
func (i instrumenter) exprFuncGenerator(pkg *loader.PackageInfo, blk *[]ast.Stmt, j int, pos token.Pos) func(ast.Expr) error {
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
	if true {
		// This unnamed function "instruments" a block.
		// First, it calls mkEnterBlk in the first line of the block,
		// unless it's a top-level block, in which case mkEnterFunc has
		// already been called.
		// Then, it iterates through the statements of the block, marking
		// which points are re-entry points - the target of a goto, the end
		// of a loop, etc.
		err := analysis.Blocks(fnBody, nil, func(blk *[]ast.Stmt, id int) error {
			var pos token.Pos = fnAst.Pos()
			if len(*blk) > 0 {
				pos = (*blk)[0].Pos()
			}
			// Enter goes at the beginning unless we're a function body
			if id != 0 {
				*blk = Insert(*blk, 0, i.mkEnterBlk(pos, id))
			}
			for j := 0; j < len(*blk)-1; j++ {
				if j+1 < len(*blk) {
					pos = (*blk)[j+1].Pos()
				} else { // can this ever happen?
					pos = (*blk)[j].Pos()
				}
				switch stmt := (*blk)[j].(type) {
				case *ast.BranchStmt:
					// *blk = Insert(*blk, j, i.mkPrint(pos, fmt.Sprintf("exit-blk:\t %d\t of %v", id, fnName)))
					// j++
				case *ast.IfStmt, *ast.ForStmt, *ast.SelectStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.RangeStmt:
					// In the case of a statement that defines its own block,
					// make sure to insert a re-entry after it.
					// (Why 2+j+1)? Do I break this with my MethodCall calls?
					*blk = Insert(*blk, j+1, i.mkRe_enterBlk(pos, id, 2+j+1))
					j++ // skip over the line we just created
				case *ast.LabeledStmt:
					switch stmt.Stmt.(type) {
					case *ast.ForStmt, *ast.SwitchStmt, *ast.SelectStmt, *ast.TypeSwitchStmt, *ast.RangeStmt:
						*blk = Insert(*blk, j+1, i.mkRe_enterBlk(pos, id, 2+j+1))
					default:
						// errors.Logf("DEBUG", "label stmt %T %T in %v", stmt.Stmt, (*blk)[j+1], fnName)
						// For labeled statements (targets of goto, break, etc.)
						// control re-enters and then executes the line of the
						// labeled statement. So we should mark the re-entry as
						// the line above the LabeledStmt
						*blk = Insert(*blk, j+1, stmt.Stmt)
						stmt.Stmt = i.mkRe_enterBlk(pos, id, 2+j)
					}
					// TODO : see if this placement of the call to statement()
					// makes sense.
				default:
					statement(&stmt, i.exprFuncGenerator(pkg, blk, j, stmt.Pos()))
				}
			}
			// The above leaves out the last line of the block, so we complete
			// it below.
			if len(*blk) > 0 {
				switch stmt := (*blk)[len(*blk)-1].(type) {
				case *ast.IfStmt, *ast.ForStmt, *ast.SelectStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.RangeStmt, *ast.LabeledStmt:
				default:
					statement(&stmt, i.exprFuncGenerator(pkg, blk, len(*blk)-1, stmt.Pos()))
				}
			}

			// This second bottom-up pass performs the insertions without any
			// strange recursive issues.
			for j := len(*blk) - 1; j >= 0; j-- {
				pos = (*blk)[j].Pos()
				if stmt, has := methodCallLoc[pos]; has {
					*blk = Insert(*blk, j+1, stmt)
				}
			}
			for j := 0; j < len(*blk); j++ {
				pos = (*blk)[j].Pos()
				switch stmt := (*blk)[j].(type) {
				default:
					err := analysis.Exprs(stmt, func(expr ast.Expr) error {
						switch e := expr.(type) {
						case *ast.SelectorExpr:
							if ident, ok := e.X.(*ast.Ident); ok {
								if ident.Name == "os" && e.Sel.Name == "Exit" {
									*blk = Insert(*blk, j, i.mkShutdownNow(pos))
									j++
								}
							}
						}
						return nil
					})
					if err != nil {
						return err
					}
				}
			}
			return nil
		})
		if err != nil {
			return nil
		}
	}
	*fnBody = Insert(*fnBody, 0, i.mkEnterFunc(fnAst.Pos(), fnName))
	*fnBody = Insert(*fnBody, 1, i.mkExitFunc(fnAst.Pos(), fnName))
	if pkg.Pkg.Path() == i.entry && fnName == fmt.Sprintf("%v.main", pkg.Pkg.Path()) {
		*fnBody = Insert(*fnBody, 0, i.mkShutdown(fnAst.Pos()))
	}
	return nil
}

//<<<<<<< HEAD
//func FuncName(pkg *types.Package, fnType *types.Signature, fnAst *ast.FuncDecl) string {
//	recv := fnType.Recv()
//	recvName := pkg.Path()
//	if recv != nil {
//		recvName = fmt.Sprintf("(%v)", TypeName(pkg, recv.Type()))
//	}
//	return fmt.Sprintf("%v.%v", recvName, fnAst.Name.Name)
//}
//
//func TypeName(pkg *types.Package, t types.Type) string {
//	switch r := t.(type) {
//	case *types.Pointer:
//		return fmt.Sprintf("*%v", TypeName(pkg, r.Elem()))
//	case *types.Named:
//		return fmt.Sprintf("%v.%v", pkg.Path(), r.Obj().Name())
//	default:
//		panic(errors.Errorf("unexpected recv %T", t))
//	}
//}
//
//// insert inserts a statement stmt to block blk at index j
//=======
//>>>>>>> master
func Insert(blk []ast.Stmt, j int, stmt ast.Stmt) []ast.Stmt {
	if j > len(blk) {
		j = len(blk)
	}
	if j < 0 {
		j = 0
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

func (i *instrumenter) mkPrint(pos token.Pos, data string) ast.Stmt {
	s := fmt.Sprintf("dgruntime.Println(%v)", strconv.Quote(data))
	e, err := parser.ParseExprFrom(i.program.Fset, i.program.Fset.File(pos).Name(), s, parser.Mode(0))
	if err != nil {
		panic(fmt.Errorf("mkPrint (%v) error: %v", s, err))
	}
	return &ast.ExprStmt{e}
}

func (i *instrumenter) mkDeferPrint(pos token.Pos, data string) ast.Stmt {
	s := fmt.Sprintf("func() { dgruntime.Println(%v) }()", strconv.Quote(data))
	e, err := parser.ParseExprFrom(i.program.Fset, i.program.Fset.File(pos).Name(), s, parser.Mode(0))
	if err != nil {
		panic(fmt.Errorf("mkDeferPrint (%v) error: %v", s, err))
	}
	return &ast.DeferStmt{Call: e.(*ast.CallExpr)}
}

func (i *instrumenter) mkEnterFunc(pos token.Pos, name string) ast.Stmt {
	p := i.program.Fset.Position(pos)
	s := fmt.Sprintf("dgruntime.EnterFunc(%v, %v)", strconv.Quote(name), strconv.Quote(p.String()))
	e, err := parser.ParseExprFrom(i.program.Fset, i.program.Fset.File(pos).Name(), s, parser.Mode(0))
	if err != nil {
		panic(fmt.Errorf("mkEnterFunc (%v) error: %v", s, err))
	}
	return &ast.ExprStmt{e}
}

func (i *instrumenter) mkExitFunc(pos token.Pos, name string) ast.Stmt {
	s := fmt.Sprintf("func() { dgruntime.ExitFunc(%v) }()", strconv.Quote(name))
	e, err := parser.ParseExprFrom(i.program.Fset, i.program.Fset.File(pos).Name(), s, parser.Mode(0))
	if err != nil {
		panic(fmt.Errorf("mkExitFunc (%v) error: %v", s, err))
	}
	return &ast.DeferStmt{Call: e.(*ast.CallExpr)}
}

func (i *instrumenter) mkShutdown(pos token.Pos) ast.Stmt {
	s := "func() { dgruntime.Shutdown() }()"
	e, err := parser.ParseExprFrom(i.program.Fset, i.program.Fset.File(pos).Name(), s, parser.Mode(0))
	if err != nil {
		panic(fmt.Errorf("mkShutdown (%v) error: %v", s, err))
	}
	return &ast.DeferStmt{Call: e.(*ast.CallExpr)}
}

func (i *instrumenter) mkShutdownNow(pos token.Pos) ast.Stmt {
	s := "dgruntime.Shutdown()"
	e, err := parser.ParseExprFrom(i.program.Fset, i.program.Fset.File(pos).Name(), s, parser.Mode(0))
	if err != nil {
		panic(fmt.Errorf("mkShutdown (%v) error: %v", s, err))
	}
	return &ast.ExprStmt{e}
}

func (i *instrumenter) mkEnterBlk(pos token.Pos, blkid int) ast.Stmt {
	p := i.program.Fset.Position(pos)
	s := fmt.Sprintf("dgruntime.EnterBlk(%d, %v)", blkid, strconv.Quote(p.String()))
	e, err := parser.ParseExprFrom(i.program.Fset, i.program.Fset.File(pos).Name(), s, parser.Mode(0))
	if err != nil {
		panic(fmt.Errorf("mkEnterBlk (%v) error: %v", s, err))
	}
	return &ast.ExprStmt{e}
}

func (i *instrumenter) mkRe_enterBlk(pos token.Pos, blkid, at int) ast.Stmt {
	p := i.program.Fset.Position(pos)
	s := fmt.Sprintf("dgruntime.Re_enterBlk(%d, %d, %v) // %v", blkid, at, strconv.Quote(p.String()))
	e, err := parser.ParseExprFrom(i.program.Fset, i.program.Fset.File(pos).Name(), s, parser.Mode(0))
	if err != nil {
		panic(fmt.Errorf("mkEnterBlk (%v) error: %v", s, err))
	}
	return &ast.ExprStmt{e}
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
