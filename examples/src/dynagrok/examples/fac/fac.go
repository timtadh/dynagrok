package main

func main() {
	println(fac(5))
}

func fac(x int) int {
	// id := func(x int) int {
	// 	return x
	// }
	// mul := func(a, b int) int {
	// 	return id(a) * id(b)
	// }
	if x <= 1 {
		return 1
	}
	// f := mul(x, fac(x-1))
	select {
	default:
	}
	f := x * fac(x-1)
	return f
}
