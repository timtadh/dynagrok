package analysis

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
)

type BlockLocation struct {
	Block int // Basic block id
	Stmt  int // Statement Id in the Block
}

type Object struct {
	Id       token.Pos
	Ident    *ast.Ident
	Object   types.Object
	Location *BlockLocation
	Redefs   map[token.Pos]bool
}

type Reference struct {
	Id  token.Pos
	Oid token.Pos
}

type ReachingDefinitions struct {
	cfg  *CFG
	objs map[token.Pos]*Object
	refs map[token.Pos]*Reference
	info *types.Info
	in   map[BlockLocation]map[token.Pos]bool // reaching inputs (as def Id)
	out  map[BlockLocation]map[token.Pos]bool // reaching outputs (as def Id)
}

func NewReachingDefinitions(cfg *CFG, info *types.Info) *ReachingDefinitions {
	rd := &ReachingDefinitions{
		cfg:  cfg,
		objs: make(map[token.Pos]*Object),
		refs: make(map[token.Pos]*Reference),
		info: info,
	}
	for _, blk := range cfg.Blocks {
		for sid, stmt := range blk.Stmts {
			blkExprs(*stmt, func(expr ast.Expr) {
				switch e := expr.(type) {
				case *ast.Ident:
					if obj := info.Defs[e]; obj != nil {
						// this is a definition
						fmt.Fprintln(os.Stderr, "def", FmtNode(cfg.FSet, e), ":", obj.Id(), obj.Pos())
						object := &Object{
							Id:     obj.Pos(),
							Ident:  e,
							Object: obj,
							Location: &BlockLocation{
								Block: blk.Id,
								Stmt:  sid,
							},
							Redefs: make(map[token.Pos]bool),
						}
						ref := &Reference{
							Id:  e.Pos(),
							Oid: obj.Pos(),
						}
						rd.objs[object.Id] = object
						rd.refs[ref.Id] = ref
					} else if obj := info.Uses[e]; obj != nil {
						fmt.Fprintln(os.Stderr, "use", FmtNode(cfg.FSet, e), ":", obj.Id(), obj.Pos())
						ref := &Reference{
							Id:  e.Pos(),
							Oid: obj.Pos(),
						}
						rd.refs[ref.Id] = ref
					}
				}
			})
		}
		add := func(e *ast.Ident) {
			ref := rd.refs[e.Pos()]
			obj := rd.objs[ref.Oid]
			obj.Redefs[ref.Id] = true
		}
		for _, stmt := range blk.Stmts {
			switch s := (*stmt).(type) {
			case *ast.IncDecStmt:
				switch e := s.X.(type) {
				case *ast.Ident:
					add(e)
				}
			case *ast.AssignStmt:
				for _, expr := range s.Lhs {
					switch e := expr.(type) {
					case *ast.Ident:
						add(e)
					}
				}
			}
		}
	}
	return rd
}

func (rd *ReachingDefinitions) In(loc *BlockLocation) map[token.Pos]bool {
	return rd.in[*loc]
}

func (rd *ReachingDefinitions) Out(loc *BlockLocation) map[token.Pos]bool {
	return rd.out[*loc]
}

func (rd *ReachingDefinitions) Solve() {
	different := func(a, b map[token.Pos]bool) bool {
		if len(a) != len(b) {
			return true
		}
		for x := range a {
			if !b[x] {
				return true
			}
		}
		for x := range b {
			if !a[x] {
				return true
			}
		}
		return false
	}
	in := make(map[BlockLocation]map[token.Pos]bool)
	out := make(map[BlockLocation]map[token.Pos]bool)
	stack := make([]BlockLocation, 0, 10)
	for _, blk := range rd.cfg.Blocks {
		for sid, _ := range blk.Stmts {
			loc := BlockLocation{blk.Id, sid}
			in[loc] = make(map[token.Pos]bool)
			out[loc] = make(map[token.Pos]bool)
			stack = append(stack, loc)
		}
	}
	for len(stack) > 0 {
		var cur BlockLocation
		stack, cur = stack[:len(stack)-1], stack[len(stack)-1]
		res := rd.Flow(&cur, in[cur])
		if different(res, out[cur]) {
			out[cur] = res
			blk := rd.cfg.Blocks[cur.Block]
			if cur.Stmt+1 < len(blk.Stmts) {
				next := BlockLocation{blk.Id, cur.Stmt + 1}
				in[next] = out[cur]
				stack = append(stack, next)
			} else {
				for _, n := range blk.Next {
					next := BlockLocation{n.Block.Id, 0}
					in[next] = out[cur]
					stack = append(stack, next)
				}
			}
		}
	}
	rd.in = in
	rd.out = out
}

func (rd *ReachingDefinitions) Flow(loc *BlockLocation, in map[token.Pos]bool) (out map[token.Pos]bool) {
	out = make(map[token.Pos]bool)
	gen, kill := rd.GenKill(loc)
	for _, x := range gen {
		out[x] = true
	}
	killed := make(map[token.Pos]bool)
	for _, x := range kill {
		killed[x] = true
	}
	for x := range in {
		if !killed[x] {
			out[x] = true
		}
	}
	return out
}

func (rd *ReachingDefinitions) GenKill(loc *BlockLocation) (gen, kill []token.Pos) {
	proc := func(e *ast.Ident) {
		if rd.info.Uses[e] == nil && rd.info.Defs[e] == nil {
			return
		}
		ref := rd.refs[e.Pos()]
		obj := rd.objs[ref.Oid]
		gen = append(gen, ref.Id)
		for redef := range obj.Redefs {
			if redef != ref.Id {
				kill = append(kill, redef)
			}
		}
	}
	stmt := rd.cfg.Blocks[loc.Block].Stmts[loc.Stmt]
	switch s := (*stmt).(type) {
	case *ast.IncDecStmt:
		switch e := s.X.(type) {
		case *ast.Ident:
			proc(e)
		}
	case *ast.AssignStmt:
		for _, expr := range s.Lhs {
			switch e := expr.(type) {
			case *ast.Ident:
				proc(e)
			}
		}
	}
	return gen, kill
}
