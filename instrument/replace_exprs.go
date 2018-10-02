package instrument

import "go/ast"

// Find all the exprs in the statement
func ReplaceExprs(stmt ast.Stmt, do func(parent ast.Node, node ast.Expr) ast.Expr) ast.Stmt {
	v := &exprVisitor{
		do: do,
	}
	return ReplacingWalk(v, nil, stmt).(ast.Stmt)
}

// A stmtVisitor visits ast.Nodes which are statements or expressions.
// it executes its "do" function on certain of them
type exprVisitor struct {
	do func(parent ast.Node, node ast.Expr) ast.Expr
}

func (v *exprVisitor) VisitKids(n ast.Node) bool {
	return true
}

func (v *exprVisitor) Replace(parent, node ast.Node) ast.Node {
	switch n := node.(type) {
	case ast.Expr:
		return v.do(parent, n).(ast.Node)
	default:
		return node
	}
}
