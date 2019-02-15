package analysis

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
	"strings"

	ds_types "github.com/timtadh/data-structures/types"
)

type StatementFlowGraph struct {
	cfg            *CFG
	Statements     []*Statement
	Flows          [][]int
	lookup         map[token.Pos]*Statement
	LocationToStmt map[BlockLocation]*Statement
}

type ProcedureDependenceGraph struct {
	StatementFlowGraph
	Controls     [][]int // Controls[i][j] says statement i controls the execution of j
	ProvidesData [][]int // ProvidesData[i][j] says statement i provides data to statement j
}

type Statement struct {
	Id            int // Id is local to the statements containing function
	Stmt          *ast.Stmt
	Fn            ast.Node
	Blk           *Block // Containing CFG Basic Block
	BlockLocation *BlockLocation
	Uses          []*Use
	Assigns       []*Use
	Declares      []*Declaration
}

func (s *Statement) Pos() token.Pos {
	if s.Stmt != nil {
		return (*s.Stmt).Pos()
	}
	return s.Fn.Pos()
}

func MakeProcedureDependenceGraph(cfg *CFG, cdg *ControlDependenceGraph, defs *ReachingDefinitions) *ProcedureDependenceGraph {
	g := &ProcedureDependenceGraph{
		StatementFlowGraph: *MakeStatementFlowGraph(cfg),
	}
	g.Controls = g.controlDependencies(cfg, cdg)
	g.ProvidesData = g.dataDependencies(cfg, defs)
	return g
}

func (g *StatementFlowGraph) Statement(loc BlockLocation) *Statement {
	return g.LocationToStmt[loc]
}

func (g *StatementFlowGraph) controlDependencies(cfg *CFG, cdg *ControlDependenceGraph) [][]int {
	if len(g.Statements) <= 0 {
		return nil
	}
	if len(cfg.Blocks) <= 0 {
		return nil
	}
	edges := make([][]int, len(g.Statements))
	entry := g.Statements[0]
	for _, s := range cfg.Blocks[0].Stmts {
		stmt := g.lookup[(*s).Pos()]
		edges[entry.Id] = append(edges[entry.Id], stmt.Id)
	}
	for _, blk := range cfg.Blocks {
		if len(blk.Stmts) <= 0 {
			continue
		}
		last := g.lookup[(*blk.Stmts[len(blk.Stmts)-1]).Pos()]
		for _, nblk := range cdg.Next(blk) {
			for _, s := range nblk.Stmts {
				stmt := g.lookup[(*s).Pos()]
				edges[last.Id] = append(edges[last.Id], stmt.Id)
			}
		}
	}
	return edges
}
func (g *StatementFlowGraph) dataDependencies(cfg *CFG, defs *ReachingDefinitions) [][]int {
	if len(g.Statements) <= 0 {
		return nil
	}
	if len(cfg.Blocks) <= 0 {
		return nil
	}
	type edge struct {
		source int
		target int
	}
	added := make(map[edge]bool)
	edges := make([][]int, len(g.Statements))
	for _, def := range defs.objs {
		loc := def.Location
		var stmt *Statement
		if loc.Block < 0 || loc.Stmt < 0 {
			// a parameter
			stmt = g.Statements[0]
		} else {
			s := cfg.Blocks[loc.Block].Stmts[loc.Stmt]
			stmt = g.lookup[(*s).Pos()]
		}
		stmt.Declares = append(stmt.Declares, def)
	}
	assigned := make(map[int]map[token.Pos]bool)
	for _, stmt := range g.Statements {
		assigned[stmt.Id] = make(map[token.Pos]bool)
	}
	for _, use := range defs.uses {
		if !use.HasObject() {
			continue
		}
		loc := use.Location
		var stmt *Statement
		if loc.Block < 0 || loc.Stmt < 0 {
			// a parameter
			stmt = g.Statements[0]
		} else {
			s := cfg.Blocks[loc.Block].Stmts[loc.Stmt]
			stmt = g.lookup[(*s).Pos()]
		}
		gen, _ := defs.GenKill(loc)
		if gen.Has(ds_types.Int(use.Id)) {
			assigned[stmt.Id][use.Declaration.Id] = true
			stmt.Assigns = append(stmt.Assigns, use)
		}
	}
	for _, use := range defs.uses {
		if !use.HasObject() {
			continue
		}
		loc := use.Location
		var stmt *Statement
		if loc.Block < 0 || loc.Stmt < 0 {
			// a parameter
			stmt = g.Statements[0]
		} else {
			s := cfg.Blocks[loc.Block].Stmts[loc.Stmt]
			stmt = g.lookup[(*s).Pos()]
		}
		if !assigned[stmt.Id][use.Declaration.Id] {
			stmt.Uses = append(stmt.Uses, use)
		}
	}
	for bid, blk := range cfg.Blocks {
		for sid, t := range blk.Stmts {
			target := g.lookup[(*t).Pos()]
			loc := &BlockLocation{Block: bid, Stmt: sid}
			blkExprs(*t, func(e ast.Expr) {
				switch i := e.(type) {
				case *ast.Ident:
					use := defs.References()[i.Pos()]
					if !use.HasObject() {
						return
					}
					for _, use := range defs.InFor(loc, use.Declaration) {
						if use.Location.Block < 0 || use.Location.Stmt < 0 {
							// these come from the params
							source := g.Statements[0]
							e := edge{source.Id, target.Id}
							if added[e] {
								continue
							}
							added[e] = true
							edges[source.Id] = append(edges[source.Id], target.Id)
							continue
						}
						s := cfg.Blocks[use.Location.Block].Stmts[use.Location.Stmt]
						source := g.lookup[(*s).Pos()]
						e := edge{source.Id, target.Id}
						if added[e] {
							continue
						}
						added[e] = true
						edges[source.Id] = append(edges[source.Id], target.Id)
					}
				}
			})
		}
	}
	return edges
}

func MakeStatementFlowGraph(cfg *CFG) *StatementFlowGraph {
	stmts := make([]*Statement, 0, len(cfg.Blocks))
	lookup := make(map[token.Pos]*Statement)
	if len(cfg.Blocks) <= 0 {
		return &StatementFlowGraph{}
	}
	locToStmt := make(map[BlockLocation]*Statement)
	entry := &Statement{
		Id:            len(stmts),
		Fn:            cfg.Fn,
		Blk:           cfg.Blocks[0],
		BlockLocation: &BlockLocation{-1, -1},
	}
	stmts = append(stmts, entry)
	lookup[cfg.Fn.Pos()] = entry
	locToStmt[BlockLocation{-1, -1}] = entry
	for bid, blk := range cfg.Blocks {
		for sid, stmt := range blk.Stmts {
			s := &Statement{
				Id:            len(stmts),
				Stmt:          stmt,
				Blk:           blk,
				BlockLocation: &BlockLocation{bid, sid},
			}
			stmts = append(stmts, s)
			lookup[(*stmt).Pos()] = s
			locToStmt[BlockLocation{bid, sid}] = s
		}
	}
	edges := make([][]int, len(stmts))
	if len(cfg.Blocks[0].Stmts) > 0 {
		first := lookup[(*cfg.Blocks[0].Stmts[0]).Pos()]
		edges[entry.Id] = append(edges[entry.Id], first.Id)
	}
	for _, blk := range cfg.Blocks {
		if len(blk.Stmts) <= 0 {
			continue
		}
		for i := 0; i < len(blk.Stmts)-1; i++ {
			cur := lookup[(*blk.Stmts[i]).Pos()]
			next := lookup[(*blk.Stmts[i+1]).Pos()]
			edges[cur.Id] = append(edges[cur.Id], next.Id)
		}
		last := lookup[(*blk.Stmts[len(blk.Stmts)-1]).Pos()]
		for _, flow := range blk.Next {
			nextBlk := flow.Block
			if len(nextBlk.Stmts) > 0 {
				first := lookup[(*nextBlk.Stmts[0]).Pos()]
				edges[last.Id] = append(edges[last.Id], first.Id)
			}
		}
	}
	return &StatementFlowGraph{
		cfg:            cfg,
		Statements:     stmts,
		Flows:          edges,
		lookup:         lookup,
		LocationToStmt: locToStmt,
	}
}

func (g *StatementFlowGraph) Dotty() string {
	return g.dotty("statement-flow-graph", []edgeSet{{g.Flows, ""}})
}

func (g *ProcedureDependenceGraph) CDGDotty() string {
	return g.dotty("statement-control-dependence-graph", []edgeSet{{g.Controls, ""}})
}

func (g *ProcedureDependenceGraph) DDGDotty() string {
	return g.dotty("statement-data-dependence-graph", []edgeSet{{g.ProvidesData, ""}})
}

func (g *ProcedureDependenceGraph) Dotty() string {
	return g.dotty("statement-procedure-dependence-graph", []edgeSet{
		{g.Controls, "controls"},
		{g.ProvidesData, "provides-data"}})
}

func (g *ProcedureDependenceGraph) JSON() string {
	type jdecl struct {
		Id   int
		Name string
	}
	type juse struct {
		Id     int
		Name   string
		DeclId int
	}
	type jstatement struct {
		Id            int
		Text          string
		Position      string
		BlockLocation *BlockLocation
		Declares      []jdecl
		Assigns       []jdecl
		Uses          []juse
	}
	type jgraph struct {
		Statements []jstatement
		Flows      [][]int
		Controls   [][]int
		DataFlows  [][]int
	}
	toJdecl := func(def *Declaration) jdecl {
		return jdecl{
			Id:   int(def.Id),
			Name: def.Ident.Name,
		}
	}
	toJuse := func(use *Use) juse {
		return juse{
			Id:     use.Id,
			DeclId: int(use.Declaration.Id),
			Name:   use.Declaration.Ident.Name,
		}
	}
	toJstmt := func(stmt *Statement) jstatement {
		decls := make([]jdecl, 0, len(stmt.Declares))
		assigns := make([]jdecl, 0, len(stmt.Assigns))
		uses := make([]juse, 0, len(stmt.Uses))
		for _, d := range stmt.Declares {
			decls = append(decls, toJdecl(d))
		}
		for _, a := range stmt.Assigns {
			assigns = append(assigns, toJdecl(a.Declaration))
		}
		for _, u := range stmt.Uses {
			uses = append(uses, toJuse(u))
		}
		if stmt.Stmt != nil {
			return jstatement{
				Id:            int((*stmt.Stmt).Pos()),
				Text:          FmtStmt(g.cfg.FSet, stmt.Stmt),
				Position:      fmt.Sprint(g.cfg.FSet.Position((*stmt.Stmt).Pos())),
				BlockLocation: stmt.BlockLocation,
				Declares:      decls,
				Assigns:       assigns,
				Uses:          uses,
			}
		} else {
			var label string
			switch fn := stmt.Fn.(type) {
			case *ast.FuncDecl:
				label = strconv.Quote(FmtNode(stmt.Blk.FSet, fn.Type))
			case *ast.FuncLit:
				label = strconv.Quote(FmtNode(stmt.Blk.FSet, fn.Type))
			default:
				label = strconv.Quote("unknown-func-type")
			}
			return jstatement{
				Id:            int(stmt.Fn.Pos()),
				Text:          label,
				Position:      fmt.Sprint(g.cfg.FSet.Position(stmt.Fn.Pos())),
				BlockLocation: stmt.BlockLocation,
				Declares:      decls,
				Assigns:       assigns,
				Uses:          uses,
			}
		}
	}
	copyEdges := func(edges [][]int) [][]int {
		arcs := make([][]int, 0, len(edges))
		for _, adj := range edges {
			cur := make([]int, len(adj))
			copy(cur, adj)
			arcs = append(arcs, cur)
		}
		return arcs
	}
	toJgraph := func(g *ProcedureDependenceGraph) jgraph {
		stmts := make([]jstatement, 0, len(g.Statements))
		for _, stmt := range g.Statements {
			stmts = append(stmts, toJstmt(stmt))
		}
		return jgraph{
			Statements: stmts,
			Flows:      copyEdges(g.Flows),
			Controls:   copyEdges(g.Controls),
			DataFlows:  copyEdges(g.ProvidesData),
		}
	}
	var buf bytes.Buffer
	e := json.NewEncoder(&buf)
	err := e.Encode(toJgraph(g))
	if err != nil {
		panic(err)
	}
	return buf.String()
}

type edgeSet struct {
	edges [][]int
	label string
}

func (g *StatementFlowGraph) dotty(graphName string, edgeSets []edgeSet) string {
	name := fmt.Sprintf("%s:%s", g.cfg.Name, graphName)
	nodes := make([]string, 0, len(g.Statements))
	arcs := make([]string, 0, len(g.Statements))
	for _, stmt := range g.Statements {
		var label string
		if stmt.Stmt != nil {
			label = strconv.Quote(FmtStmt(stmt.Blk.FSet, stmt.Stmt))
		} else {
			switch fn := stmt.Fn.(type) {
			case *ast.FuncDecl:
				label = strconv.Quote(FmtNode(stmt.Blk.FSet, fn.Type))
			case *ast.FuncLit:
				label = strconv.Quote(FmtNode(stmt.Blk.FSet, fn.Type))
			default:
				label = strconv.Quote("unknown-func-type")
			}
		}
		label = strings.Replace(label, "\\n", "\\l", -1)
		nodes = append(nodes, fmt.Sprintf("n%d [label=%v];", stmt.Id, label))
	}
	for _, es := range edgeSets {
		for sid, kids := range es.edges {
			for _, tid := range kids {
				arcs = append(arcs, fmt.Sprintf("n%d -> n%d [label=%q];", sid, tid, es.label))
			}
		}
	}
	return fmt.Sprintf(`digraph %q {
label=%q;
labelloc=top;
node [shape="rect", labeljust=l];
%v
%v
}`, name, name, strings.Join(nodes, "\n"), strings.Join(arcs, "\n"))
}
