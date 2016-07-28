package main

func main() {
	println(fac(5))
	println(unstructuredFac(5))
}

func fac(x int) int {
	if x <= 1 {
		return 1
	}
	return x * fac(x-1)
}

func unstructuredFac(x int) (y int) {
	defer func() {
		y = 7
	}()
	acc := 1
start:
	if x <= 1 {
		return acc
	}
	acc *= x
	x--
	goto start
}
