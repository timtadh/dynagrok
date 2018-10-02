package instrument

import (
	"fmt"
	"go/ast"
)

type Visitor interface {
	VisitKids(ast.Node) bool                // called in pre-order
	Replace(parent, node ast.Node) ast.Node // called in post-order
}

func walkIdentList(v Visitor, parent ast.Node, list []*ast.Ident) []*ast.Ident {
	replacement := make([]*ast.Ident, 0, len(list))
	for _, x := range list {
		replacement = append(replacement, ReplacingWalk(v, parent, x).(*ast.Ident))
	}
	return replacement
}

func walkExprList(v Visitor, parent ast.Node, list []ast.Expr) []ast.Expr {
	replacement := make([]ast.Expr, 0, len(list))
	for _, x := range list {
		replacement = append(replacement, ReplacingWalk(v, parent, x).(ast.Expr))
	}
	return replacement
}

func walkStmtList(v Visitor, parent ast.Node, list []ast.Stmt) []ast.Stmt {
	replacement := make([]ast.Stmt, 0, len(list))
	for _, x := range list {
		replacement = append(replacement, ReplacingWalk(v, parent, x).(ast.Stmt))
	}
	return replacement
}

func walkDeclList(v Visitor, parent ast.Node, list []ast.Decl) []ast.Decl {
	replacement := make([]ast.Decl, 0, len(list))
	for _, x := range list {
		replacement = append(replacement, ReplacingWalk(v, parent, x).(ast.Decl))
	}
	return replacement
}

func ReplacingWalk(v Visitor, parent, node ast.Node) (replacement ast.Node) {
	if !v.VisitKids(node) {
		return v.Replace(parent, node)
	}

	// walk children
	// (the order of the cases matches the order
	// of the corresponding node types in ast.go)
	switch n := node.(type) {
	// Comments and fields
	case *ast.Comment:
		// nothing to do

	case *ast.CommentGroup:
		for i, c := range n.List {
			n.List[i] = ReplacingWalk(v, n, c).(*ast.Comment)
		}

	case *ast.Field:
		if n.Doc != nil {
			n.Doc = ReplacingWalk(v, n, n.Doc).(*ast.CommentGroup)
		}
		n.Names = walkIdentList(v, n, n.Names)
		n.Type = ReplacingWalk(v, n, n.Type).(ast.Expr)
		if n.Tag != nil {
			n.Tag = ReplacingWalk(v, n, n.Tag).(*ast.BasicLit)
		}
		if n.Comment != nil {
			n.Comment = ReplacingWalk(v, n, n.Comment).(*ast.CommentGroup)
		}

	case *ast.FieldList:
		for i, f := range n.List {
			n.List[i] = ReplacingWalk(v, n, f).(*ast.Field)
		}

	// Expressions
	case *ast.BadExpr, *ast.Ident, *ast.BasicLit:
		// nothing to do

	case *ast.Ellipsis:
		if n.Elt != nil {
			n.Elt = ReplacingWalk(v, n, n.Elt).(ast.Expr)
		}

	case *ast.FuncLit:
		n.Type = ReplacingWalk(v, n, n.Type).(*ast.FuncType)
		n.Body = ReplacingWalk(v, n, n.Body).(*ast.BlockStmt)

	case *ast.CompositeLit:
		if n.Type != nil {
			n.Type = ReplacingWalk(v, n, n.Type).(ast.Expr)
		}
		n.Elts = walkExprList(v, n, n.Elts)

	case *ast.ParenExpr:
		n.X = ReplacingWalk(v, n, n.X).(ast.Expr)

	case *ast.SelectorExpr:
		n.X = ReplacingWalk(v, n, n.X).(ast.Expr)
		n.Sel = ReplacingWalk(v, n, n.Sel).(*ast.Ident)

	case *ast.IndexExpr:
		n.X = ReplacingWalk(v, n, n.X).(ast.Expr)
		n.Index = ReplacingWalk(v, n, n.Index).(ast.Expr)

	case *ast.SliceExpr:
		n.X = ReplacingWalk(v, n, n.X).(ast.Expr)
		if n.Low != nil {
			n.Low = ReplacingWalk(v, n, n.Low).(ast.Expr)
		}
		if n.High != nil {
			n.High = ReplacingWalk(v, n, n.High).(ast.Expr)
		}
		if n.Max != nil {
			n.Max = ReplacingWalk(v, n, n.Max).(ast.Expr)
		}

	case *ast.TypeAssertExpr:
		n.X = ReplacingWalk(v, n, n.X).(ast.Expr)
		if n.Type != nil {
			n.Type = ReplacingWalk(v, n, n.Type).(ast.Expr)
		}

	case *ast.CallExpr:
		n.Fun = ReplacingWalk(v, n, n.Fun).(ast.Expr)
		n.Args = walkExprList(v, n, n.Args)

	case *ast.StarExpr:
		n.X = ReplacingWalk(v, n, n.X).(ast.Expr)

	case *ast.UnaryExpr:
		n.X = ReplacingWalk(v, n, n.X).(ast.Expr)

	case *ast.BinaryExpr:
		n.X = ReplacingWalk(v, n, n.X).(ast.Expr)
		n.Y = ReplacingWalk(v, n, n.Y).(ast.Expr)

	case *ast.KeyValueExpr:
		n.Key = ReplacingWalk(v, n, n.Key).(ast.Expr)
		n.Value = ReplacingWalk(v, n, n.Value).(ast.Expr)

	// Types
	case *ast.ArrayType:
		if n.Len != nil {
			n.Len = ReplacingWalk(v, n, n.Len).(ast.Expr)
		}
		n.Elt = ReplacingWalk(v, n, n.Elt).(ast.Expr)

	case *ast.StructType:
		n.Fields = ReplacingWalk(v, n, n.Fields).(*ast.FieldList)

	case *ast.FuncType:
		if n.Params != nil {
			n.Params = ReplacingWalk(v, n, n.Params).(*ast.FieldList)
		}
		if n.Results != nil {
			n.Results = ReplacingWalk(v, n, n.Results).(*ast.FieldList)
		}

	case *ast.InterfaceType:
		n.Methods = ReplacingWalk(v, n, n.Methods).(*ast.FieldList)

	case *ast.MapType:
		n.Key = ReplacingWalk(v, n, n.Key).(ast.Expr)
		n.Value = ReplacingWalk(v, n, n.Value).(ast.Expr)

	case *ast.ChanType:
		n.Value = ReplacingWalk(v, n, n.Value).(ast.Expr)

	// Statements
	case *ast.BadStmt:
		// nothing to do

	case *ast.DeclStmt:
		n.Decl = ReplacingWalk(v, n, n.Decl).(ast.Decl)

	case *ast.EmptyStmt:
		// nothing to do

	case *ast.LabeledStmt:
		n.Label = ReplacingWalk(v, n, n.Label).(*ast.Ident)
		n.Stmt = ReplacingWalk(v, n, n.Stmt).(ast.Stmt)

	case *ast.ExprStmt:
		n.X = ReplacingWalk(v, n, n.X).(ast.Expr)

	case *ast.SendStmt:
		n.Chan = ReplacingWalk(v, n, n.Chan).(ast.Expr)
		n.Value = ReplacingWalk(v, n, n.Value).(ast.Expr)

	case *ast.IncDecStmt:
		n.X = ReplacingWalk(v, n, n.X).(ast.Expr)

	case *ast.AssignStmt:
		n.Lhs = walkExprList(v, n, n.Lhs)
		n.Rhs = walkExprList(v, n, n.Rhs)

	case *ast.GoStmt:
		n.Call = ReplacingWalk(v, n, n.Call).(*ast.CallExpr)

	case *ast.DeferStmt:
		n.Call = ReplacingWalk(v, n, n.Call).(*ast.CallExpr)

	case *ast.ReturnStmt:
		n.Results = walkExprList(v, n, n.Results)

	case *ast.BranchStmt:
		if n.Label != nil {
			n.Label = ReplacingWalk(v, n, n.Label).(*ast.Ident)
		}

	case *ast.BlockStmt:
		n.List = walkStmtList(v, n, n.List)

	case *ast.IfStmt:
		if n.Init != nil {
			n.Init = ReplacingWalk(v, n, n.Init).(ast.Stmt)
		}
		n.Cond = ReplacingWalk(v, n, n.Cond).(ast.Expr)
		n.Body = ReplacingWalk(v, n, n.Body).(*ast.BlockStmt)
		if n.Else != nil {
			n.Else = ReplacingWalk(v, n, n.Else).(ast.Stmt)
		}

	case *ast.CaseClause:
		n.List = walkExprList(v, n, n.List)
		n.Body = walkStmtList(v, n, n.Body)

	case *ast.SwitchStmt:
		if n.Init != nil {
			n.Init = ReplacingWalk(v, n, n.Init).(ast.Stmt)
		}
		if n.Tag != nil {
			n.Tag = ReplacingWalk(v, n, n.Tag).(ast.Expr)
		}
		n.Body = ReplacingWalk(v, n, n.Body).(*ast.BlockStmt)

	case *ast.TypeSwitchStmt:
		if n.Init != nil {
			n.Init = ReplacingWalk(v, n, n.Init).(ast.Stmt)
		}
		n.Assign = ReplacingWalk(v, n, n.Assign).(ast.Stmt)
		n.Body = ReplacingWalk(v, n, n.Body).(*ast.BlockStmt)

	case *ast.CommClause:
		if n.Comm != nil {
			n.Comm = ReplacingWalk(v, n, n.Comm).(ast.Stmt)
		}
		n.Body = walkStmtList(v, n, n.Body)

	case *ast.SelectStmt:
		n.Body = ReplacingWalk(v, n, n.Body).(*ast.BlockStmt)

	case *ast.ForStmt:
		if n.Init != nil {
			n.Init = ReplacingWalk(v, n, n.Init).(ast.Stmt)
		}
		if n.Cond != nil {
			n.Cond = ReplacingWalk(v, n, n.Cond).(ast.Expr)
		}
		if n.Post != nil {
			n.Post = ReplacingWalk(v, n, n.Post).(ast.Stmt)
		}
		n.Body = ReplacingWalk(v, n, n.Body).(*ast.BlockStmt)

	case *ast.RangeStmt:
		if n.Key != nil {
			n.Key = ReplacingWalk(v, n, n.Key).(ast.Expr)
		}
		if n.Value != nil {
			n.Value = ReplacingWalk(v, n, n.Value).(ast.Expr)
		}
		n.X = ReplacingWalk(v, n, n.X).(ast.Expr)
		n.Body = ReplacingWalk(v, n, n.Body).(*ast.BlockStmt)

	// Declarations
	case *ast.ImportSpec:
		if n.Doc != nil {
			n.Doc = ReplacingWalk(v, n, n.Doc).(*ast.CommentGroup)
		}
		if n.Name != nil {
			n.Name = ReplacingWalk(v, n, n.Name).(*ast.Ident)
		}
		n.Path = ReplacingWalk(v, n, n.Path).(*ast.BasicLit)
		if n.Comment != nil {
			n.Comment = ReplacingWalk(v, n, n.Comment).(*ast.CommentGroup)
		}

	case *ast.ValueSpec:
		if n.Doc != nil {
			n.Doc = ReplacingWalk(v, n, n.Doc).(*ast.CommentGroup)
		}
		n.Names = walkIdentList(v, n, n.Names)
		if n.Type != nil {
			n.Type = ReplacingWalk(v, n, n.Type).(ast.Expr)
		}
		n.Values = walkExprList(v, n, n.Values)
		if n.Comment != nil {
			n.Comment = ReplacingWalk(v, n, n.Comment).(*ast.CommentGroup)
		}

	case *ast.TypeSpec:
		if n.Doc != nil {
			n.Doc = ReplacingWalk(v, n, n.Doc).(*ast.CommentGroup)
		}
		n.Name = ReplacingWalk(v, n, n.Name).(*ast.Ident)
		n.Type = ReplacingWalk(v, n, n.Type).(ast.Expr)
		if n.Comment != nil {
			n.Comment = ReplacingWalk(v, n, n.Comment).(*ast.CommentGroup)
		}

	case *ast.BadDecl:
		// nothing to do

	case *ast.GenDecl:
		if n.Doc != nil {
			n.Doc = ReplacingWalk(v, n, n.Doc).(*ast.CommentGroup)
		}
		for i, s := range n.Specs {
			n.Specs[i] = ReplacingWalk(v, n, s).(ast.Spec)
		}

	case *ast.FuncDecl:
		if n.Doc != nil {
			n.Doc = ReplacingWalk(v, n, n.Doc).(*ast.CommentGroup)
		}
		if n.Recv != nil {
			n.Recv = ReplacingWalk(v, n, n.Recv).(*ast.FieldList)
		}
		n.Name = ReplacingWalk(v, n, n.Name).(*ast.Ident)
		n.Type = ReplacingWalk(v, n, n.Type).(*ast.FuncType)
		if n.Body != nil {
			n.Body = ReplacingWalk(v, n, n.Body).(*ast.BlockStmt)
		}

	// Files and packages
	case *ast.File:
		if n.Doc != nil {
			n.Doc = ReplacingWalk(v, n, n.Doc).(*ast.CommentGroup)
		}
		n.Name = ReplacingWalk(v, n, n.Name).(*ast.Ident)
		n.Decls = walkDeclList(v, n, n.Decls)
		// don't walk n.Comments - they have been
		// visited already through the individual
		// nodes

	case *ast.Package:
		for i, f := range n.Files {
			n.Files[i] = ReplacingWalk(v, n, f).(*ast.File)
		}

	default:
		panic(fmt.Sprintf("instrument.ReplacingWalk: unexpected node type %T", n))
	}

	return v.Replace(parent, node)
}
