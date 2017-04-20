package analysis

import (
	"fmt"
	"strconv"
	"strings"
	"bytes"
	"go/ast"
	"go/token"
	"go/printer"
	"unsafe"
)

import (
	"github.com/timtadh/data-structures/errors"
)

func FmtNode(fset *token.FileSet, n ast.Node) string {
	var buf bytes.Buffer
	printer.Fprint(&buf, fset, n)
	return buf.String()
}

type CFG struct {
	FSet   *token.FileSet
	Name   string
	Fn     ast.Node
	Body   *[]ast.Stmt
	Blocks []*Block
	nodes  map[uintptr]*Block
	labels map[string]*Block
	loopHeaders []*Block
	exits []*Block
	nextCase []*Block
	breakLabel, continueLabel string
}

type Block struct {
	FSet       *token.FileSet
	Id         int
	Name       string
	Stmts      []*ast.Stmt
	Next       []*Flow
	Prev       []*Flow
	Body       *[]ast.Stmt
	StartsAt   int
	Cond       *ast.Expr
}

type Flow struct {
	FSet  *token.FileSet
	Block *Block
	Type  FlowType
	Comm  *ast.Stmt
	Cases *[]ast.Expr
}

type FlowType uint8

const (
	INVALID = iota
	Unconditional
	True
	False
	Range
	RangeExit
	Switch
	Select
	TypeSwitch
)

func BuildCFG(fset *token.FileSet, fnName string, fn ast.Node, body *[]ast.Stmt) *CFG {
	cfg := &CFG{
		FSet: fset,
		Name: fnName,
		Fn: fn,
		Body: body,
		Blocks: make([]*Block, 0, 10),
		nodes: make(map[uintptr]*Block),
		labels: make(map[string]*Block),
	}
	cfg.build()
	return cfg
}

func (c *CFG) String() string {
	blocks := make([]string, 0, len(c.Blocks))
	for _, b := range c.Blocks {
		blocks = append(blocks, b.String())
	}
	return fmt.Sprintf("fn %v\n%v", c.Name, strings.Join(blocks, "\n\n"))
}

func (c *CFG) Dotty() string {
	nodes := make([]string, 0, len(c.Blocks))
	edges := make([]string, 0, len(c.Blocks))
	for _, b := range c.Blocks {
		label := strconv.Quote(b.DotLabel())
		label = strings.Replace(label, "\\n", "\\l", -1)
		nodes = append(nodes, fmt.Sprintf("n%d [label=%v]", b.Id, label))
		for _, f := range b.Next {
			if f.Block != nil {
				edges = append(edges, fmt.Sprintf("n%d -> n%d [label=%v]", b.Id, f.Block.Id, strconv.Quote(f.DotLabel())))
			}
		}
	}
	return fmt.Sprintf(`digraph %v {
rankdir=LR
label=%v
labelloc=top
node [shape="rect", labeljust=l]
%v
%v
}`, strconv.Quote(c.Name), strconv.Quote(c.Name), strings.Join(nodes, "\n"), strings.Join(edges, "\n"))
}

func (c *CFG) nptr(n ast.Node) uintptr {
	type intr struct {
		typ uintptr
		data uintptr
	}
	return (*intr)(unsafe.Pointer(&n)).data
}

func (c *CFG) Block(node ast.Node) *Block {
	return c.nodes[c.nptr(node)]
}

func (c *CFG) GetClosestBlk(i int, blk []ast.Stmt, s ast.Node) *Block {
	// p := c.FSet.Position(s.Pos())
	// fmt.Printf("get-blk (i %v) %T %v %v \n", i, s, FmtNode(c.FSet, s), p)
	switch stmt := s.(type) {
	case *ast.BlockStmt:
		if len(stmt.List) > 0 {
			return c.GetClosestBlk(0, stmt.List, stmt.List[0])
		} else if i + 1 < len(blk) {
			return c.GetClosestBlk(i+1, blk, blk[i+1])
		} else if i - 1 >= 0 {
			return c.GetClosestBlk(i-1, blk, blk[i-1])
		} else {
			return nil
		}
	case *ast.LabeledStmt:
		return c.GetClosestBlk(i, blk, stmt.Stmt)
	default:
		b := c.Block(s)
		// if b != nil {
		// 	fmt.Printf("default %T %v : %d\n", stmt, FmtNode(c.FSet, stmt), b.Id)
		// } else {
		// 	fmt.Printf("default %T %v : <nil>\n", stmt, FmtNode(c.FSet, stmt))
		// }
		return b
	}
}

func (c *CFG) AddToBlk(blk *Block, node ast.Node) {
	c.nodes[c.nptr(node)] = blk
}

// Do not pass in a branching statement or function literal
func (c *CFG) AddAllToBlk(blk *Block, node ast.Node) {
	c.AddToBlk(blk, node)
	blkExprs(node, func(expr ast.Expr) {
		c.AddToBlk(blk, expr)
	})
}

func (c *CFG) build() {
	blk := c.addBlock(c.Body, 0)
	_ = c.visitStmts(c.Body, blk)
	// fmt.Println(c)
	c.filterEmpty()
	for _, blk := range c.Blocks {
		for _, s := range blk.Stmts {
			c.AddToBlk(blk, *s)
			blkExprs(*s, func(expr ast.Expr) {
				c.AddToBlk(blk, expr)
			})
		}
	}
}

func (c *CFG) filterEmpty() {
	for i := len(c.Blocks) - 1; i >= 0; i-- {
		blk := c.Blocks[i]
		// there are no stmts AND one or more prev blks ===> a next blk exists
		if len(blk.Stmts) == 0 && (len(blk.Prev) <= 0 || len(blk.Next) > 0) {
			err := c.removeBlock(blk)
			if err != nil {
				panic(err)
			}
		}
	}
}

func (c *CFG) visitStmts(stmts *[]ast.Stmt, blk *Block) *Block {
	for i := range *stmts {
		if blk != nil && blk.Body == nil {
			blk.Body = stmts
			blk.StartsAt = i
		}
		blk = c.visitStmt(i, stmts, &(*stmts)[i], blk)
	}
	return blk
}

// At some point I want to change this to:
//   func (c *CFG) visitStmt(i int, body *[]ast.Stmt, s *ast.Stmt, preds []*Flow) []*Flow 
//
// This would ensure that there would never be a dangling empty block. However, this
// requires large changes in every method. I don't want to do this re-write now. Side
// note it would be easier to have `preds` be a []*Block but you can't make that work
// with branch statements as their outgoing exits are not unconditional jumps.
//
func (c *CFG) visitStmt(i int, body *[]ast.Stmt, s *ast.Stmt, blk *Block) *Block {
	switch stmt := (*s).(type) {
	case *ast.LabeledStmt:
		blk = c.visitLabeledStmt(i, body, s, blk)
	case *ast.BranchStmt:
		blk = c.visitBranchStmt(i, body, s, blk)
	case *ast.BlockStmt:
		blk = c.visitBlockStmt(i, body, s, blk)
	case *ast.IfStmt:
		blk = c.visitIfStmt(i, body, s, blk)
	case *ast.ForStmt:
		blk = c.visitForStmt(i, body, s, blk)
	case *ast.RangeStmt:
		blk = c.visitRangeStmt(i, body, s, blk)
	case *ast.SelectStmt:
		blk = c.visitSelectStmt(i, body, s, blk)
	case *ast.TypeSwitchStmt:
		blk = c.visitTypeSwitchStmt(i, body, s, blk)
	case *ast.SwitchStmt:
		blk = c.visitSwitchStmt(i, body, s, blk)
	case *ast.CaseClause:
		panic(fmt.Errorf("Unexpected case clause %T %v", stmt, stmt))
	case *ast.CommClause:
		panic(fmt.Errorf("Unexpected comm clause %T %v", stmt, stmt))
	default:
		if blk == nil {
			blk = c.addBlock(body, i)
		}
		switch stmt := (*s).(type) {
		case *ast.BadStmt:
			blk.Add(s)
		case *ast.DeclStmt:
			blk.Add(s)
		case *ast.EmptyStmt:
			blk.Add(s)
		case *ast.ExprStmt:
			blk.Add(s)
		case *ast.SendStmt:
			blk.Add(s)
		case *ast.IncDecStmt:
			blk.Add(s)
		case *ast.AssignStmt:
			blk.Add(s)
		case *ast.GoStmt:
			blk.Add(s)
		case *ast.DeferStmt:
			blk.Add(s)
		case *ast.ReturnStmt:
			blk.Add(s)
		default:
			panic(fmt.Errorf("unexpected node %T", stmt))
		}
	}
	return blk
}

func (c *CFG) visitLabeledStmt(idx int, body *[]ast.Stmt, s *ast.Stmt, from *Block) *Block {
	stmt := (*s).(*ast.LabeledStmt)
	label := stmt.Label.Name
	var to *Block
	if b, has := c.labels[label]; has {
		to = b
	} else {
		to = c.addBlock(body, idx)
		to.Name = label
		c.labels[label] = to
	}
	if from != nil && len(from.Next) <= 0 {
		from.Link(&Flow{
			Block: to,
			Type: Unconditional,
		})
	}
	switch stmt.Stmt.(type) {
	case *ast.ForStmt, *ast.RangeStmt, *ast.SelectStmt, *ast.TypeSwitchStmt, *ast.SwitchStmt:
		c.breakLabel = label+"-break"
	}
	switch stmt.Stmt.(type) {
	case *ast.ForStmt, *ast.RangeStmt:
		c.continueLabel = label+"-continue"
	}
	if to.Body == nil {
		to.Body = body
		to.StartsAt = idx
	}
	return c.visitStmt(idx, body, &stmt.Stmt, to)
}

func (c *CFG) visitBranchStmt(idx int, body *[]ast.Stmt, s *ast.Stmt, from *Block) *Block {
	if from == nil {
		from = c.addBlock(body, idx)
	}
	stmt := (*s).(*ast.BranchStmt)
	from.Add(s)
	getLabel := func(label string) *Block {
		label = stmt.Label.Name
		if b, has := c.labels[label]; has {
			return b
		} else {
			x := c.addBlock(nil, -1)
			x.Name = label
			c.labels[label] = x
			return x
		}
	}
	var to *Block
	if stmt.Label != nil {
		if stmt.Tok == token.BREAK {
			label := stmt.Label.Name + "-break"
			if b, has := c.labels[label]; has {
				to = b
			} else {
				to = getLabel(stmt.Label.Name)
			}
		} else if stmt.Tok == token.CONTINUE {
			label := stmt.Label.Name + "-continue"
			if b, has := c.labels[label]; has {
				to = b
			} else {
				to = getLabel(stmt.Label.Name)
			}
		} else {
			to = getLabel(stmt.Label.Name)
		}
	} else if stmt.Tok == token.CONTINUE && len(c.loopHeaders) > 0 {
		to = c.loopHeaders[len(c.loopHeaders)-1]
	} else if stmt.Tok == token.BREAK && len(c.exits) > 0 {
		to = c.exits[len(c.exits)-1]
	} else if stmt.Tok == token.FALLTHROUGH && len(c.nextCase) > 0 && c.nextCase[len(c.nextCase)-1] != nil {
		to = c.nextCase[len(c.nextCase)-1]
	} else {
		panic(fmt.Errorf("%v can't be used here", FmtNode(c.FSet, stmt)))
	}
	from.Link(&Flow{
		Block: to,
		Type: Unconditional,
	})
	return nil
}

func (c *CFG) visitBlockStmt(idx int, body *[]ast.Stmt, s *ast.Stmt, blk *Block) *Block {
	stmt := (*s).(*ast.BlockStmt)
	return c.visitStmts(&stmt.List, blk)
}

func (c *CFG) visitIfStmt(idx int, body *[]ast.Stmt, s *ast.Stmt, entry *Block) *Block {
	stmt := (*s).(*ast.IfStmt)
	if entry == nil {
		entry = c.addBlock(body, idx)
	}
	if stmt.Init != nil {
		entry.Add(&stmt.Init)
	}
	entry.Add(s)
	entry.Cond = &stmt.Cond
	thenBlk := c.addBlock(nil, -1)
	var elseBlk *Block = nil
	if stmt.Else != nil {
		elseBlk = c.addBlock(nil, -1)
	}
	exitBlk := c.addBlock(nil, -1)
	{
		entry.Link(&Flow{
			Block: thenBlk,
			Type: True,
		})
		thenBody := ast.Stmt(stmt.Body)
		thenBlk = c.visitBlockStmt(idx, body, &thenBody, thenBlk)
		if thenBlk != nil && !thenBlk.Exits() {
			thenBlk.Link(&Flow{
				Block: exitBlk,
				Type: Unconditional,
			})
		}
	}
	if stmt.Else != nil {
		entry.Link(&Flow{
			Block: elseBlk,
			Type: False,
		})
		elseBlk = c.visitStmt(idx, body, &stmt.Else, elseBlk)
		if elseBlk != nil && !elseBlk.Exits() {
			elseBlk.Link(&Flow{
				Block: exitBlk,
				Type: Unconditional,
			})
		}
	} else {
		entry.Link(&Flow{
			Block: exitBlk,
			Type: False,
		})
	}
	return exitBlk
}

func (c *CFG) visitForStmt(idx int, stmts *[]ast.Stmt, s *ast.Stmt, entry *Block) *Block {
	stmt := (*s).(*ast.ForStmt)
	if entry == nil {
		entry = c.addBlock(stmts, idx)
	}
	if stmt.Init != nil {
		entry.Add(&stmt.Init)
	}
	header := c.addBlock(nil, -1)
	entry.Link(&Flow{
		Block: header,
		Type: Unconditional,
	})
	header.Add(s)
	body := ast.Stmt(stmt.Body)
	exitBlk := c.addBlock(nil, -1)
	if c.breakLabel != "" {
		c.labels[c.breakLabel] = exitBlk
		c.breakLabel = ""
	}
	if c.continueLabel != "" {
		c.labels[c.continueLabel] = header
		c.continueLabel = ""
	}
	var bodyBlk *Block = nil
	if stmt.Cond != nil {
		bodyBlk = c.addBlock(nil, -1)
		header.Cond = &stmt.Cond
		header.Link(&Flow{
			Block: bodyBlk,
			Type: True,
		})
		header.Link(&Flow{
			Block: exitBlk,
			Type: False,
		})
	} else {
		bodyBlk = header
	}

	var postBlk *Block
	if stmt.Post != nil {
		postBlk = c.addBlock(nil, -1)
		postBlk = c.visitStmt(idx, stmts, &stmt.Post, postBlk)
		postBlk.Link(&Flow{
			Block: header,
			Type: Unconditional,
		})
	}

	if postBlk != nil {
		c.pushLoop(postBlk, exitBlk)
	} else {
		c.pushLoop(header, exitBlk)
	}
	bodyBlk = c.visitBlockStmt(idx, stmts, &body, bodyBlk)
	c.popLoop()

	if postBlk != nil && bodyBlk != nil {
		bodyBlk.Link(&Flow{
			Block: postBlk,
			Type: Unconditional,
		})
	} else if postBlk == nil && bodyBlk != nil {
		bodyBlk.Link(&Flow{
			Block: header,
			Type: Unconditional,
		})
	}
	return exitBlk
}

func (c *CFG) pushLoop(header, exit *Block) {
	c.loopHeaders = append(c.loopHeaders, header)
	c.exits = append(c.exits, exit)
}

func (c *CFG) popLoop() {
	c.loopHeaders = c.loopHeaders[:len(c.loopHeaders)-1]
	c.exits = c.exits[:len(c.exits)-1]
}

func (c *CFG) visitRangeStmt(idx int, stmts *[]ast.Stmt, s *ast.Stmt, entry *Block) *Block {
	stmt := (*s).(*ast.RangeStmt)
	if entry == nil {
		entry = c.addBlock(stmts, idx)
	}
	header := c.addBlock(nil, -1)
	entry.Link(&Flow{
		Block: header,
		Type: Unconditional,
	})
	header.Add(s)
	body := ast.Stmt(stmt.Body)
	bodyBlk := c.addBlock(nil, -1)
	exitBlk := c.addBlock(nil, -1)
	if c.breakLabel != "" {
		c.labels[c.breakLabel] = exitBlk
		c.breakLabel = ""
	}
	if c.continueLabel != "" {
		c.labels[c.continueLabel] = header
		c.continueLabel = ""
	}
	header.Cond = &stmt.X
	header.Link(&Flow{
		Block: bodyBlk,
		Type: Range,
	})
	header.Link(&Flow{
		Block: exitBlk,
		Type: RangeExit,
	})

	c.pushLoop(header, exitBlk)
	bodyBlk = c.visitBlockStmt(idx, stmts, &body, bodyBlk)
	c.popLoop()

	if bodyBlk != nil {
		bodyBlk.Link(&Flow{
			Block: header,
			Type: Unconditional,
		})
	}
	return exitBlk
}

func (c *CFG) visitSelectStmt(idx int, body *[]ast.Stmt, s *ast.Stmt, entry *Block) *Block {
	stmt := (*s).(*ast.SelectStmt)
	if entry == nil {
		entry = c.addBlock(body, idx)
	}
	entry.Add(s)
	if len(stmt.Body.List) <= 0 {
		return entry
	}
	exit := c.addBlock(nil, -1)
	if c.breakLabel != "" {
		c.labels[c.breakLabel] = exit
		c.breakLabel = ""
	}
	for i, s := range stmt.Body.List {
		comm := s.(*ast.CommClause)
		commBlk := c.addBlock(nil, -1)
		var cond *ast.Stmt = nil
		if comm.Comm != nil {
			cond = &comm.Comm
		}
		entry.Link(&Flow{
			FSet: c.FSet,
			Block: commBlk,
			Type: Select,
			Comm: cond,
		})
		if cond != nil {
			commBlk = c.visitStmt(i, &stmt.Body.List, cond, commBlk)
		}
		commBlk = c.visitStmts(&comm.Body, commBlk)
		if commBlk != nil {
			commBlk.Link(&Flow{
				Block: exit,
				Type: Unconditional,
			})
		}
	}
	return exit
}

func (c *CFG) visitTypeSwitchStmt(idx int, body *[]ast.Stmt, s *ast.Stmt, entry *Block) *Block {
	stmt := (*s).(*ast.TypeSwitchStmt)
	if entry == nil {
		entry = c.addBlock(body, idx)
	}
	if stmt.Init != nil {
		entry.Add(&stmt.Init)
	}
	entry.Add(s)
	if len(stmt.Body.List) <= 0 {
		return entry
	}
	exit := c.addBlock(nil, -1)
	if c.breakLabel != "" {
		c.labels[c.breakLabel] = exit
		c.breakLabel = ""
	}
	for _, s := range stmt.Body.List {
		cas := s.(*ast.CaseClause)
		caseBlk := c.addBlock(nil, -1)
		var cases *[]ast.Expr = nil
		if cas.List != nil {
			cases = &cas.List
		}
		entry.Link(&Flow{
			FSet: c.FSet,
			Block: caseBlk,
			Type: TypeSwitch,
			Cases: cases,
		})
		c.pushSwitch(nil, exit)
		caseBlk = c.visitStmts(&cas.Body, caseBlk)
		c.popSwitch()
		if caseBlk != nil {
			caseBlk.Link(&Flow{
				Block: exit,
				Type: Unconditional,
			})
		}
	}
	return exit
}

func (c *CFG) visitSwitchStmt(idx int, body *[]ast.Stmt, s *ast.Stmt, entry *Block) *Block {
	stmt := (*s).(*ast.SwitchStmt)
	if entry == nil {
		entry = c.addBlock(body, idx)
	}
	if stmt.Init != nil {
		entry.Add(&stmt.Init)
	}
	entry.Add(s)
	entry.Cond = &stmt.Tag
	if len(stmt.Body.List) <= 0 {
		return entry
	}
	exit := c.addBlock(nil, -1)
	if c.breakLabel != "" {
		c.labels[c.breakLabel] = exit
		c.breakLabel = ""
	}
	if len(stmt.Body.List) <= 0 {
	}
	blks := make([]*Block, 0, len(stmt.Body.List))
	for range stmt.Body.List {
		blks = append(blks, c.addBlock(nil, -1))
	}
	for i, s := range stmt.Body.List {
		cas := s.(*ast.CaseClause)
		caseBlk := blks[i]
		var cases *[]ast.Expr = nil
		if cas.List != nil {
			cases = &cas.List
		}
		entry.Link(&Flow{
			FSet: c.FSet,
			Block: caseBlk,
			Type: Switch,
			Cases: cases,
		})
		if i + 1 < len(blks) {
			c.pushSwitch(blks[i+1], exit)
		} else {
			c.pushSwitch(nil, exit)
		}
		caseBlk = c.visitStmts(&cas.Body, caseBlk)
		c.popSwitch()
		if caseBlk != nil {
			caseBlk.Link(&Flow{
				Block: exit,
				Type: Unconditional,
			})
		}
	}
	return exit
}

func (c *CFG) pushSwitch(next, exit *Block) {
	c.nextCase = append(c.nextCase, next)
	c.exits = append(c.exits, exit)
}

func (c *CFG) popSwitch() {
	c.nextCase = c.nextCase[:len(c.nextCase)-1]
	c.exits = c.exits[:len(c.exits)-1]
}

func (c *CFG) addBlock(body *[]ast.Stmt, idx int) *Block {
	id := len(c.Blocks)
	blk := NewBlock(c.FSet, id, body, idx)
	c.Blocks = append(c.Blocks, blk)
	return blk
}

func (c *CFG) removeBlock(b *Block) error {
	if len(b.Next) > 1 {
		return errors.Errorf("Cannot remove block because it has more than 1 successor:\n%v", b)
	}
	if len(b.Next) == 0 && len(b.Prev) > 0 {
		return errors.Errorf("Cannot remove block because it has 0 successors:\n%v", b)
	}
	var fn *Flow
	remove := make([]int, 0, 10)
	if len(b.Next) > 0 {
		fn = b.Next[0] // must have 1 element from assertions above
		for i, flowTo := range fn.Block.Prev {
			if flowTo.Block.Id != b.Id {
				continue
			}
			remove = append(remove, i)
		}
	}
	// remove the flows from the next block in reverse order
	// there should only be one to remove, this is defensive.
	for x := len(remove) - 1; x >= 0; x-- {
		i := remove[x]
		dst := fn.Block.Prev[i : len(fn.Block.Prev)-1]
		src := fn.Block.Prev[i+1 : len(fn.Block.Prev)]
		copy(dst, src)
		fn.Block.Prev = fn.Block.Prev[:len(fn.Block.Prev)-1]
	}
	// hook up the new flows
	for _, fp := range b.Prev {
		found := false
		for _, flowFrom := range fp.Block.Next {
			if flowFrom.Block.Id != b.Id {
				continue
			}
			flowFrom.Block = fn.Block
			fn.Block.Prev = append(fn.Block.Prev, &Flow{
				FSet: flowFrom.FSet,
				Block: fp.Block,
				Type: flowFrom.Type,
				Comm: flowFrom.Comm,
				Cases: flowFrom.Cases,
			})
			found = true
			break
		}
		if !found {
			return errors.Errorf("Flow from blk-%d to blk-%d but could not find matching flows:\n%v\n\n%v",
				fp.Block.Id, b.Id, fp.Block, b)
		}
	}
	dst := c.Blocks[b.Id : len(c.Blocks)-1]
	src := c.Blocks[b.Id+1 : len(c.Blocks)]
	copy(dst, src)
	c.Blocks = c.Blocks[:len(c.Blocks)-1]
	for i, blk := range c.Blocks {
		blk.Id = i
		for _, f := range blk.Next {
			if f.Block == b {
				return errors.Errorf("Found flow from blk-%d to removed blk: \n%v\n\n%v",
					f.Block.Id, f.Block, b)
			}
		}
	}
	return nil
}

func NewBlock(fset *token.FileSet, id int, body *[]ast.Stmt, startsAt int) *Block {
	return &Block{
		FSet: fset,
		Id: id,
		Stmts: make([]*ast.Stmt, 0, 10),
		Next: make([]*Flow, 0, 2),
		Prev: make([]*Flow, 0, 2),
		Body: body,
		StartsAt: startsAt,
	}
}

func (b *Block) Add(stmt *ast.Stmt) {
	b.Stmts = append(b.Stmts, stmt)
}

func (b *Block) Link(flow *Flow) {
	b.Next = append(b.Next, flow)
	flow.Block.Prev = append(flow.Block.Prev, &Flow{
		FSet: flow.FSet,
		Block: b,
		Type: flow.Type,
		Comm: flow.Comm,
		Cases: flow.Cases,
	})
}

func (b *Block) Exits() bool {
	if b == nil || len(b.Stmts) <= 0 {
		return false
	}
	s := b.Stmts[len(b.Stmts)-1]
	switch (*s).(type) {
	case *ast.ReturnStmt:
		return true
	default:
		return ContainsPanic(*s) || ContainsOsExit(*s)
	}
}

func (b *Block) String() string {
	insts := make([]string, 0, len(b.Stmts))
	if len(b.Stmts) > 0 {
		insts = append(insts, "")
	}
	for _, s := range b.Stmts {
		switch stmt := (*s).(type) {
		default:
			n := fmt.Sprintf("%T %v", stmt, FmtNode(b.FSet, stmt))
			insts = append(insts, n)
		case *ast.IfStmt:
			insts = append(insts, fmt.Sprintf("if %v", FmtNode(b.FSet, stmt.Cond)))
		case *ast.ForStmt:
			cond := ""
			if stmt.Cond != nil {
				cond = " " + FmtNode(b.FSet, stmt.Cond)
			}
			insts = append(insts, fmt.Sprintf("for%v", cond))
		case *ast.SelectStmt:
			insts = append(insts, fmt.Sprintf("select"))
		case *ast.SwitchStmt:
			tag := ""
			if stmt.Tag != nil {
				tag = " " + FmtNode(b.FSet, stmt.Tag)
			}
			insts = append(insts, fmt.Sprintf("switch%v", tag))
		case *ast.TypeSwitchStmt:
			insts = append(insts, fmt.Sprintf("type-switch %v", FmtNode(b.FSet, stmt.Assign)))
		case *ast.RangeStmt:
			kv := ""
			if stmt.Key != nil {
				kv = FmtNode(b.FSet, stmt.Key)
			}
			if stmt.Value != nil {
				kv += ", " + FmtNode(b.FSet, stmt.Value)
			}
			if kv != "" {
				kv += " := "
			}
			x := FmtNode(b.FSet, stmt.X)
			insts = append(insts, fmt.Sprintf("for %vrange %v", kv, x))
		}
	}
	nextFlows := make([]string, 0, len(b.Next))
	for _, f := range b.Next {
		nextFlows = append(nextFlows, f.String())
	}
	next := strings.Join(nextFlows, ", ")
	prevFlows := make([]string, 0, len(b.Prev))
	for _, f := range b.Prev {
		prevFlows = append(prevFlows, f.String())
	}
	prev := strings.Join(prevFlows, ", ")
	stmts := strings.Join(insts, "\n\t")
	name := ""
	if b.Name != "" {
		name = fmt.Sprintf(" label: %v ", b.Name)
	}
	// body := ""
	// if b.Body != nil {
	// 	body = fmt.Sprintf("\n\tStarts at (%d) <%v> ", b.StartsAt, FmtNode(b.FSet, (*b.Body)[b.StartsAt]))
	// }
	return fmt.Sprintf("Block %v%v%v\n\tNext: %v\n\tPrev: %v", b.Id, name, stmts, next, prev)
}

func (b *Block) DotLabel() string {
	insts := make([]string, 0, len(b.Stmts))
	insts = append(insts, fmt.Sprintf("blk-%v", b.Id+1))
	for _, s := range b.Stmts {
		switch stmt := (*s).(type) {
		default:
			n := fmt.Sprintf("%v", FmtNode(b.FSet, stmt))
			insts = append(insts, n)
		case *ast.IfStmt:
			insts = append(insts, fmt.Sprintf("if %v", FmtNode(b.FSet, stmt.Cond)))
		case *ast.ForStmt:
			cond := ""
			if stmt.Cond != nil {
				cond = " " + FmtNode(b.FSet, stmt.Cond)
			}
			insts = append(insts, fmt.Sprintf("for%v", cond))
		case *ast.SelectStmt:
			insts = append(insts, fmt.Sprintf("select"))
		case *ast.SwitchStmt:
			tag := ""
			if stmt.Tag != nil {
				tag = " " + FmtNode(b.FSet, stmt.Tag)
			}
			insts = append(insts, fmt.Sprintf("switch%v", tag))
		case *ast.TypeSwitchStmt:
			insts = append(insts, fmt.Sprintf("type-switch %v", FmtNode(b.FSet, stmt.Assign)))
		case *ast.RangeStmt:
			kv := ""
			if stmt.Key != nil {
				kv = FmtNode(b.FSet, stmt.Key)
			}
			if stmt.Value != nil {
				kv += ", " + FmtNode(b.FSet, stmt.Value)
			}
			if kv != "" {
				kv += " := "
			}
			x := FmtNode(b.FSet, stmt.X)
			insts = append(insts, fmt.Sprintf("for %vrange %v", kv, x))
		}
	}
	stmts := strings.Join(insts, "\n")
	return fmt.Sprintf("%v\n", stmts)
}

func (f *Flow) String() string {
	comm := ""
	if f.Comm != nil {
		comm = fmt.Sprintf(" with comm: %v", FmtNode(f.FSet, *f.Comm))
	}
	cases := ""
	if f.Cases != nil {
		parts := make([]string, 0, len(*f.Cases))
		for _, c := range *f.Cases {
			parts = append(parts, FmtNode(f.FSet, c))
		}
		cases = " with cases: " + strings.Join(parts, ", ")
	}
	when := ""
	if f.Type != Unconditional {
		when = fmt.Sprintf(" on %v", f.Type)
	}
	return fmt.Sprintf("(goto %v%v%v%v)", f.Block.Id, when, comm, cases)
}

func (f *Flow) DotLabel() string {
	comm := ""
	if f.Comm != nil {
		comm = fmt.Sprintf("on comm: %v", FmtNode(f.FSet, *f.Comm))
	}
	cases := ""
	if f.Cases != nil {
		parts := make([]string, 0, len(*f.Cases))
		for _, c := range *f.Cases {
			parts = append(parts, FmtNode(f.FSet, c))
		}
		cases = "on cases: " + strings.Join(parts, ", ")
	}
	when := ""
	if f.Type != Unconditional {
		when = fmt.Sprintf("on %v", f.Type)
	}
	return fmt.Sprintf("%v%v%v", when, comm, cases)
}

func (t FlowType) String() string {
	switch t {
	case INVALID: return "INVALID"
	case Unconditional: return "Unconditional"
	case True: return "True"
	case False: return "False"
	case Range: return "Range"
	case RangeExit: return "RangeExit"
	case Switch: return "Switch"
	case Select: return "Select"
	case TypeSwitch: return "TypeSwitch"
	}
	return "INVALID"
}

