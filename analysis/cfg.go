package analysis

import (
	"fmt"
	"strings"
	"bytes"
	"go/ast"
	"go/token"
	"go/printer"
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
	labels map[string]*Block
	loopHeaders []*Block
	exits []*Block
	nextCase []*Block
	breakLabel, continueLabel string
}

type Block struct {
	FSet  *token.FileSet
	Id    int
	Name  string
	Stmts []*ast.Stmt
	Next  []*Flow
	Prev  []*Flow
	Cond  *ast.Expr
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

func (c *CFG) build() {
	blk := c.addBlock()
	_ = c.visitStmts(c.Body, blk)
}

func (c *CFG) visitStmts(stmts *[]ast.Stmt, blk *Block) *Block {
	for i := range *stmts {
		x := c.visitStmt(&(*stmts)[i], blk)
		if x == nil && i + 1 < len(*stmts) {
			// blk = c.addBlock()
			blk = nil
		} else {
			blk = x
		}
	}
	return blk
}

func (c *CFG) visitStmt(s *ast.Stmt, blk *Block) *Block {
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
	case *ast.LabeledStmt:
		blk = c.visitLabeledStmt(s, blk)
	case *ast.BranchStmt:
		blk = c.visitBranchStmt(s, blk)
	case *ast.BlockStmt:
		blk = c.visitBlockStmt(s, blk)
	case *ast.IfStmt:
		blk = c.visitIfStmt(s, blk)
	case *ast.ForStmt:
		blk = c.visitForStmt(s, blk)
	case *ast.RangeStmt:
		blk = c.visitRangeStmt(s, blk)
	case *ast.SelectStmt:
		blk = c.visitSelectStmt(s, blk)
	case *ast.TypeSwitchStmt:
		blk = c.visitTypeSwitchStmt(s, blk)
	case *ast.SwitchStmt:
		blk = c.visitSwitchStmt(s, blk)
	case *ast.CaseClause:
		panic(fmt.Errorf("Unexpected case clause %T %v", stmt, stmt))
	case *ast.CommClause:
		panic(fmt.Errorf("Unexpected comm clause %T %v", stmt, stmt))
	default:
		panic(fmt.Errorf("unexpected node %T", stmt))
	}
	return blk
}

func (c *CFG) visitLabeledStmt(s *ast.Stmt, from *Block) *Block {
	stmt := (*s).(*ast.LabeledStmt)
	label := stmt.Label.Name
	var to *Block
	if b, has := c.labels[label]; has {
		to = b
	} else {
		to = c.addBlock()
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
	return c.visitStmt(&stmt.Stmt, to)
}

func (c *CFG) visitBranchStmt(s *ast.Stmt, from *Block) *Block {
	if from == nil {
		from = c.addBlock()
	}
	from.Add(s)
	stmt := (*s).(*ast.BranchStmt)
	getLabel := func(label string) *Block {
		label = stmt.Label.Name
		if b, has := c.labels[label]; has {
			return b
		} else {
			x := c.addBlock()
			x.Name = label
			c.labels[label] = x
			return x
		}
	}
	var to *Block
	if stmt.Label != nil {
		if stmt.Tok == token.BREAK {
			label := stmt.Label.Name + "-break"
			fmt.Println(label, c.labels)
			if b, has := c.labels[label]; has {
				to = b
			} else {
				to = getLabel(stmt.Label.Name)
			}
		} else if stmt.Tok == token.CONTINUE {
			label := stmt.Label.Name + "-continue"
			fmt.Println(label, c.labels)
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

func (c *CFG) visitBlockStmt(s *ast.Stmt, blk *Block) *Block {
	stmt := (*s).(*ast.BlockStmt)
	blk = c.visitStmts(&stmt.List, blk)
	return blk
}

func (c *CFG) visitIfStmt(s *ast.Stmt, entry *Block) *Block {
	stmt := (*s).(*ast.IfStmt)
	if entry == nil {
		entry = c.addBlock()
	}
	entry.Add(s)
	entry.Cond = &stmt.Cond
	thenBlk := c.addBlock()
	var elseBlk *Block = nil
	if stmt.Else != nil {
		elseBlk = c.addBlock()
	}
	exitBlk := c.addBlock()
	{
		entry.Link(&Flow{
			Block: thenBlk,
			Type: True,
		})
		thenBody := ast.Stmt(stmt.Body)
		thenBlk = c.visitBlockStmt(&thenBody, thenBlk)
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
		elseBlk = c.visitStmt(&stmt.Else, elseBlk)
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

func (c *CFG) visitForStmt(s *ast.Stmt, entry *Block) *Block {
	stmt := (*s).(*ast.ForStmt)
	if entry == nil {
		entry = c.addBlock()
	}
	if stmt.Init != nil {
		entry.Add(&stmt.Init)
	}
	header := c.addBlock()
	entry.Link(&Flow{
		Block: header,
		Type: Unconditional,
	})
	header.Add(s)
	body := ast.Stmt(stmt.Body)
	exitBlk := c.addBlock()
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
		bodyBlk = c.addBlock()
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
		postBlk = c.addBlock()
		postBlk = c.visitStmt(&stmt.Post, postBlk)
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
	bodyBlk = c.visitBlockStmt(&body, bodyBlk)
	c.popLoop()

	if postBlk == nil && bodyBlk != nil {
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

func (c *CFG) visitRangeStmt(s *ast.Stmt, entry *Block) *Block {
	stmt := (*s).(*ast.RangeStmt)
	header := c.addBlock()
	if entry == nil {
		entry.Link(&Flow{
			Block: header,
			Type: Unconditional,
		})
	}
	header.Add(s)
	body := ast.Stmt(stmt.Body)
	bodyBlk := c.addBlock()
	exitBlk := c.addBlock()
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
	bodyBlk = c.visitBlockStmt(&body, bodyBlk)
	c.popLoop()

	if bodyBlk != nil {
		bodyBlk.Link(&Flow{
			Block: header,
			Type: Unconditional,
		})
	}
	return exitBlk
}

func (c *CFG) visitSelectStmt(s *ast.Stmt, entry *Block) *Block {
	stmt := (*s).(*ast.SelectStmt)
	if entry == nil {
		entry = c.addBlock()
	}
	entry.Add(s)
	if len(stmt.Body.List) <= 0 {
		return entry
	}
	exit := c.addBlock()
	if c.breakLabel != "" {
		c.labels[c.breakLabel] = exit
		c.breakLabel = ""
	}
	for _, s := range stmt.Body.List {
		comm := s.(*ast.CommClause)
		commBlk := c.addBlock()
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

func (c *CFG) visitTypeSwitchStmt(s *ast.Stmt, entry *Block) *Block {
	stmt := (*s).(*ast.TypeSwitchStmt)
	if entry == nil {
		entry = c.addBlock()
	}
	if stmt.Init != nil {
		entry.Add(&stmt.Init)
	}
	entry.Add(s)
	if len(stmt.Body.List) <= 0 {
		return entry
	}
	exit := c.addBlock()
	if c.breakLabel != "" {
		c.labels[c.breakLabel] = exit
		c.breakLabel = ""
	}
	for _, s := range stmt.Body.List {
		cas := s.(*ast.CaseClause)
		caseBlk := c.addBlock()
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

func (c *CFG) visitSwitchStmt(s *ast.Stmt, entry *Block) *Block {
	stmt := (*s).(*ast.SwitchStmt)
	if entry == nil {
		entry = c.addBlock()
	}
	if stmt.Init != nil {
		entry.Add(&stmt.Init)
	}
	entry.Add(s)
	entry.Cond = &stmt.Tag
	if len(stmt.Body.List) <= 0 {
		return entry
	}
	exit := c.addBlock()
	if c.breakLabel != "" {
		c.labels[c.breakLabel] = exit
		c.breakLabel = ""
	}
	if len(stmt.Body.List) <= 0 {
	}
	blks := make([]*Block, 0, len(stmt.Body.List))
	for range stmt.Body.List {
		blks = append(blks, c.addBlock())
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

func (c *CFG) addBlock() *Block {
	id := len(c.Blocks)
	blk := NewBlock(c.FSet, id)
	c.Blocks = append(c.Blocks, blk)
	return blk
}

func NewBlock(fset *token.FileSet, id int) *Block {
	return &Block{
		FSet: fset,
		Id: id,
		Stmts: make([]*ast.Stmt, 0, 10),
		Next: make([]*Flow, 0, 2),
		Prev: make([]*Flow, 0, 2),
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
	branches := make([]string, 0, len(b.Next))
	for _, f := range b.Next {
		branches = append(branches, f.String())
	}
	next := strings.Join(branches, ", ")
	stmts := strings.Join(insts, "\n\t")
	name := ""
	if b.Name != "" {
		name = fmt.Sprintf(" label: %v ", b.Name)
	}
	return fmt.Sprintf("Block %v%v%v\n\tNext: %v", b.Id, name, stmts, next)
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

