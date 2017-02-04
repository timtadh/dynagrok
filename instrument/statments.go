package instrument

import (
	"go/ast"
)

// statement walks a statement with the statementvisitor
func Statement(stmt ast.Stmt, do func(ast.Expr) error) error {
	if _, ok := stmt.(*ast.ReturnStmt); ok {
		return nil
	}
	v := &stmtVisitor{do: do}
	ast.Walk(v, stmt)
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
	switch expr := n.(type) {
	case *ast.SelectorExpr:
		v.do(expr)
		return v
	case *ast.ForStmt, *ast.SwitchStmt, *ast.FuncLit:
		return nil
	}
	return v
}
