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
	loopExits []*Block
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
	Cond  *ast.Expr
}

type FlowType uint8

const (
	INVALID = iota
	Unconditional
	TrueBranch
	FalseBranch
	Switch
	Select
	TypeSwitch
	Panic
)

// type cfgBuilder struct {
// 	cfg *CFG
// 	stack []*Block
// 	err error
// }

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
		blk = c.visitStmt(&(*stmts)[i], blk)
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
	case *ast.SelectStmt:
	case *ast.CaseClause:
	case *ast.CommClause:
	case *ast.SwitchStmt:
	case *ast.TypeSwitchStmt:
	case *ast.RangeStmt:
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
	if len(from.Next) <= 0 {
		from.Link(&Flow{
			Block: to,
			Type: Unconditional,
		})
	}
	return c.visitStmt(&stmt.Stmt, to)
}

func (c *CFG) visitBranchStmt(s *ast.Stmt, from *Block) *Block {
	from.Add(s)
	stmt := (*s).(*ast.BranchStmt)

	var to *Block
	if stmt.Label != nil {
		label := stmt.Label.Name
		if b, has := c.labels[label]; has {
			to = b
		} else {
			to = c.addBlock()
			to.Name = label
			c.labels[label] = to
		}
	} else if len(c.loopHeaders) > 0 && (stmt.Tok == token.BREAK || stmt.Tok == token.CONTINUE) {
		if stmt.Tok == token.BREAK {
			to = c.loopExits[len(c.loopExits)-1]
		} else {
			to = c.loopHeaders[len(c.loopHeaders)-1]
		}
	} else {
		panic(fmt.Errorf("%v outside of a loop", stmt.Tok))
	}
	from.Link(&Flow{
		Block: to,
		Type: Unconditional,
	})
	return c.addBlock()
}

func (c *CFG) visitBlockStmt(s *ast.Stmt, blk *Block) *Block {
	stmt := (*s).(*ast.BlockStmt)
	blk = c.visitStmts(&stmt.List, blk)
	return blk
}

func (c *CFG) visitIfStmt(s *ast.Stmt, entry *Block) *Block {
	stmt := (*s).(*ast.IfStmt)
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
			Type: TrueBranch,
		})
		thenBody := ast.Stmt(stmt.Body)
		thenBlk = c.visitBlockStmt(&thenBody, thenBlk)
		if !thenBlk.Exits() {
			thenBlk.Link(&Flow{
				Block: exitBlk,
				Type: Unconditional,
			})
		}
	}
	if stmt.Else != nil {
		entry.Link(&Flow{
			Block: elseBlk,
			Type: FalseBranch,
		})
		elseBlk = c.visitStmt(&stmt.Else, elseBlk)
		if !elseBlk.Exits() {
			elseBlk.Link(&Flow{
				Block: exitBlk,
				Type: Unconditional,
			})
		}
	} else {
		entry.Link(&Flow{
			Block: exitBlk,
			Type: FalseBranch,
		})
	}
	return exitBlk
}

func (c *CFG) visitForStmt(s *ast.Stmt, entry *Block) *Block {
	stmt := (*s).(*ast.ForStmt)
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
	var bodyBlk *Block = nil
	if stmt.Cond != nil {
		bodyBlk = c.addBlock()
		header.Cond = &stmt.Cond
		header.Link(&Flow{
			Block: bodyBlk,
			Type: TrueBranch,
		})
		header.Link(&Flow{
			Block: exitBlk,
			Type: FalseBranch,
		})
	} else {
		bodyBlk = header
	}

	c.pushLoop(header, exitBlk)
	bodyBlk = c.visitBlockStmt(&body, bodyBlk)
	c.popLoop()

	if stmt.Post != nil {
		bodyBlk = c.visitStmt(&stmt.Post, bodyBlk)
	}
	bodyBlk.Link(&Flow{
		Block: header,
		Type: Unconditional,
	})
	return exitBlk
}

func (c *CFG) pushLoop(header, exit *Block) {
	c.loopHeaders = append(c.loopHeaders, header)
	c.loopExits = append(c.loopExits, exit)
}

func (c *CFG) popLoop() {
	c.loopHeaders = c.loopHeaders[:len(c.loopHeaders)-1]
	c.loopExits = c.loopExits[:len(c.loopExits)-1]
}

// func (b *cfgBuilder) build() (*CFG, error) {
// 	ast.Walk(b, b.cfg.Fn)
// 	if b.err != nil {
// 		return nil, b.err
// 	}
// 	return b.cfg, nil
// }
// 
// func (b *cfgBuilder) Visit(n ast.Node) (ast.Visitor) {
// 	return b
// }
// 
// func (b *cfgBuilder) push(blk *Block) {
// 	b.stack = append(b.stack, blk)
// }
// 
// func (b *cfgBuilder) pop() *Block {
// 	blk := b.stack[len(b.stack)-1]
// 	b.stack = b.stack[:len(b.stack)-1]
// 	return blk
// }

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
		Cond: flow.Cond,
	})
}

func (b *Block) Exits() bool {
	if len(b.Stmts) <= 0 {
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
	cond := ""
	if f.Cond != nil {
		cond = fmt.Sprintf(" with cond %v", FmtNode(f.FSet, *f.Cond))
	}
	return fmt.Sprintf("flow to %v of type %v%v", f.Block.Id, f.Type, cond)
}

func (t FlowType) String() string {
	switch t {
	case INVALID: return "INVALID"
	case Unconditional: return "Unconditional"
	case TrueBranch: return "TrueBranch"
	case FalseBranch: return "FalseBranch"
	case Switch: return "Switch"
	case Select: return "Select"
	case TypeSwitch: return "TypeSwitch"
	case Panic: return "Panic"
	}
	return "INVALID"
}

