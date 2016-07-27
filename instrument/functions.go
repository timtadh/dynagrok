package instrument

import (
	"go/ast"
	"unsafe"
)

import ()


func functions(n ast.Node, do func(fn ast.Node, parent *ast.FuncDecl, count int) error) error {
	count := 0
	v := &funcVisitor{
		do: do,
		seen: make(map[uintptr]bool),
		count: &count,
	}
	ast.Walk(v, n)
	return v.err
}

type funcVisitor struct {
	err error
	do func(fn ast.Node, parent *ast.FuncDecl, count int) error
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
	case *ast.FuncLit:
		fn = n
		blk = &x.Body.List
	}
	if fn != nil {
		iv := &funcVisitor{
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
			err := v.do(fn, v.fn, *count)
			if err != nil {
				v.err = err
				return nil
			}
		}
		return nil
	}
	return v
}

