package main

func main() {
	x := func()int{return 1}
	((*wacky)(&x)).fib(5)
}

type wacky func() int

func (w *wacky) fib(x int) int {
	id := func(x int) int {
		return x
	}
	add := func(a, b int) int {
		z := id(a) + id(b)
		return z
	}
	if x <= 1 {
		return 1
	}
	f := add(w.fib(x-1), w.fib(x-2))
	return f
}
