package main

func main() {
	println(fac(5))
}

func fac(x int) int {
	if x <= 1 {
		return 1
	}
	f := x * fac(x-1)
	return f
}
