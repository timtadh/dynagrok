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
	foo := make(chan int, 1)
	foo <- 1
	select {
	case i := <-foo:
		println(i)
	case foo <- x:
		i := <-foo
		println(i)
	default:
		println("wiz")
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

func rangeEx(x int) {
	l := make([]int, 10)
	for i, c := range l {
		print(i)
		print(":")
		print(c)
		if i+1 < len(l) {
			print(", ")
		}
	}
	println()
}

func typeSwitch(x interface{}) {
	switch x.(type) {
	case uint, float64:
		println("is uint or float64")
	case int:
		println("is int")
	}
}

func switchStmt(x int) {
	switch x {
	default:
		println("default")
		fallthrough
	case 1:
		println(1)
		if false {
			break
		}
		fallthrough
	case 2:
		println(2)
		fallthrough
	case 3:
		println(3)
	}
	println("done")
}

func ifElseIf(x int) {
	if x == 0 {
		println("== 0")
	} else if x == 1 {
		println("== 1")
	} else if x == 2 {
		println("== 2")
	} else if x == 3 {
		println("== 3")
	} else {
		println("bigger than 3 or less than 0")
	}
	println("done")
}
