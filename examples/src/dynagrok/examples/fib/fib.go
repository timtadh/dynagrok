package main

func init() {
	wizard_1()
}

func init() {
	wizard_2()
}

func wizard_1() {
	println("1 wizard")
}

func wizard_2() {
	println("2 wizard")
}

func main() {
	c := make(chan bool)
	x := func()int{return 1}
	go func() {
		println(((*wacky)(&x)).fib(5))
		c<-true
	}()
	println(fibLoop(5))
	<-c
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
