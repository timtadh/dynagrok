package instrument

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/types"
	"go/token"
	"strconv"
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
	}
	return i.instrument()
}


func (i *instrumenter) instrument() (err error) {
	for _, pkg := range i.program.AllPackages {
		for _, fileAst := range pkg.Files {
			err = functions(fileAst, func(fn ast.Node, parent *ast.FuncDecl, count int) error {
				switch x := fn.(type) {
				case *ast.FuncDecl:
					if x.Body == nil {
						return nil
					}
					fnName := funcName(pkg.Pkg, pkg.Info.TypeOf(x.Name).(*types.Signature), x)
					return i.fnBody(pkg, fnName, fn, &x.Body.List)
				case *ast.FuncLit:
					if x.Body == nil {
						return nil
					}
					parentName := pkg.Pkg.Name()
					if parent != nil {
						parentType := pkg.Info.TypeOf(parent.Name)
						if parentType != nil {
							parentName = funcName(pkg.Pkg, parentType.(*types.Signature), parent)
						}
					}
					fnName := fmt.Sprintf("%v$%d", parentName, count)
					return i.fnBody(pkg, fnName, fn, &x.Body.List)
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

func (i instrumenter) fnBody(pkg *loader.PackageInfo, fnName string, fnAst ast.Node, fnBody *[]ast.Stmt) error {
	return blocks(fnBody, nil, func(blk *[]ast.Stmt, id int) error {
		var pos token.Pos = fnAst.Pos()
		if len(*blk) > 0 {
			pos = (*blk)[0].Pos()
		}
		*blk = insert(*blk, 0, i.mkPrint(pos, fmt.Sprintf("enter %v-blk-%d", fnName, id)))
		if _, is := (*blk)[len(*blk)-1].(*ast.ReturnStmt); !is {
			*blk = insert(*blk, len(*blk), i.mkPrint(pos, fmt.Sprintf("exit %v-blk-%d", fnName, id)))
		} else {
			*blk = insert(*blk, len(*blk)-1, i.mkPrint(pos, fmt.Sprintf("exit %v-blk-%d", fnName, id)))
		}
		return nil
	})
}

func funcName(pkg *types.Package, fnType *types.Signature, fnAst *ast.FuncDecl) string {
	recv := fnType.Recv()
	recvName := pkg.Name()
	if recv != nil {
		recvName = fmt.Sprintf("(%v)", typeName(pkg, recv.Type()))
	}
	return fmt.Sprintf("%v.%v", recvName, fnAst.Name.Name)
}

func typeName(pkg *types.Package, t types.Type) string {
	switch r := t.(type) {
	case *types.Pointer:
		return fmt.Sprintf("*%v", typeName(pkg, r.Elem()))
	case *types.Named:
		return fmt.Sprintf("%v.%v", pkg.Name(), r.Obj().Name())
	default:
		panic(errors.Errorf("unexpected recv %T", t))
	}
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

func (i instrumenter) mkPrint(pos token.Pos, data string) ast.Stmt {
	s := fmt.Sprintf("println(%v)", strconv.Quote(data))
	e, err := parser.ParseExprFrom(i.program.Fset, i.program.Fset.File(pos).Name(), s, parser.Mode(0))
	if err != nil {
		panic(fmt.Errorf("mkPrint (%v) error: %v", s, err))
	}
	return &ast.ExprStmt{e}
}

