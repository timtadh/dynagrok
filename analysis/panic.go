package analysis

import (
	"go/ast"
)

import ()

// Find all the exprs in the statement
func ContainsPanic(stmt ast.Stmt) bool {
	panics := false
	err := Exprs(stmt, func(n ast.Expr) error {
		switch expr := n.(type) {
		case *ast.Ident:
			if expr.Name == "panic" {
				panics = true
			}
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
	return panics
}

// Find all the exprs in the statement
func ContainsOsExit(stmt ast.Stmt) bool {
	exits := false
	err := Exprs(stmt, func(n ast.Expr) error {
		switch expr := n.(type) {
		case *ast.SelectorExpr:
			if ident, ok := expr.X.(*ast.Ident); ok {
				if ident.Name == "os" && expr.Sel.Name == "Exit" {
					exits = true
				}
			}
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
	return exits
}

