package instrument

import (
	"go/ast"
	"go/token"
	"unsafe"
)

import (
	"github.com/timtadh/data-structures/tree/bptree"
	"github.com/timtadh/data-structures/types"
)

type Pos token.Pos

func (p Pos) Equals(other types.Equatable) bool {
	if o, ok := other.(Pos); ok {
		return p == o
	} else {
		return false
	}
}

func (p Pos) Less(other types.Sortable) bool {
	if o, ok := other.(Pos); ok {
		return p < o
	} else {
		return false
	}
}

func (p Pos) Hash() int {
	return int(p)
}



type parents struct {
	posParents *bptree.BpTree
}

type parent struct {
	blk *[]ast.Stmt
	after uintptr
}

type parentVisitor struct {
	posParents *bptree.BpTree
	v *parentVisitor
	parent *[]ast.Stmt
	prev uintptr
}


func findParents(n ast.Node) *parents {
	v := &parentVisitor{
		posParents: bptree.NewBpTree(16),
	}
	ast.Walk(v, n)
	return &parents{
		posParents: v.posParents,
	}
}

func stmtPtr(n ast.Stmt) uintptr {
	type intr struct {
		typ uintptr
		data uintptr
	}
	return (*intr)(unsafe.Pointer(&n)).data
}

func (p *parentVisitor) Visit(n ast.Node) (ast.Visitor) {
	if n == nil {
		return nil
	}
	if p.parent != nil {
		p.posParents.Add(Pos(n.Pos()), &parent{p.parent, p.prev})
		p.posParents.Add(Pos(n.End()), &parent{p.parent, p.prev})
	}
	parent := p.parent
	prev := p.prev
	if stmt, is := n.(ast.Stmt); is && p.parent != nil {
		for _, s := range *p.parent {
			if stmtPtr(s) == stmtPtr(stmt) {
				prev = stmtPtr(stmt)
				break
			}
		}
	}
	switch x := n.(type) {
	case *ast.BlockStmt:
		parent = &x.List
		prev = 0
	case *ast.CommClause:
		parent = &x.Body
		prev = 0
	case *ast.CaseClause:
		parent = &x.Body
		prev = 0
	}
	return &parentVisitor{
		posParents: p.posParents,
		v: p,
		parent: parent,
		prev: prev,
	}
}

