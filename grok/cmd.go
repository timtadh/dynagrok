package grok

import (
	"fmt"
	"go/ast"
	"os"

	"github.com/timtadh/data-structures/errors"
	"github.com/timtadh/dynagrok/analysis"
	"github.com/timtadh/dynagrok/cmd"
	"github.com/timtadh/dynagrok/dgruntime/excludes"
	"github.com/timtadh/getopt"
)

func NewCommand(c *cmd.Config) cmd.Runnable {
	return cmd.Cmd(
		"grok",
		`[options] <pkg>`,
		`
Print CFGs for the functions in the program.

Option Flags
    -h,--help                         Show this message
    -f,--fn=<name>                    Only show the CFG for func <name>
`,
		"f:",
		[]string{
			"fn=",
		},
		func(r cmd.Runnable, args []string, optargs []getopt.OptArg) ([]string, *cmd.Error) {
			onlyFn := ""
			for _, oa := range optargs {
				switch oa.Opt() {
				case "-f", "--fn":
					onlyFn = oa.Arg()
				}
			}
			if len(args) != 1 {
				return nil, cmd.Usage(r, 5, "Expected one package name got %v", args)
			}
			pkgName := args[0]
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
						if onlyFn != "" && onlyFn != fnName {
							return nil
						}
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
						rd := analysis.FindDefinitions(cfg, &pkg.Info).ReachingDefinitions()
						{
							loc := &analysis.BlockLocation{-1, -1}
							gen, kill := rd.GenKill(loc)
							fmt.Fprintln(os.Stderr, loc, gen, kill)
						}
						for _, blk := range cfg.Blocks {
							for sid, stmt := range blk.Stmts {
								loc := &analysis.BlockLocation{blk.Id, sid}
								gen, kill := rd.GenKill(loc)
								fmt.Fprintln(os.Stderr, loc, analysis.FmtNode(cfg.FSet, *stmt), gen, kill)
							}
						}
						fmt.Fprintln(os.Stderr)
						fmt.Fprintln(os.Stderr)
						{
							loc := &analysis.BlockLocation{-1, -1}
							fmt.Fprintln(os.Stderr, loc, rd.In(loc), rd.Out(loc))
						}
						for _, blk := range cfg.Blocks {
							for sid, _ := range blk.Stmts {
								loc := &analysis.BlockLocation{blk.Id, sid}
								fmt.Fprintln(os.Stderr, loc, rd.In(loc), rd.Out(loc))
							}
						}
						fmt.Fprintln(os.Stderr)
						fmt.Fprintln(os.Stderr, cfg.Nexts())
						// fmt.Fprintln(os.Stderr, cfg.Dominators())
						// fmt.Fprintln(os.Stderr, cfg.Dominators().Frontier())
						// fmt.Fprintln(os.Stderr)
						fmt.Fprintln(os.Stderr, cfg.PostDominators())
						// fmt.Fprintln(os.Stderr, cfg.PostDominators().Frontier())
						// fmt.Fprintln(os.Stderr)
						// fmt.Fprintln(os.Stderr)
						fmt.Fprintln(os.Stderr, cfg.PostDominators().ImmediateDominators())
						fmt.Println(cfg.Dotty())
						for i, blk := range cfg.Blocks {
							if blk == nil {
								fmt.Fprintln(os.Stderr, "nil blk", i, cfg.Name)
							}
						}
						fmt.Println(cfg.Dominators().Dotty(cfg))
						fmt.Println(cfg.PostDominators().Dotty(cfg))
						fmt.Println(analysis.ControlDependencies(cfg).Dotty(cfg))
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
