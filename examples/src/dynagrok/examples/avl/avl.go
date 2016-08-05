package main

import (
	"fmt"
)


func main() {
	t := New()
	t.Put(1,1)
	t.Put(2,2)
	fmt.Println(t)
	t.Put(3,3)
	t.Put(4,4)
	t.Put(5,5)
	t.Put(6,6)
	t.Put(7,7)
	t.Put(8,8)
	t.Put(9,9)
	t.Put(10,10)
	t.Put(11,11)
	fmt.Println(t)
	for i := 13; i > -2; i-- {
		fmt.Println(t.Get(i))
		t.Remove(i)
		fmt.Println(t)
	}
}


type Avl struct {
	root *Node
}

func New() *Avl {
	return &Avl{}
}

func (a *Avl) Verify() bool {
	return a.root.Verify()
}

func (a *Avl) Has(k int) bool {
	return a.root.Has(k)
}

func (a *Avl) Get(k int) (v int, has bool) {
	return a.root.Get(k)
}

func (a *Avl) Put(k, v int) {
	a.root = a.root.Put(k, v)
}

func (a *Avl) Remove(k int) {
	a.root = a.root.Remove(k)
}

func (a *Avl) String() string {
	return a.root.String()
}

