package main

import (
	"fmt"
)

const maxSep = 2

type Node struct {
	Key, Value int
	left, right *Node
	height int
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
		Key: k,
		Value: v,
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
		return false
	}
	if n.right != nil && n.right.Key < n.Key {
		return false
	}
	if n.height != max(n.right.Height(), n.left.Height()) + 1 {
		return false
	}
	if abs(n.right.Height() - n.left.Height()) >= maxSep {
		fmt.Println("bad")
		fmt.Println("tree:", n)
		fmt.Println("left:", n.left, n.left.Height())
		fmt.Println("right:", n.right, n.right.Height())
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
	} else if (k < n.Key) {
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
		n.Value = k
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
			n.right = n.right.popNode(r)
		} else {
			// promote from the left side
			r = n.left.rightmostDescendent()
			n.left = n.left.popNode(r)
		}
		r.left = n.left;
		r.right = n.right;
		r.height = max(r.left.Height(), r.right.Height()) + 1
		return r.balance()
	}
}

func (n *Node) balance() *Node {
	if n == nil {
		return nil
	}
	if abs(n.left.Height() - n.right.Height()) < maxSep {
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
	n = n.popNode(r)
	r.left = n.left
	r.right = n.right
	n.left = nil
	n.right = nil
	n.height = 1
	return r.pushNode(n)
}

func (n *Node) rotateLeft() *Node {
	if n == nil {
		return nil
	} else if n.right == nil {
		return n
	}
	r := n.right.leftmostDescendent()
	n = n.popNode(r)
	r.left = n.left
	r.right = n.right
	n.left = nil
	n.right = nil
	n.height = 1
	return r.pushNode(n)
}

func (n *Node) pushNode(x *Node) *Node {
	if x == nil {
		return n
	} else if n == nil {
		return x
	}
	if x.Key == n.Key {
		panic("pushing a node to the same node")
	} else if x.Key < n.Key {
		n.left = n.left.pushNode(x)
	} else {
		n.right = n.right.pushNode(x)
	}
	n.height = max(n.left.Height(), n.right.Height()) + 1
	return n.balance()
}

func (n *Node) popNode(x *Node) *Node {
	if n == nil {
		return nil
	} else if x == nil {
		return n
	} else if x.left != nil && x.right != nil {
		panic("x may have left or right but not both subtrees")
	}
	if n == x {
		var k *Node = nil
		if n.left != nil {
			k = n.left
		} else if n.right != nil {
			k = n.right
		}
		n.left = nil
		n.right = nil
		return k.balance()
	} else if x.Key == n.Key {
		panic("popping a node with the same key")
	} else if x.Key < n.Key {
		n.left = n.left.popNode(x)
	} else {
		n.right = n.right.popNode(x)
	}
	n.height = max(n.left.Height(), n.right.Height()) + 1
	return n.balance()
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
	left := n.left.String();
	right := n.right.String();
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

