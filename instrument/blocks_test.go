package instrument

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/timtadh/data-structures/test"
)

func TestSanity(x *testing.T) {
	t := (*test.T)(x)
	t.Assert(true, "The environment isn't sane")
}

func TestParse(x *testing.T) {
	t := (*test.T)(x)
	src := "x()"
	e, err := parser.ParseExpr(src)
	if err != nil {
		panic(err)
	}

	k := e.(*ast.CallExpr)

	t.Assert(k != nil, "The expression is not a CallExpr")
}

func TestParseFile(x *testing.T) {
	t := (*test.T)(x)

	src := `
package dummy
func main() {
}
`
	// Create the AST by parsing src.
	fset := token.NewFileSet() // positions are relative to fset
	f, err := parser.ParseFile(fset, "", src, 0)
	if err != nil {
		panic(err)
	}

	if funcD, ok := f.Decls[0].(*ast.FuncDecl); ok {
		t.Assert("main" == funcD.Name.Name, "expected 'main', got '%v'", funcD.Name.Name)
	}
}

type mockDo struct {
	Block []*[]ast.Stmt
	Id    []int
}

// Should do() the empty block and then stop
func TestEmptyFunc(x *testing.T) {
	t := (*test.T)(x)

	src := `
    package dummy
    func main(){
    }
    `
	// Create the AST by parsing src.
	fset := token.NewFileSet() // positions are relative to fset
	f, err := parser.ParseFile(fset, "", src, 0)
	if err != nil {
		panic(err)
	}

	if funcD, ok := f.Decls[0].(*ast.FuncDecl); ok {
		mDo := mockDo{make([]*[]ast.Stmt, 0), make([]int, 0)}

		blocks(&funcD.Body.List, nil, func(blk *[]ast.Stmt, id int) error {
			mDo.Block = append(mDo.Block, blk)
			mDo.Id = append(mDo.Id, id)
			return nil
		})

		if len(mDo.Block) > 1 {
			t.Fail()
		}
		//ast.Print(fset, f)
	}
}
