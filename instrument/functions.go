package instrument

import (
	"go/ast"
	"unsafe"
)

import ()


func functions(n ast.Node, do func(fn ast.Node, parent *ast.FuncDecl, count int) error) error {
	v := &funcVisitor{
		do: do,
		seen: make(map[uintptr]bool),
	}
	ast.Walk(v, n)
	return v.err
}

type funcVisitor struct {
	err error
	do func(fn ast.Node, parent *ast.FuncDecl, count int) error
	seen map[uintptr]bool
	fn *ast.FuncDecl
	count int
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
	var parent *ast.FuncDecl
	var fn ast.Node
	switch x := n.(type) {
	case *ast.FuncDecl:
		parent = x
		fn = n
	case *ast.FuncLit:
		fn = n
	}
	if fn != nil {
		p := ptr(fn)
		if !v.seen[p] {
			v.seen[p] = true
			v.count++
			err := v.do(fn, v.fn, v.count)
			if err != nil {
				v.err = err
				return nil
			}
		}
	}
	if parent != nil {
		return &funcVisitor{
			do: v.do,
			seen: v.seen,
			fn: parent,
			prev: v,
		}
	} else {
		return v
	}
}

