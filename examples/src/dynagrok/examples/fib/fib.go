package main

func main() {
	println(fibLoop(5))
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
