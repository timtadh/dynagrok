package main

func main() {
	println(fac(5))
	println(loopFac(5))
	println(unstructuredFac(5))
}

func fac(x int) int {
	if x <= 1 {
		return 1
	}
	return x * fac(x-1)
}

func loopFac(x int) int {
	f := 1
	for i := 1; i <= x; i++ {
		f = f * i
	}
	return f
}

func loopFacBreak(x int) int {
	f := 1
	for i := 1; ; i++ {
		if i < 0 {
			i--
			continue
		}
		if i > x {
			break
		}
		f = f * i
	}
	return f
}

func unstructuredFac(x int) (y int) {
	acc := 1
start:
	if x <= 1 {
		return acc
	}
	acc *= x
	x--
	goto start
}
