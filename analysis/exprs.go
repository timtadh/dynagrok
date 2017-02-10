package analysis

import (
	"go/ast"
)

import ()

// Find all the exprs in the statement
func Exprs(stmt ast.Stmt, do func(ast.Expr) error) error {
	v := &exprVisitor{
		do: do,
	}
	ast.Walk(v, stmt)
	if v.err != nil {
		return v.err
	}
	return nil
}

// A stmtVisitor visits ast.Nodes which are statements or expressions.
// it executes its "do" function on certain of them
type exprVisitor struct {
	err error
	do  func(ast.Expr) error
}

// Visit executes the visitor's function onto selector statements
// and returns otherwise
func (v *exprVisitor) Visit(n ast.Node) ast.Visitor {
	switch expr := n.(type) {
	case *ast.LabeledStmt:
		v.err = Exprs(expr.Stmt, v.do)
		return nil
	case *ast.IfStmt:
		return nil
	case *ast.ForStmt:
		return nil
	case *ast.RangeStmt:
		return nil
	case *ast.SelectStmt:
		return nil
	case *ast.TypeSwitchStmt:
		return nil
	case *ast.SwitchStmt:
		return nil
	case *ast.CaseClause:
		return nil
	case *ast.CommClause:
		return nil
	case *ast.FuncLit:
		return nil
	case ast.Expr:
		err := v.do(expr)
		if err != nil {
			v.err = err
			return nil
		}
	}
	return v
}

// Find all the exprs in the statement from a stmt in a basic block
func blkExprs(n ast.Node, do func(ast.Expr)) {
	v := &blkExprVisitor{
		do: do,
	}
	ast.Walk(v, n)
}

// A stmtVisitor visits ast.Nodes which are statements or expressions.
// it executes its "do" function on certain of them
type blkExprVisitor struct {
	do  func(ast.Expr)
}

// Visit executes the visitor's function onto selector statements
// and returns otherwise
func (v *blkExprVisitor) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.IfStmt:
		blkExprs(n.Cond, v.do)
		return nil
	case *ast.ForStmt:
		if n.Cond != nil {
			blkExprs(n.Cond, v.do)
		}
		return nil
	case *ast.RangeStmt:
		if n.Key != nil {
			blkExprs(n.Key, v.do)
		}
		if n.Value != nil {
			blkExprs(n.Value, v.do)
		}
		if n.X != nil {
			blkExprs(n.X, v.do)
		}
		return nil
	case *ast.SelectStmt:
		return nil
	case *ast.TypeSwitchStmt:
		blkExprs(n.Assign, v.do)
		return nil
	case *ast.SwitchStmt:
		if n.Tag != nil {
			blkExprs(n.Tag, v.do)
		}
		return nil
	case *ast.CaseClause:
		for _, c := range n.List {
			blkExprs(c, v.do)
		}
		return nil
	case *ast.CommClause:
		return nil
	case *ast.FuncLit:
		return nil
	case ast.Expr:
		v.do(n)
	}
	return v
}
