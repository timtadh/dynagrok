package grok

import (
	"fmt"
	"go/ast"
)

import (
	"github.com/timtadh/getopt"
	"github.com/timtadh/data-structures/errors"
)

import (
	"github.com/timtadh/dynagrok/cmd"
	"github.com/timtadh/dynagrok/analysis"
	"github.com/timtadh/dynagrok/dgruntime/excludes"
)

func NewCommand(c *cmd.Config) cmd.Runnable {
	return cmd.Cmd(
	"grok",
	`[options] <pkg>`,
	`
Option Flags
    -h,--help                         Show this message
`,
	"",
	[]string{},
	func(r cmd.Runnable, args []string, optargs []getopt.OptArg) ([]string, *cmd.Error) {
		if len(args) != 1 {
			return nil, cmd.Usage(r, 5, "Expected one package name got %v", args)
		}
		pkgName := args[0]
		fmt.Println("grokking", pkgName)
		program, err := cmd.LoadPkg(c, pkgName)
		if err != nil {
			return nil, cmd.Usage(r, 6, err.Error())
		}
		for _, pkg := range program.AllPackages {
			if excludes.ExcludedPkg(pkg.Pkg.Path()) {
				continue
			}
			for _, fileAst := range pkg.Files {
				err = analysis.Functions(pkg, fileAst, func(fn ast.Node, fnName string) error {
					var body *[]ast.Stmt
					switch x := fn.(type) {
					case *ast.FuncDecl:
						if x.Body == nil {
							return nil
						}
						body = &x.Body.List
					case *ast.FuncLit:
						if x.Body == nil {
							return nil
						}
						body = &x.Body.List
					default:
						return errors.Errorf("unexpected type %T", x)
					}
					cfg := analysis.BuildCFG(program.Fset, fnName, fn, body)
					fmt.Println(cfg)
					return nil
				})
				if err != nil {
					return nil, cmd.Errorf(9, "Error building cfg: %v", err)
				}
			}
		}
		return nil, nil
	})
}

