package mutate

import (
	"go/ast"
)

// Find mutable the exprs in the statement
func Exprs(stmt ast.Stmt, do func(ast.Expr)) {
	v := &exprVisitor{
		do: do,
	}
	ast.Walk(v, stmt)
}

// A stmtVisitor visits ast.Nodes which are statements or expressions.
// it executes its "do" function on certain of them
type exprVisitor struct {
	do func(ast.Expr)
}

// Visit executes the visitor's function onto selector statements
// and returns otherwise
func (v *exprVisitor) Visit(n ast.Node) ast.Visitor {
	switch expr := n.(type) {
	case *ast.IfStmt, *ast.ForStmt, *ast.SelectStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.RangeStmt, *ast.FuncLit:
		return nil
	case *ast.IndexExpr:
		// cannot mutate into index expressions
		return nil
	case ast.Expr:
		v.do(expr)
	}
	return v
}
