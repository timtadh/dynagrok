package main

import (
	"os"
)

func main() {
	println(fac(5))
	x := 0
	a: {
		print("wizard\n")
		if x > 0 {
			goto c
		}
		goto b
	}
	b: {
		print("wooze\n")
		x++
		goto a
	}
	c: {
		if true {
			os.Exit(5)
		}
	}
}

func fac(x int) int {
	if x <= 1 {
		return 1
	}
	return x * fac(x-1)
}

