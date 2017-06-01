package analysis

import (
	"fmt"
	"strings"
)

type DominatorTree struct {
	roots    []*Block
	parent   map[*Block]*Block
	children map[*Block][]*Block
	succ     func(*Block) []*Block
	pred     func(*Block) []*Block
}

type DominatorFrontier struct {
	frontier map[*Block]map[*Block]bool
}

func (t *DominatorTree) Roots() []*Block {
	roots := make([]*Block, len(t.roots))
	copy(roots, t.roots)
	return roots
}

func (t *DominatorTree) Children(blk *Block) []*Block {
	kids := t.children[blk]
	children := make([]*Block, len(kids))
	copy(children, kids)
	return children
}

func (t *DominatorTree) Parent(blk *Block) *Block {
	return t.parent[blk]
}

func (t *DominatorTree) IDom(blk *Block) *Block {
	return t.parent[blk]
}

func (t *DominatorTree) Frontier() *DominatorFrontier {
	frontier := make(map[*Block]map[*Block]bool)
	var postfix func(*Block)
	postfix = func(blk *Block) {
		fmt.Println("postfix blk", blk.Id+1)
		for _, kid := range t.Children(blk) {
			fmt.Println("    kid blk", kid.Id+1)
			postfix(kid)
		}
		fmt.Println("  frontier blk", blk.Id+1)
		frontier[blk] = make(map[*Block]bool)
		for _, y := range t.succ(blk) {
			if t.IDom(y) != blk {
				frontier[blk][y] = true
			}
		}
		for _, kid := range t.Children(blk) {
			for y := range frontier[kid] {
				if t.IDom(y) != blk {
					frontier[blk][y] = true
				}
			}
		}
	}
	for _, r := range t.roots {
		postfix(r)
	}
	return &DominatorFrontier{frontier}
}

func (t *DominatorTree) String() string {
	type entry struct {
		n *Block
		j int
	}
	lines := make([]string, 0, 10)
	roots := make([]string, 0, len(t.roots))
	stack := make([]entry, 0, 10)
	for _, r := range t.roots {
		roots = append(roots, fmt.Sprintf("blk-%d", r.Id+1))
		stack = append(stack, entry{r, 0})
	}
	lines = append(lines, strings.Join(roots, ", "))
	for len(stack) > 0 {
		var e entry
		stack, e = stack[:len(stack)-1], stack[len(stack)-1]
		if e.j == 0 {
			lines = append(lines, fmt.Sprintf("%d : blk-%d", len(t.children[e.n]), e.n.Id+1))
		}
		kids := t.children[e.n]
		if e.j < len(kids) {
			kid := kids[e.j]
			stack = append(stack, entry{e.n, e.j + 1})
			stack = append(stack, entry{kid, 0})
		}
	}
	return strings.Join(lines, "\n")
}

func (f *DominatorFrontier) String() string {
	lines := make([]string, 0, 10)
	for blk, frontier := range f.frontier {
		lines = append(lines, fmt.Sprintf("blk-%d", blk.Id+1))
		for x := range frontier {
			lines = append(lines, fmt.Sprintf("    blk-%d", x.Id+1))
		}
	}
	return strings.Join(lines, "\n")
}

func Dominators(cfg *CFG) *DominatorTree {
	return dominators(cfg, len(cfg.Blocks), cfg.Blocks[0],
		func(blk *Block) []*Block {
			next := make([]*Block, 0, len(blk.Next))
			for _, flow := range blk.Next {
				next = append(next, flow.Block)
			}
			return next
		},
		func(blk *Block) []*Block {
			prev := make([]*Block, 0, len(blk.Prev))
			for _, flow := range blk.Prev {
				prev = append(prev, flow.Block)
			}
			return prev
		},
	)
}

func PostDominators(cfg *CFG) *DominatorTree {
	id := len(cfg.Blocks)
	exit := NewBlock(cfg.FSet, id, nil, -1)
	exits := make([]*Block, 0, 10)
	for _, blk := range cfg.Blocks {
		if len(blk.Next) == 0 {
			exits = append(exits, blk)
			blk.Link(&Flow{
				Block: exit,
				Type:  Unconditional,
			})
		}
	}
	t := dominators(cfg, len(cfg.Blocks)+1, exit,
		func(blk *Block) []*Block {
			prev := make([]*Block, 0, len(blk.Prev))
			for _, flow := range blk.Prev {
				prev = append(prev, flow.Block)
			}
			return prev
		},
		func(blk *Block) []*Block {
			next := make([]*Block, 0, len(blk.Next))
			for _, flow := range blk.Next {
				next = append(next, flow.Block)
			}
			return next
		},
	)
	for _, blk := range exits {
		blk.Next = blk.Next[:0]
	}
	root := t.roots[0]
	t.roots = t.children[root]
	delete(t.children, root)
	for _, r := range t.roots {
		t.parent[r] = nil
	}
	return t
}

// computes the immediate dominators using the classic Lengauer-Tarjan algorithm
//
// Lengauer T., Tarjan R. E. "A Fast Algorithm for Finding Dominators in a Flow
//   Graph." ACM TOPLAS. July 1979. https://doi.org/10.1145/357062.357071
//
func dominators(cfg *CFG, V int, root *Block, succ, pred func(*Block) []*Block) *DominatorTree {

	numbers := make(map[*Block]int, V)
	vertex := make([]*Block, V) // vertex[i] gives *Block whose number is i
	parent := make([]int, V)    // parent[i] parent for block i
	semi := make([]int, V)      // semi[i] semi-dominator for block i
	bucket := make([][]int, V)  // the set of vertices whose semi-dom is i
	dom := make([]int, V)       // the immediate dominator of i (at the end)

	ancestor := make([]int, V) // ancestor[i] the parent for block i in the DFS Tree forest
	label := make([]int, V)    // the label array for LINK/EVAL
	child := make([]int, V)    // the child array for the advanced LINK method
	size := make([]int, V)     // the size array for the advanced LINK method

	dfs := func(v *Block) {
		type entry struct {
			blk    *Block
			parent int
		}
		id := 0
		stack := make([]entry, 0, V)
		stack = append(stack, entry{root, 0})
		for len(stack) > 0 {
			var e entry
			stack, e = stack[:len(stack)-1], stack[len(stack)-1]
			if _, has := numbers[e.blk]; has {
				continue
			}
			numbers[e.blk] = id
			vertex[id] = e.blk
			parent[id] = e.parent
			semi[id] = id
			ancestor[id] = 0
			label[id] = id
			child[id] = 0
			size[id] = 1
			for _, kid := range succ(e.blk) {
				stack = append(stack, entry{kid, id})
			}
			id++
		}
	}

	var compress func(int)
	compress = func(v int) {
		if ancestor[ancestor[v]] != 0 {
			compress(ancestor[v])
			if semi[label[ancestor[v]]] < semi[label[v]] {
				label[v] = label[ancestor[v]]
			}
			ancestor[v] = ancestor[ancestor[v]]
		}
	}

	eval := func(v int) int {
		if ancestor[v] == 0 {
			return label[v]
		}
		compress(v)
		if semi[label[ancestor[v]]] >= semi[label[v]] {
			return label[v]
		} else {
			return label[ancestor[v]]
		}
	}

	link := func(v, w int) {
		s := w
		for semi[label[w]] < semi[label[child[s]]] {
			if size[s]+size[child[child[s]]] >= 2*size[child[s]] {
				ancestor[child[s]] = s
				child[s] = child[child[s]]
			} else {
				size[child[s]] = size[s]
				ancestor[s] = child[s]
				s = child[s]
			}
		}
		label[s] = label[w]
		size[v] = size[v] + size[w]
		if size[v] < 2*size[w] {
			s, child[v] = child[v], s
		}
		for s != 0 {
			ancestor[s] = v
			s = child[s]
		}
	}

	// dominator computation
	for i := range bucket {
		bucket[i] = make([]int, 0, 5)
	}
	dfs(root)
	for i := len(vertex) - 1; i > 0; i-- {
		blk := vertex[i]
		for _, p := range pred(blk) {
			v := numbers[p]
			u := eval(v)
			if semi[u] < semi[i] {
				semi[i] = semi[u]
			}
		}
		bucket[semi[i]] = append(bucket[semi[i]], i)
		link(parent[i], i)
		for _, v := range bucket[parent[i]] {
			u := eval(v)
			if semi[u] < semi[v] {
				dom[v] = u
			} else {
				dom[v] = parent[i]
			}
		}
		bucket[parent[i]] = bucket[parent[i]][:0] // clear the bucket
	}
	for i := 1; i < len(vertex); i++ {
		if dom[i] != semi[i] {
			dom[i] = dom[dom[i]]
		}
	}

	t := &DominatorTree{
		roots:    []*Block{root},
		parent:   make(map[*Block]*Block),
		children: make(map[*Block][]*Block),
		succ:     succ,
		pred:     pred,
	}
	for i, blk := range vertex {
		if i > 0 {
			t.parent[blk] = vertex[dom[i]]
			t.children[vertex[dom[i]]] = append(t.children[vertex[dom[i]]], blk)
		}
	}

	return t
}
