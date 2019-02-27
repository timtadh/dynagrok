// instrument walks a source program's AST to insert instrumentation
// statements.

package instrument

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	gotypes "go/types"
	"sort"
	"strconv"
	"strings"

	"github.com/timtadh/data-structures/errors"
	"github.com/timtadh/data-structures/set"
	"github.com/timtadh/data-structures/types"
	"github.com/timtadh/dynagrok/analysis"
	"github.com/timtadh/dynagrok/dgruntime/excludes"
	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/loader"
)

type InstrumentOption func(*instrumenter)

func InstrumentDataflow(i *instrumenter) {
	i.dataflow = true
}

type instrumenter struct {
	program     *loader.Program
	entry       string
	currentFile *ast.File
	dataflow    bool
}

func Instrument(entryPkgName string, program *loader.Program, options ...InstrumentOption) (err error) {
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
	}
	for _, opt := range options {
		opt(i)
	}
	return i.instrument()
}

func (i *instrumenter) instrument() (err error) {
	for _, pkg := range i.program.AllPackages {
		// if pkg.Cgo {
		// 	continue
		// }
		if excludes.ExcludedPkg(pkg.Pkg.Path()) {
			continue
		}
		for _, fileAst := range pkg.Files {
			i.currentFile = fileAst
			hadFunc := false
			err = analysis.Functions(pkg, fileAst, func(fn ast.Node, fnName string) error {
				hadFunc = true
				switch x := fn.(type) {
				case *ast.FuncDecl:
					if x.Body == nil {
						return nil
					}
					return i.fnBody(pkg, fileAst, fnName, fn, &x.Body.List)
				case *ast.FuncLit:
					if x.Body == nil {
						return nil
					}
					return i.fnBody(pkg, fileAst, fnName, fn, &x.Body.List)
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
func (i *instrumenter) fnBody(pkg *loader.PackageInfo, fileAst *ast.File, fnName string, fnAst ast.Node, fnBody *[]ast.Stmt) error {
	cfg := analysis.BuildCFG(i.program.Fset, fnName, fnAst, fnBody)
	var pdg string
	if true {
		var defs *analysis.Definitions
		var reachingDefs *analysis.ReachingDefinitions
		var locToStmt map[analysis.BlockLocation]*analysis.Statement
		if i.dataflow {
			defs = analysis.FindDefinitions(cfg, &pkg.Info)
			reachingDefs = defs.ReachingDefinitions()
			t := analysis.MakeProcedureDependenceGraph(cfg, analysis.ControlDependencies(cfg), reachingDefs)
			locToStmt = t.LocationToStmt
			pdg = t.JSON()
		}

		// first collect the instrumentation points (IPs)
		// build a map from lexical blocks to a sequence of IPs
		// The IPs are basic blocks from the CFG
		instr := make(map[*[]ast.Stmt][]*analysis.Block)
		for _, b := range cfg.Blocks {
			if b.Id == 0 {
				// skip the entry block as it is covered by the EnterFunc call.
			} else if len(b.Stmts) <= 0 {
				// if there are no statements skip this block
			} else if b.Body != nil {
				// we can insert a statement
				// associate the basic block with the lexical block
				instr[b.Body] = append(instr[b.Body], b)
			} else {
				// try expression level instrumentation
				err := i.exprInstrument(b)
				if err != nil {
					return err
				}
			}
			if !i.dataflow {
				continue
			}

			for _, stmt := range b.Stmts {
				switch (*stmt).(type) {
				case *ast.SwitchStmt:
					continue
				}
				*stmt = ReplaceExprs(*stmt, func(parent ast.Node, expr ast.Expr) ast.Expr {
					noInstrument := set.NewSortedSet(10)
					switch n := parent.(type) {
					case *ast.AssignStmt:
						for _, e := range n.Lhs {
							noInstrument.Add(types.Int(e.Pos()))
						}
					case *ast.ValueSpec:
						for _, e := range n.Names {
							noInstrument.Add(types.Int(e.Pos()))
						}
					case *ast.IncDecStmt:
						return expr
					}
					switch x := expr.(type) {
					case *ast.Ident:
						if x.Name == "_" {
							return expr
						}
						if noInstrument.Has(types.Int(expr.Pos())) {
							return expr
						}
						use := defs.References()[expr.Pos()]
						if use.HasObject() {
							typ := use.Declaration.Object.Type()
							switch typ.(type) {
							case *gotypes.Basic:
							default:
								return expr
							}
							typeName := i.localTypeName(use.Declaration.Object, pkg.Pkg, fileAst)
							// obj := use.Obj
							defs := reachingDefs.Reaches(use)
							if len(defs) <= 1 {
								return expr
							} else {
								// fmt.Println("blk", b.Id, "ident", use, "location", use.Location, use.Position)
								// fmt.Println("   ", "reaching defs", defs)
								loc := use.Location
								s := fmt.Sprintf(
									"func() %v { dgruntime.RecordValue(%q, %d, %d, %d, %q, %q, %v); return %v; }()",
									typeName, use.Position, loc.Block, loc.Stmt, locToStmt[*loc].Pos(), expr, use.Declaration.Position, expr, expr)
								instrumented, err := parser.ParseExprFrom(i.program.Fset, i.program.Fset.File(expr.Pos()).Name(), s, parser.Mode(0))
								if err != nil {
									fmt.Println(s)
									panic(err)
								}
								return instrumented
							}
						} else {
							return expr
						}
					default:
						return expr
					}
				})
			}
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
		if i.dataflow {
			err := analysis.Blocks(fnBody, nil, func(blk *[]ast.Stmt, id int) error {
				for idx := len(*blk) - 1; idx >= 0; idx-- {
					stmt := (*blk)[idx]
					switch s := stmt.(type) {
					case *ast.AssignStmt:
						for _, variable := range s.Lhs {
							if fmt.Sprint(variable) == "_" {
								continue
							}
							use := defs.References()[variable.Pos()]
							if !use.HasObject() {
								continue
							}
							typ := use.Declaration.Object.Type()
							loc := use.Location
							switch typ.(type) {
							case *gotypes.Basic:
								x := fmt.Sprintf(
									"dgruntime.RecordValue(%q, %d, %d, %d, %q, %q, %v)",
									use.Position, loc.Block, loc.Stmt, locToStmt[*loc].Pos(), variable, use.Declaration.Position, variable)
								instrumented, err := parser.ParseExprFrom(i.program.Fset, i.program.Fset.File(s.Pos()).Name(), x, parser.Mode(0))
								if err != nil {
									fmt.Println(x)
									panic(err)
								}
								*blk = Insert(cfg, nil, *blk, idx+1, &ast.ExprStmt{instrumented})
							}
						}
					}
				}
				return nil
			})
			if err != nil {
				return err
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
	pdt := cfg.PostDominators()
	cfgName := "__cfg"
	ipdomName := "__ipdom"
	pdgName := "__pdg"
	if len(cfg.Blocks) > 0 {
		*fnBody = Insert(cfg, cfg.Blocks[0], *fnBody, 0, i.mkCfg(fnAst.Pos(), cfg, cfgName))
		*fnBody = Insert(cfg, cfg.Blocks[0], *fnBody, 1, i.mkIdom(fnAst.Pos(), pdt, ipdomName))
		*fnBody = Insert(cfg, cfg.Blocks[0], *fnBody, 2, i.mkPdg(fnAst.Pos(), pdg, pdgName))
		*fnBody = Insert(cfg, cfg.Blocks[0], *fnBody, 3, i.mkEnterFunc(fnAst.Pos(), fnName, cfgName, ipdomName, pdgName))
		*fnBody = Insert(cfg, cfg.Blocks[0], *fnBody, 4, i.mkExitFunc(fnAst.Pos(), fnName))
		if pkg.Pkg.Path() == i.entry && fnName == fmt.Sprintf("%v.main", pkg.Pkg.Path()) {
			*fnBody = Insert(cfg, cfg.Blocks[0], *fnBody, 0, i.mkShutdown(fnAst.Pos()))
		}
	} else {
		emptyBlk := analysis.NewBlock(cfg.FSet, 0, fnBody, 0)
		cfg.Blocks = append(cfg.Blocks, emptyBlk)
		*fnBody = Insert(cfg, emptyBlk, *fnBody, 0, i.mkCfg(fnAst.Pos(), cfg, cfgName))
		*fnBody = Insert(cfg, emptyBlk, *fnBody, 1, i.mkIdom(fnAst.Pos(), pdt, ipdomName))
		*fnBody = Insert(cfg, cfg.Blocks[0], *fnBody, 2, i.mkPdg(fnAst.Pos(), "", pdgName))
		*fnBody = Insert(cfg, emptyBlk, *fnBody, 3, i.mkEnterFunc(fnAst.Pos(), fnName, cfgName, ipdomName, pdgName))
		*fnBody = Insert(cfg, emptyBlk, *fnBody, 4, i.mkExitFunc(fnAst.Pos(), fnName))
		if pkg.Pkg.Path() == i.entry && fnName == fmt.Sprintf("%v.main", pkg.Pkg.Path()) {
			*fnBody = Insert(cfg, emptyBlk, *fnBody, 0, i.mkShutdown(fnAst.Pos()))
		}
	}
	return nil
}

func (i *instrumenter) exprInstrument(b *analysis.Block) error {
	if len(b.Stmts) <= 0 {
		return nil
	}
	s := b.Stmts[0]
	// This is a list of all statement types.
	// More may be instrumentable in this fashion than are shown
	switch stmt := (*s).(type) {
	case *ast.BadStmt:
	case *ast.DeclStmt:
	case *ast.EmptyStmt:
	case *ast.ExprStmt:
	case *ast.SendStmt:
	case *ast.IncDecStmt:
	case *ast.AssignStmt:
	case *ast.GoStmt:
	case *ast.DeferStmt:
	case *ast.ReturnStmt:
	case *ast.LabeledStmt:
	case *ast.BranchStmt:
	case *ast.BlockStmt:
	case *ast.IfStmt:
		stmt.Cond = i.mkEnterBlkCond(stmt, stmt.Cond, b.Id)
	case *ast.ForStmt:
		stmt.Cond = i.mkEnterBlkCond(stmt, stmt.Cond, b.Id)
	case *ast.RangeStmt:
	case *ast.SelectStmt:
	case *ast.TypeSwitchStmt:
	case *ast.SwitchStmt:
	case *ast.CaseClause:
		panic(fmt.Errorf("Unexpected case clause %T %v", stmt, stmt))
	case *ast.CommClause:
		panic(fmt.Errorf("Unexpected comm clause %T %v", stmt, stmt))
	default:
		panic(fmt.Errorf("unexpected node %T", stmt))
	}
	return nil
}

func (i *instrumenter) localTypeName(object gotypes.Object, pkg *gotypes.Package, fileAst *ast.File) (name string) {
	addImport := func(other *gotypes.Package) string {
		imports := astutil.Imports(i.program.Fset, fileAst)
		for _, group := range imports {
			for _, importSpec := range group {
				path := strings.Trim(importSpec.Path.Value, "\"")
				if path != other.Path() {
					continue
				}
				if importSpec.Name != nil && importSpec.Name.Name == "." {
					return ""
				} else if importSpec.Name != nil {
					return importSpec.Name.Name
				} else {
					return other.Name()
				}
			}
		}
		astutil.AddImport(i.program.Fset, fileAst, other.Path())
		return other.Name()
	}
	name = gotypes.TypeString(object.Type(), func(other *gotypes.Package) string {
		if pkg == other {
			return "" // same package; unqualified
		}
		return addImport(other)
	})
	return name
}

func Insert(cfg *analysis.CFG, cfgBlk *analysis.Block, blk []ast.Stmt, j int, stmt ast.Stmt) []ast.Stmt {
	if cfg != nil {
		if cfgBlk == nil {
			if len(blk) == 0 {
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
				// p := cfg.FSet.Position(stmt.Pos())
				// fmt.Printf("nil cfg-blk %T %v %v \n", stmt, analysis.FmtNode(cfg.FSet, stmt), p)
				// panic(fmt.Errorf("nil cfgBlk"))
			}
		}
		if cfgBlk != nil {
			cfg.AddAllToBlk(cfgBlk, stmt)
		}
	} else {
		if j > len(blk) {
			j = len(blk)
		}
		if j < 0 {
			j = 0
		}
	}

	if cap(blk) <= len(blk)+1 {
		nblk := make([]ast.Stmt, len(blk), (cap(blk)+1)*2)
		copy(nblk, blk)
		blk = nblk
	}
	blk = blk[:len(blk)+1]
	for i := len(blk) - 1; i > 0; i-- {
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

func (i *instrumenter) mkEnterFunc(pos token.Pos, name, cfg, ipdom, pdg string) ast.Stmt {
	p := i.program.Fset.Position(pos)
	s := fmt.Sprintf("dgruntime.EnterFunc(%v, %v, %v, %v, %v)", strconv.Quote(name), strconv.Quote(p.String()), cfg, ipdom, pdg)
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

func (i *instrumenter) mkEnterBlkCond(stmt ast.Stmt, expr ast.Expr, bbid int) ast.Expr {
	var pos token.Pos
	if expr != nil {
		pos = expr.Pos()
	} else {
		pos = stmt.Pos()
	}
	p := i.program.Fset.Position(pos)
	enterStr := fmt.Sprintf("dgruntime.EnterBlkFromCond(%d, %v)", bbid, strconv.Quote(p.String()))
	enter, err := parser.ParseExprFrom(i.program.Fset, i.program.Fset.File(pos).Name(), enterStr, parser.Mode(0))
	if err != nil {
		panic(err)
	}
	if expr == nil {
		return enter
	} else {
		return &ast.BinaryExpr{
			X:     enter,
			Y:     expr,
			Op:    token.LAND,
			OpPos: pos,
		}
	}
}

func (i *instrumenter) mkCfg(pos token.Pos, cfg *analysis.CFG, varName string) ast.Stmt {
	nexts := cfg.Nexts()
	parts := make([]string, 0, len(nexts))
	for _, next := range nexts {
		bits := make([]string, 0, len(next))
		for _, x := range next {
			bits = append(bits, fmt.Sprintf("%d", x))
		}
		parts = append(parts, fmt.Sprintf("[]int{%s}", strings.Join(bits, ", ")))
	}
	s := fmt.Sprintf("[][]int{%s}", strings.Join(parts, ", "))
	arr, err := parser.ParseExprFrom(i.program.Fset, i.program.Fset.File(pos).Name(), s, parser.Mode(0))
	if err != nil {
		panic(fmt.Errorf("mkCfg (%v) error: %v", s, err))
	}
	variable := ast.NewIdent(varName)
	return &ast.AssignStmt{
		Lhs: []ast.Expr{variable},
		Tok: token.DEFINE,
		Rhs: []ast.Expr{arr},
	}
}

func (i *instrumenter) mkIdom(pos token.Pos, dt *analysis.DominatorTree, varName string) ast.Stmt {
	idom := dt.ImmediateDominators()
	parts := make([]string, 0, len(idom))
	for _, y := range idom {
		parts = append(parts, fmt.Sprintf("%d", y))
	}
	s := fmt.Sprintf("[]int{%s}", strings.Join(parts, ", "))
	arr, err := parser.ParseExprFrom(i.program.Fset, i.program.Fset.File(pos).Name(), s, parser.Mode(0))
	if err != nil {
		panic(fmt.Errorf("mkIdom (%v) error: %v", s, err))
	}
	variable := ast.NewIdent(varName)
	return &ast.AssignStmt{
		Lhs: []ast.Expr{variable},
		Tok: token.DEFINE,
		Rhs: []ast.Expr{arr},
	}
}

func (i *instrumenter) mkPdg(pos token.Pos, pdg string, varName string) ast.Stmt {
	s := strconv.Quote(pdg)
	out, err := parser.ParseExprFrom(i.program.Fset, i.program.Fset.File(pos).Name(), s, parser.Mode(0))
	if err != nil {
		panic(fmt.Errorf("mkPdg (%v) error: %v", s, err))
	}
	variable := ast.NewIdent(varName)
	return &ast.AssignStmt{
		Lhs: []ast.Expr{variable},
		Tok: token.DEFINE,
		Rhs: []ast.Expr{out},
	}
}
