package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Node struct {
	Val  int
	Next *Node
}

func main() {
	var head *Node = &Node{}
	scanner := bufio.NewScanner(os.Stdin)
OuterLoop:
	for scanner.Scan() {
		cmd := strings.Split(scanner.Text(), " ")
		if len(cmd) == 0 {
			continue
		}
		switch cmd[0] {
		case "exit":
			break OuterLoop
		case "put":
			fmt.Printf("Put %d\n", Put(cmd[1], &head))
		case "pop":
			Pop(&head)
		case "show":
			Show(head)
		case "has":
			fmt.Printf("%t\n", Has(cmd[1], head))
		default:
			fmt.Printf("Didn't recognize %s\n", cmd[0])
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
	}
}

func Put(v string, list **Node) int {
	val, err := strconv.Atoi(v)
	if err != nil {
		return 0
	}
	return put(val, list)
}

func put(v int, head **Node) int {
	if **head == (Node{}) {
		*head = &Node{Val: v, Next: nil}
	} else {
		*head = &Node{Val: v, Next: *head}
	}
	return v
}

func Pop(head **Node) int {
	defer func() {
		*head = (*head).Next
	}()
	return (*head).Val
}

func Show(head *Node) {
	cur := head
	if cur != nil {
		fmt.Printf("%d", cur.Val)
		cur = cur.Next
	}
	for cur != nil {
		fmt.Printf(", %d", cur.Val)
		cur = cur.Next
	}
	fmt.Println("")
}

func Has(v string, head *Node) bool {
	val, err := strconv.Atoi(v)
	if err != nil {
		return false
	}
	return has(val, head)
}

func has(v int, head *Node) bool {
	ret := false
	cur := head
	for cur.Next != nil { //fault
		if cur.Val == v {
			ret = true
			break
		}
		cur = cur.Next
	}
	return ret
}
