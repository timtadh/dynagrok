package instrument

import (
	"go/ast"
)

// statement walks a statement with the statementvisitor
func statement(stmt *ast.Stmt, do func(ast.Expr) error) error {
	v := &stmtVisitor{do: do}
	ast.Walk(v, *stmt)
	if v.err != nil {
		return v.err
	}
	return nil
}

// A stmtVisitor visits ast.Nodes which are statements or expressions.
// it executes its "do" function on certain of them
type stmtVisitor struct {
	err error
	do  func(ast.Expr) error
}

// Visit executes the visitor's function onto selector statements
// and returns otherwise
func (v *stmtVisitor) Visit(n ast.Node) ast.Visitor {
	if expr, ok := n.(*ast.SelectorExpr); ok {
		v.do(expr)
		return v
	} else {
		return v
	}
}
