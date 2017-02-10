package analysis

import (
	"fmt"
	"go/ast"
	"go/types"
	"unsafe"
)

import (
	"golang.org/x/tools/go/loader"
	"github.com/timtadh/data-structures/errors"
)



func Functions(pkg *loader.PackageInfo, n ast.Node, do func(fn ast.Node, fnName string) error) error {
	count := 0
	v := &funcVisitor{
		do: do,
		pkg: pkg,
		seen: make(map[uintptr]bool),
		count: &count,
	}
	ast.Walk(v, n)
	return v.err
}

type funcVisitor struct {
	pkg *loader.PackageInfo
	err error
	do func(fn ast.Node, fnName string) error
	seen map[uintptr]bool
	fn *ast.FuncDecl
	count *int
	prev *funcVisitor
}

func ptr(n ast.Node) uintptr {
	type intr struct {
		typ uintptr
		data uintptr
	}
	return (*intr)(unsafe.Pointer(&n)).data
}

func (v *funcVisitor) Visit(n ast.Node) (ast.Visitor) {
	if n == nil || v.err != nil {
		return nil
	}
	var parent *ast.FuncDecl = v.fn
	var fn ast.Node
	var fnName string
	var blk *[]ast.Stmt
	var count *int = v.count
	switch x := n.(type) {
	case *ast.FuncDecl:
		if x.Body == nil {
			break
		}
		c := 0
		parent = x
		fn = n
		blk = &x.Body.List
		count = &c
		fnName = FuncName(v.pkg.Pkg, v.pkg.Info.TypeOf(x.Name).(*types.Signature), x)
	case *ast.FuncLit:
		fn = n
		blk = &x.Body.List
		parentName := v.pkg.Pkg.Path()
		if parent != nil {
			parentType := v.pkg.Info.TypeOf(parent.Name)
			if parentType != nil {
				parentName = FuncName(v.pkg.Pkg, parentType.(*types.Signature), parent)
			}
		}
		fnName = fmt.Sprintf("%v$%d", parentName, *count)
	}
	if fn != nil {
		iv := &funcVisitor{
			pkg: v.pkg,
			do: v.do,
			seen: v.seen,
			fn: parent,
			prev: v,
			count: count,
		}
		for _, stmt := range *blk {
			ast.Walk(iv, stmt)
		}
		p := ptr(fn)
		if !v.seen[p] {
			v.seen[p] = true
			(*count)++
			err := v.do(fn, fnName)
			if err != nil {
				v.err = err
				return nil
			}
		}
		return nil
	}
	return v
}

func FuncName(pkg *types.Package, fnType *types.Signature, fnAst *ast.FuncDecl) string {
	recv := fnType.Recv()
	recvName := pkg.Path()
	if recv != nil {
		recvName = fmt.Sprintf("(%v)", TypeName(pkg, recv.Type()))
	}
	return fmt.Sprintf("%v.%v", recvName, fnAst.Name.Name)
}

func TypeName(pkg *types.Package, t types.Type) string {
	switch r := t.(type) {
	case *types.Pointer:
		return fmt.Sprintf("*%v", TypeName(pkg, r.Elem()))
	case *types.Named:
		return fmt.Sprintf("%v.%v", pkg.Path(), r.Obj().Name())
	default:
		panic(errors.Errorf("unexpected recv %T", t))
	}
}
