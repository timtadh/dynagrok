package main

import (
	"fmt"
	"os"
	"strings"
)

const maxSep = 2

type Node struct {
	Key, Value  int
	left, right *Node
	height      int
}

func max(a, b int) int {
	if a < b {
		return b
	}
	return a
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

func NewNode(k, v int) *Node {
	return &Node{
		Key:    k,
		Value:  v,
		height: 1,
	}
}

func (n *Node) Verify() bool {
	if n == nil {
		return true
	} else if n.left == nil && n.right == nil {
		return true
	}
	if n.left != nil && n.left.Key > n.Key {
		fmt.Fprintln(os.Stderr, "out of order (on left)")
		fmt.Fprintln(os.Stderr, "tree:", n)
		fmt.Fprintln(os.Stderr, "left:", n.left, n.left.Height())
		fmt.Fprintln(os.Stderr, "right:", n.right, n.right.Height())
		return false
	}
	if n.right != nil && n.right.Key < n.Key {
		fmt.Fprintln(os.Stderr, "out of order (on right)")
		fmt.Fprintln(os.Stderr, "tree:", n)
		fmt.Fprintln(os.Stderr, "left:", n.left, n.left.Height())
		fmt.Fprintln(os.Stderr, "right:", n.right, n.right.Height())
		return false
	}
	if n.height != max(n.right.Height(), n.left.Height())+1 {
		fmt.Fprintln(os.Stderr, "bad height")
		fmt.Fprintln(os.Stderr, "tree:", n)
		fmt.Fprintln(os.Stderr, "left:", n.left, n.left.Height())
		fmt.Fprintln(os.Stderr, "right:", n.right, n.right.Height())
		return false
	}
	if abs(n.right.Height()-n.left.Height()) >= maxSep {
		fmt.Fprintln(os.Stderr, "bad")
		fmt.Fprintln(os.Stderr, "tree:", n)
		fmt.Fprintln(os.Stderr, "left:", n.left, n.left.Height())
		fmt.Fprintln(os.Stderr, "right:", n.right, n.right.Height())
		return false
	}
	return n.right.Verify() && n.left.Verify()
}

func (n *Node) Height() int {
	if n == nil {
		return 0
	}
	return n.height
}

func (n *Node) Has(k int) bool {
	if n == nil {
		return false
	} else if k == n.Key {
		return true
	} else if k < n.Key {
		return n.left.Has(k)
	} else {
		return n.right.Has(k)
	}
}

type Iterator func() (k, v int, next Iterator)

func (n *Node) Iterate() (it Iterator) {
	pop := func(stack []*Node) (*Node, []*Node) {
		return stack[len(stack)-1], stack[:len(stack)-1]
	}
	stack := make([]*Node, 0, n.Height())
	cur := n
	it = func() (k, v int, _ Iterator) {
		if cur == nil && len(stack) <= 0 {
			return 0, 0, nil
		}
		for cur != nil {
			stack = append(stack, cur)
			cur = cur.left
		}
		cur, stack = pop(stack)
		k = cur.Key
		v = cur.Value
		cur = cur.right
		return k, v, it
	}
	return it
}

func (n *Node) Get(k int) (v int, has bool) {
	if n == nil {
		return 0, false
	} else if k == n.Key {
		return n.Value, true
	} else if k < n.Key {
		return n.left.Get(k)
	} else {
		return n.right.Get(k)
	}
}

func (n *Node) Put(k, v int) *Node {
	if n == nil {
		return NewNode(k, v)
	} else if k == n.Key {
		n.Value = v
		return n
	} else if k < n.Key {
		n.left = n.left.Put(k, v)
	} else {
		n.right = n.right.Put(k, v)
	}
	n.height = max(n.left.Height(), n.right.Height()) + 1
	return n.balance()
}

func (n *Node) Remove(k int) *Node {
	if n == nil {
		return nil
	} else if k == n.Key {
		// found remove this node
		return n.remove()
	} else if k < n.Key {
		// it would be on the left
		n.left = n.left.Remove(k)
	} else {
		n.right = n.right.Remove(k)
	}
	n.height = max(n.left.Height(), n.right.Height()) + 1
	return n.balance()
}

func (n *Node) remove() *Node {
	if n == nil {
		return nil
	} else if n.left == nil && n.right == nil {
		return nil
	} else if n.left == nil {
		return n.right
	} else if n.right == nil {
		return n.left
	} else {
		var r *Node
		if n.left.Height() < n.right.Height() {
			// promote from the right side
			r = n.right.leftmostDescendent()
		} else {
			// promote from the left side
			r = n.left.rightmostDescendent()
		}
		n = n.Remove(r.Key)
		r.left = n.left
		r.right = n.right
		r.height = max(r.left.Height(), r.right.Height()) + 1
		return r.balance()
	}
}

func (n *Node) balance() *Node {
	if n == nil {
		return nil
	}
	if abs(n.left.Height()-n.right.Height()) < maxSep {
		return n
	} else if n.left.Height() < n.right.Height() {
		return n.rotateLeft()
	} else {
		return n.rotateRight()
	}
}

func (n *Node) rotateRight() *Node {
	if n == nil {
		return nil
	} else if n.left == nil {
		return n
	}
	r := n.left.rightmostDescendent()
	n.left = n.left.Remove(r.Key)
	return n.doRotate(r)
}

func (n *Node) rotateLeft() *Node {
	if n == nil {
		return nil
	} else if n.right == nil {
		return n
	}
	r := n.right.leftmostDescendent()
	n.right = n.right.Remove(r.Key)
	return n.doRotate(r)
}

func (n *Node) doRotate(r *Node) *Node {
	r.left = n.left
	r.right = n.right
	return r.Put(n.Key, n.Value)
}

func (n *Node) rightmostDescendent() *Node {
	if n == nil {
		return nil
	} else if n.right == nil {
		return n
	} else {
		return n.right.rightmostDescendent()
	}
}

func (n *Node) leftmostDescendent() *Node {
	if n == nil {
		return nil
	} else if n.left == nil {
		return n
	} else {
		return n.left.leftmostDescendent()
	}
}

func (n *Node) String() string {
	if n == nil {
		return ""
	}
	left := n.left.String()
	right := n.right.String()
	if left == "" && right == "" {
		return fmt.Sprintf("%v", n.Key)
	} else if left == "" {
		return fmt.Sprintf("(%v _ %v)", n.Key, right)
	} else if right == "" {
		return fmt.Sprintf("(%v %v _)", n.Key, left)
	} else {
		return fmt.Sprintf("(%v %v %v)", n.Key, left, right)
	}
}

func (n *Node) Serialize() string {
	if n == nil {
		return ""
	}
	type item struct {
		n     *Node
		depth int
	}
	pop := func(stack []*item) (*item, []*item) {
		return stack[len(stack)-1], stack[:len(stack)-1]
	}
	list := make([]string, 0, 10)
	stack := make([]*item, 0, 10)
	stack = append(stack, &item{n, 0})
	for len(stack) > 0 {
		var c *item
		c, stack = pop(stack)
		kids := ""
		for i := 0; i < c.depth*4; i++ {
			kids += " "
		}
		if c.n.left != nil {
			kids += "l"
		} else {
			kids += "-"
		}
		if c.n.right != nil {
			kids += "r"
		} else {
			kids += "-"
		}
		if c.n.right != nil {
			stack = append(stack, &item{c.n.right, c.depth + 1})
		}
		if c.n.left != nil {
			stack = append(stack, &item{c.n.left, c.depth + 1})
		}
		list = append(list, fmt.Sprintf(
			"%v:%v:%v:%v",
			kids,
			c.n.Key,
			c.n.Value,
			c.n.Height(),
		))
	}
	return strings.Join(list, "\n")
}

func (n *Node) Dotty() string {
	if n == nil {
		return "digraph AVL {}"
	}
	type item struct {
		n      *Node
		depth  int
		parent int
		side   string
	}
	pop := func(stack []*item) (*item, []*item) {
		return stack[len(stack)-1], stack[:len(stack)-1]
	}
	lines := make([]string, 0, 10)
	lines = append(lines, "digraph {")
	lines = append(lines, "margin=0")
	lines = append(lines, "node [shape=rect, margin=.01]")

	levels := make(map[int][]int)

	nodes := make(map[int][]string)
	edges := make([]string, 0, 10)

	stack := make([]*item, 0, 10)
	stack = append(stack, &item{n, 0, -1, ""})
	id := 0
	for len(stack) > 0 {
		var i *item
		i, stack = pop(stack)
		nid := id
		id++

		levels[i.depth] = append(levels[i.depth], nid)

		nodes[i.depth] = append(nodes[i.depth], fmt.Sprintf("%v [label=%v];", nid, i.n.Key))
		if i.parent >= 0 {
			edges = append(edges, fmt.Sprintf("%v -> %v;", i.parent, nid))
		}
		if i.n.left == nil && i.n.right == nil {
			// skip
		} else if i.n.left == nil {
			kid := id
			id++
			nodes[i.depth+1] = append(nodes[i.depth+1], fmt.Sprintf("%v [label=\"\", width=.1, height=.1, margin=0];", kid))
			edges = append(edges, fmt.Sprintf("%v -> %v;", nid, kid))
			levels[i.depth+1] = append(levels[i.depth+1], kid)
		} else if i.n.right == nil {
			kid := id
			id++
			nodes[i.depth+1] = append(nodes[i.depth+1], fmt.Sprintf("%v [label=\"\", width=.1, height=.1, margin=0];", kid))
			edges = append(edges, fmt.Sprintf("%v -> %v;", nid, kid))
			levels[i.depth+1] = append(levels[i.depth+1], kid)
		}
		if i.n.right != nil {
			stack = append(stack, &item{i.n.right, i.depth + 1, nid, "right"})
		}
		if i.n.left != nil {
			stack = append(stack, &item{i.n.left, i.depth + 1, nid, "left"})
		}
	}

	for depth, level := range nodes {
		lines = append(lines, "{")
		lines = append(lines, "rank=same;")
		lines = append(lines, level...)
		if len(level) <= 1 {
			lines = append(lines, "}")
			continue
		}
		ids := make([]string, 0, len(levels))
		for _, nid := range levels[depth] {
			ids = append(ids, fmt.Sprintf("%v", nid))
		}
		lines = append(lines, fmt.Sprintf("%v [style=invis];", strings.Join(ids, " -> ")))
		lines = append(lines, "}")
	}

	lines = append(lines, edges...)

	lines = append(lines, "}")
	return strings.Join(lines, "\n")
}
