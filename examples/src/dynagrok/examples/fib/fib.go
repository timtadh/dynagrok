package main

func main() {
	x := func()int{return 1}
	println(((*wacky)(&x)).fib(5))
	println(fibLoop(5))
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

func fibLoop(x int) int {
	p := 0
	c := 1
loop:
	for i := 0; ; i++ {
		if i >= x {
			break loop
		}
		p, c = c, p+c
		continue loop
	}
	return c
}
