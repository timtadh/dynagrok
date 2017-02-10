package instrument

import (
	"sort"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
)

import (
	"github.com/timtadh/data-structures/errors"
	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/loader"
)

import (
	"github.com/timtadh/dynagrok/dgruntime/excludes"
	"github.com/timtadh/dynagrok/analysis"
)

type instrumenter struct {
	program *loader.Program
	entry string
}

func Instrument(entryPkgName string, program *loader.Program) (err error) {
	entry := program.Package(entryPkgName)
	if entry == nil {
		return errors.Errorf("The entry package was not found in the loaded program")
	}
	if entry.Pkg.Name() != "main" {
		return errors.Errorf("The entry package was not main")
	}
	i := &instrumenter{
		program: program,
		entry: entryPkgName,
	}
	return i.instrument()
}


func (i *instrumenter) instrument() (err error) {
	for _, pkg := range i.program.AllPackages {
		if len(pkg.BuildPackage.CgoFiles) > 0 {
			continue
		}
		if excludes.ExcludedPkg(pkg.Pkg.Path()) {
			continue
		}
		for _, fileAst := range pkg.Files {
			hadFunc := false
			err = analysis.Functions(pkg, fileAst, func(fn ast.Node, fnName string) error {
				hadFunc = true
				switch x := fn.(type) {
				case *ast.FuncDecl:
					if x.Body == nil {
						return nil
					}
					return i.fnBody(pkg, fnName, fn, &x.Body.List)
				case *ast.FuncLit:
					if x.Body == nil {
						return nil
					}
					return i.fnBody(pkg, fnName, fn, &x.Body.List)
				default:
					return errors.Errorf("unexpected type %T", x)
				}
			})
			if err != nil {
				return err
			}
			if hadFunc {
				astutil.AddImport(i.program.Fset, fileAst, "dgruntime")
			}
		}
	}
	return nil
}

func (i *instrumenter) fnBody(pkg *loader.PackageInfo, fnName string, fnAst ast.Node, fnBody *[]ast.Stmt) error {
	cfg := analysis.BuildCFG(i.program.Fset, fnName, fnAst, fnBody)
	fmt.Println(cfg)
	if true {
		// first collect the instrumentation points (IPs)
		// build a map from lexical blocks to a sequence of IPs
		// The IPs are basic blocks from the CFG
		instr := make(map[*[]ast.Stmt][]*analysis.Block)
		for _, b := range cfg.Blocks {
			// if the block doesn't have a body don't instrument it
			if b.Body == nil {
				continue
			}
			// associate the basic block with the lexical block
			instr[b.Body] = append(instr[b.Body], b)
		}
		// Now instrument each lexical block
		for body, blks := range instr {
			// First stort the IPs (Basic Blocks) in reverse order according
			// to where they start in the lexical block. That way we can
			// safely insert the instrumentation points (by doing it in
			// reverse
			sort.Slice(blks, func(i, j int) bool {
				return blks[i].StartsAt > blks[j].StartsAt
			})
			// instrument the entry to each block
			for _, b := range blks {
				// skip the entry block as it is covered by the EnterFunc call.
				if b.Id == 0 {
					continue
				}
				// get a position
				var pos token.Pos
				if len(b.Stmts) > 0 {
					pos = (*b.Stmts[0]).Pos()
				} else {
					// if there are no statements skip this block
					continue
				}
				switch stmt := (*body)[b.StartsAt].(type) {
				// If the insertion point for the instrumentation is a LabeledStmt
				// then we have two special cases
				case *ast.LabeledStmt:
					switch stmt.Stmt.(type) {
					case *ast.ForStmt, *ast.SwitchStmt, *ast.SelectStmt, *ast.TypeSwitchStmt, *ast.RangeStmt:
						// if it is one of the statements which allow labeled breaks/continues
						// then we can't insert instrumentation here. (But, don't worry we can insert
						// it inside of these statements so very little is lost).
					default:
						// Otherwise, in order to ensure our instrumentation is called first
						// (before any function calls) we need to replace the inner portion
						// of the LabeledStmt.
						*body = Insert(cfg, b, *body, b.StartsAt+1, stmt.Stmt)
						stmt.Stmt = i.mkEnterBlk(pos, b.Id)
						cfg.AddAllToBlk(b, stmt.Stmt)
					}
				default:
					// The general case, simply insert our instrumentation at the starting
					// points of the basic block in the lexical block.
					*body = Insert(cfg, b, *body, b.StartsAt, i.mkEnterBlk(pos, b.Id))
				}
			}
		}
		// Finally, we need to check for the existence of an os.Exit call and insert a
		// shutdown hook for Dyangrok if it exists.
		err := analysis.Blocks(fnBody, nil, func(blk *[]ast.Stmt, id int) error {
			for j := 0; j < len(*blk); j++ {
				pos := (*blk)[j].Pos()
				switch stmt := (*blk)[j].(type) {
				default:
					err := analysis.Exprs(stmt, func(expr ast.Expr) error {
						switch e := expr.(type) {
						case *ast.SelectorExpr:
							if ident, ok := e.X.(*ast.Ident); ok {
								if ident.Name == "os" && e.Sel.Name == "Exit" {
									*blk = Insert(cfg, nil, *blk, j, i.mkShutdownNow(pos))
									j++
								}
							}
						}
						return nil
					})
					if err != nil {
						return err
					}
				}
			}
			return nil
		})
		if err != nil {
			return nil
		}
	}
	*fnBody = Insert(cfg, cfg.Blocks[0], *fnBody, 0, i.mkEnterFunc(fnAst.Pos(), fnName))
	*fnBody = Insert(cfg, cfg.Blocks[0], *fnBody, 1, i.mkExitFunc(fnAst.Pos(), fnName))
	if pkg.Pkg.Path() == i.entry && fnName == fmt.Sprintf("%v.main", pkg.Pkg.Path()) {
		*fnBody = Insert(cfg, cfg.Blocks[0], *fnBody, 0, i.mkShutdown(fnAst.Pos()))
	}
	return nil
}

func Insert(cfg *analysis.CFG, cfgBlk *analysis.Block, blk []ast.Stmt, j int, stmt ast.Stmt) []ast.Stmt {
	if cfgBlk == nil {
		if len(blk) == 0{
			cfgBlk = nil
		} else if j >= len(blk) {
			j = len(blk)
			cfgBlk = cfg.GetClosestBlk(len(blk)-1, blk, blk[len(blk)-1])
		} else if j < 0 {
			j = 0
			cfgBlk = cfg.GetClosestBlk(0, blk, blk[0])
		} else if j == len(blk) {
			cfgBlk = cfg.GetClosestBlk(j-1, blk, blk[j-1])
		} else {
			cfgBlk = cfg.GetClosestBlk(j, blk, blk[j])
		}
		if cfgBlk == nil {
			p := cfg.FSet.Position(stmt.Pos())
			fmt.Printf("nil cfg-blk %T %v %v \n", stmt, analysis.FmtNode(cfg.FSet, stmt), p)
			// panic(fmt.Errorf("nil cfgBlk"))
		}
	}
	if cfgBlk != nil {
		cfg.AddAllToBlk(cfgBlk, stmt)
	}
	if cap(blk) <= len(blk) + 1 {
		nblk := make([]ast.Stmt, len(blk), (cap(blk)+1)*2)
		copy(nblk, blk)
		blk = nblk
	}
	blk = blk[:len(blk)+1]
	for i := len(blk)-1; i > 0; i-- {
		if j == i {
			blk[i] = stmt
			break
		}
		blk[i] = blk[i-1]
	}
	if j == 0 {
		blk[j] = stmt
	}
	return blk
}

func (i *instrumenter) mkPrint(pos token.Pos, data string) ast.Stmt {
	s := fmt.Sprintf("dgruntime.Println(%v)", strconv.Quote(data))
	e, err := parser.ParseExprFrom(i.program.Fset, i.program.Fset.File(pos).Name(), s, parser.Mode(0))
	if err != nil {
		panic(fmt.Errorf("mkPrint (%v) error: %v", s, err))
	}
	return &ast.ExprStmt{e}
}

func (i *instrumenter) mkDeferPrint(pos token.Pos, data string) ast.Stmt {
	s := fmt.Sprintf("func() { dgruntime.Println(%v) }()", strconv.Quote(data))
	e, err := parser.ParseExprFrom(i.program.Fset, i.program.Fset.File(pos).Name(), s, parser.Mode(0))
	if err != nil {
		panic(fmt.Errorf("mkDeferPrint (%v) error: %v", s, err))
	}
	return &ast.DeferStmt{Call: e.(*ast.CallExpr)}
}

func (i *instrumenter) mkEnterFunc(pos token.Pos, name string) ast.Stmt {
	p := i.program.Fset.Position(pos)
	s := fmt.Sprintf("dgruntime.EnterFunc(%v, %v)", strconv.Quote(name), strconv.Quote(p.String()))
	e, err := parser.ParseExprFrom(i.program.Fset, i.program.Fset.File(pos).Name(), s, parser.Mode(0))
	if err != nil {
		panic(fmt.Errorf("mkEnterFunc (%v) error: %v", s, err))
	}
	return &ast.ExprStmt{e}
}

func (i *instrumenter) mkExitFunc(pos token.Pos, name string) ast.Stmt {
	s := fmt.Sprintf("func() { dgruntime.ExitFunc(%v) }()", strconv.Quote(name))
	e, err := parser.ParseExprFrom(i.program.Fset, i.program.Fset.File(pos).Name(), s, parser.Mode(0))
	if err != nil {
		panic(fmt.Errorf("mkExitFunc (%v) error: %v", s, err))
	}
	return &ast.DeferStmt{Call: e.(*ast.CallExpr)}
}

func (i *instrumenter) mkShutdown(pos token.Pos) ast.Stmt {
	s := "func() { dgruntime.Shutdown() }()"
	e, err := parser.ParseExprFrom(i.program.Fset, i.program.Fset.File(pos).Name(), s, parser.Mode(0))
	if err != nil {
		panic(fmt.Errorf("mkShutdown (%v) error: %v", s, err))
	}
	return &ast.DeferStmt{Call: e.(*ast.CallExpr)}
}

func (i *instrumenter) mkShutdownNow(pos token.Pos) ast.Stmt {
	s := "dgruntime.Shutdown()"
	e, err := parser.ParseExprFrom(i.program.Fset, i.program.Fset.File(pos).Name(), s, parser.Mode(0))
	if err != nil {
		panic(fmt.Errorf("mkShutdown (%v) error: %v", s, err))
	}
	return &ast.ExprStmt{e}
}

func (i *instrumenter) mkEnterBlk(pos token.Pos, bbid int) ast.Stmt {
	p := i.program.Fset.Position(pos)
	s := fmt.Sprintf("dgruntime.EnterBlk(%d, %v)", bbid, strconv.Quote(p.String()))
	e, err := parser.ParseExprFrom(i.program.Fset, i.program.Fset.File(pos).Name(), s, parser.Mode(0))
	if err != nil {
		panic(fmt.Errorf("mkEnterBlk (%v) error: %v", s, err))
	}
	return &ast.ExprStmt{e}
}

