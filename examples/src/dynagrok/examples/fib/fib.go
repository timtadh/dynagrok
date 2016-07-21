package main

func main() {
	fib(5)
}

func fib(x int) int {
	if x <= 1 {
		return 1
	}
	f := fib(x-1) + fib(x-2)
	return f
}
