package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func main() {
	t := New()
	s := bufio.NewScanner(os.Stdin)
	for s.Scan() {
		line := s.Text()
		split := strings.Split(line, " ")
		cmd(t, split[0], split[1:])
	}
	if err := s.Err(); err != nil {
		panic(err)
	}
}

func cmd(t *Avl, cmd string, args []string) {
	// fmt.Fprintln(os.Stderr, cmd, args)
	switch cmd {
	case "put":
		k, err := strconv.Atoi(args[0])
		if err != nil {
			fmt.Println("err", err)
			return
		}
		v, err := strconv.Atoi(args[1])
		if err != nil {
			fmt.Println("err", err)
			return
		}
		t.Put(k, v)
		fmt.Println("ex put", k, v)
	case "has":
		k, err := strconv.Atoi(args[0])
		if err != nil {
			fmt.Println("err", err)
			return
		}
		fmt.Println("ex has", t.Has(k))
	case "get":
		k, err := strconv.Atoi(args[0])
		if err != nil {
			fmt.Println("err", err)
			return
		}
		v, has := t.Get(k)
		fmt.Println("ex get", v, has)
	case "rm":
		k, err := strconv.Atoi(args[0])
		if err != nil {
			fmt.Println("err", err)
			return
		}
		t.Remove(k)
		fmt.Println("ex rm", k)
	case "print":
		fmt.Println("ex print", t)
	case "verify":
		fmt.Println("ex verify", t.Verify())
	case "serialize":
		fmt.Println(t.root.Serialize())
	case "dotty":
		fmt.Println(t.root.Dotty())
	default:
		fmt.Printf("err not a command: %v\n", cmd)
	}
}

func verify(v interface {
	Verify() bool
}) {
	if !v.Verify() {
		panic(fmt.Errorf("Bad %v", v))
	}
}

type Avl struct {
	root *Node
}

func New() *Avl {
	return &Avl{}
}

func (a *Avl) Verify() bool {
	verify := a.root.Verify()
	i := 0
	p := 0
	for k, _, next := a.Iterate()(); next != nil; k, _, next = next() {
		if i > 0 {
			if p < k {
				// ok
			} else {
				fmt.Fprintln(os.Stderr, "prev key not less than current")
				fmt.Fprintln(os.Stderr, "prev:", p)
				fmt.Fprintln(os.Stderr, "cur:", k)
				// not ok, previous key should alwasy be less than current
				return false
			}
		}
		p = k
		i++
	}
	return verify
}

func (a *Avl) Iterate() Iterator {
	return a.root.Iterate()
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
	s := a.root.String()
	if s == "" {
		return "()"
	} else if s[0] == '(' {
		return s
	}
	return fmt.Sprintf("(%v _ _)", s)
}
