package objectstate

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
)

import (
	"github.com/timtadh/data-structures/errors"
	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/loader"
)

import (
	"github.com/timtadh/dynagrok/analysis"
	"github.com/timtadh/dynagrok/dgruntime/excludes"
	"github.com/timtadh/dynagrok/instrument"
)

type instrumenter struct {
	program *loader.Program
	entry   string
	method  string
	// TODO check if currentFile is what we want - iirc this is used to find
	// import statements
	currentFile *ast.File
}

func Instrument(entryPkgName string, methodName string, program *loader.Program) (err error) {
	entry := program.Package(entryPkgName)
	if entry == nil {
		return errors.Errorf("The entry package was not found in the loaded program")
	}
	if entry.Pkg.Name() != "main" {
		return errors.Errorf("The entry package was not main")
	}
	i := &instrumenter{
		program: program,
		entry:   entryPkgName,
		method:  methodName,
	}
	return i.instrument()
}

func (i *instrumenter) instrument() (err error) {
	for _, pkg := range i.program.AllPackages {
		//if len(pkg.BuildPackage.CgoFiles) > 0 {
		//	continue
		//}
		if excludes.ExcludedPkg(pkg.Pkg.Path()) {
			continue
		}
		for _, fileAst := range pkg.Files {
			i.currentFile = fileAst
			hadFunc := false
			err = analysis.Functions(pkg, fileAst, func(fn ast.Node, fnName string) error {
				if i.method != "" && !strings.Contains(fnName, i.method) {
					return nil
				}
				hadFunc = true
				switch x := fn.(type) {
				case *ast.FuncDecl:
					if x.Body == nil {
						return nil
					}
					receivers := []*ast.Field{}
					params := []*ast.Field{}
					results := []*ast.Field{}
					if x.Recv != nil {
						receivers = x.Recv.List
					}
					if x.Type.Params != nil {
						params = x.Type.Params.List
					}
					if x.Type.Results != nil {
						results = x.Type.Results.List
					}
					return i.function(fnName, fn, &receivers, &params, &results, &x.Body.List)
				case *ast.FuncLit:
					if x.Body == nil {
						return nil
					}
					params := []*ast.Field{}
					results := []*ast.Field{}
					if x.Type.Params != nil {
						params = x.Type.Params.List
					}
					if x.Type.Results != nil {
						results = x.Type.Results.List
					}
					return i.function(fnName, fn, new([]*ast.Field), &params, &results, &x.Body.List)
				default:
					return errors.Errorf("unexpected type %T", x)
				}
			})
			if err != nil {
				return err
			}
			// imports dgruntime package into the files that have
			// instrumentation added
			if hadFunc {
				astutil.AddImport(i.program.Fset, fileAst, "dgruntime")
			}
		}
	}
	return nil
}

func (i *instrumenter) function(fnName string, fnAst ast.Node, recv *[]*ast.Field, params *[]*ast.Field, results *[]*ast.Field, body *[]ast.Stmt) error {
	inputs := []string{}
	outputs := []string{}
	for _, r := range *recv {
		for _, name := range r.Names {
			inputs = append(inputs, name.Name)
		}
	}
	for _, input := range *params {
		for _, name := range input.Names {
			inputs = append(inputs, name.Name)
		}
	}

	// TODO You idiot, returns don't have to come at the end
	// if the function has a return statement
	if ret, ok := (*body)[len(*body)-1].(*ast.ReturnStmt); ok {
		stmt, vars, varnames := i.mkAssignment(ret.Pos(), ret.Results)
		*body = instrument.Insert(nil, nil, *body, len(*body)-1, stmt)
		ret.Results = vars
		outputs = append(outputs, varnames...)
		*body = instrument.Insert(nil, nil, *body, len(*body)-1, i.mkMethodOutput(fnAst.Pos(), fnName, outputs))
	} else {
		// Otherwise check for named outputs
		for _, output := range *results {
			for _, name := range output.Names {
				outputs = append(outputs, name.Name)
				*body = instrument.Insert(nil, nil, *body, 1, i.mkDeferMethodOutput(fnAst.Pos(), fnName, outputs))
			}
		}
	}

	if len(inputs) != 0 {
		*body = instrument.Insert(nil, nil, *body, 0, i.mkMethodInput(fnAst.Pos(), fnName, inputs))
	}

	return nil
}

func (i instrumenter) mkMethodInput(pos token.Pos, name string, inputs []string) ast.Stmt {
	p := i.program.Fset.Position(pos)
	s := fmt.Sprintf("dgruntime.MethodInput(%s, %s", strconv.Quote(name), strconv.Quote(p.String()))
	for _, input := range inputs {
		s = fmt.Sprintf("%v, struct {Name string \nVal interface{}\n}{Name: \"%v\", Val: %v}", s, input, input)
	}

	s = s + ")"
	e, err := parser.ParseExprFrom(i.program.Fset, i.program.Fset.File(pos).Name(), s, parser.Mode(0))
	if err != nil {
		panic(fmt.Errorf("mkMethodInput (%v) error: %v", s, err))
	}
	return &ast.ExprStmt{e}
}

func (i instrumenter) mkDeferMethodOutput(pos token.Pos, name string, inputs []string) ast.Stmt {
	p := i.program.Fset.Position(pos)
	s := fmt.Sprintf("func() { dgruntime.MethodOutput(%s, %s", strconv.Quote(name), strconv.Quote(p.String()))
	for _, input := range inputs {
		s = fmt.Sprintf("%v, struct {Name string \nVal interface{}\n}{Name: \"%v\", Val: %v}", s, input, input)
	}

	s = s + ") }()"
	e, err := parser.ParseExprFrom(i.program.Fset, i.program.Fset.File(pos).Name(), s, parser.Mode(0))
	if err != nil {
		panic(fmt.Errorf("mkMethodOutput (%v) error: %v", s, err))
	}
	return &ast.DeferStmt{Call: e.(*ast.CallExpr)}
}

func (i instrumenter) mkMethodOutput(pos token.Pos, name string, inputs []string) ast.Stmt {
	p := i.program.Fset.Position(pos)
	s := fmt.Sprintf("dgruntime.MethodOutput(%s, %s", strconv.Quote(name), strconv.Quote(p.String()))
	for _, input := range inputs {
		s = fmt.Sprintf("%v, struct {Name string \nVal interface{}\n}{Name: \"%v\", Val: %v}", s, input, input)
	}
	s = s + ")"
	e, err := parser.ParseExprFrom(i.program.Fset, i.program.Fset.File(pos).Name(), s, parser.Mode(0))
	if err != nil {
		panic(fmt.Errorf("mkMethodOutput (%v) error: %v", s, err))
	}
	return &ast.ExprStmt{e}
}

func (i instrumenter) mkAssignment(pos token.Pos, exprs []ast.Expr) (ast.Stmt, []ast.Expr, []string) {
	s := ""
	varnames := make([]string, len(exprs))
	for i := range exprs {
		if i != 0 {
			s += ", "
		}
		varnames[i] = fmt.Sprintf("dynagrokV%d", i)
		s += varnames[i]
	}
	s += ":="
	for i := range exprs {
		if i != 0 {
			s += ", "
		}
		s += fmt.Sprintf("%v", i)
	}
	s = fmt.Sprintf("func() { %s }()", s)
	e, err := parser.ParseExprFrom(i.program.Fset, i.program.Fset.File(pos).Name(), s, parser.Mode(0))
	if err != nil {
		panic(fmt.Errorf("AssignStmt: %v error: %v", s, err))
	}

	var stmt ast.AssignStmt
	if call, ok := e.(*ast.CallExpr); ok {
		if fun, ok := call.Fun.(*ast.FuncLit); ok {
			if assign, ok := fun.Body.List[0].(*ast.AssignStmt); ok {
				stmt = *assign
			}
		}
	}
	for i, e := range exprs {
		if id, ok := e.(*ast.Ident); ok {
			id.NamePos = stmt.Rhs[i].Pos()
		}
	}
	stmt.Rhs = exprs

	return &stmt, stmt.Lhs, varnames
}
