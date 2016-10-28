package instrument

import (
	"go/ast"
	//	"reflect"
)

func statement(stmt *ast.Stmt, do func(ast.Expr) error) error {
	v := &stmtVisitor{do: do}
	ast.Walk(v, *stmt)
	if v.err != nil {
		return v.err
	}
	return nil
}

type stmtVisitor struct {
	err error
	do  func(ast.Expr) error
}

func (v *stmtVisitor) Visit(n ast.Node) ast.Visitor {
	if expr, ok := n.(*ast.SelectorExpr); ok {
		v.do(expr)
		return v
	} else {
		//println(reflect.TypeOf(n).String())
		return v
	}
}
