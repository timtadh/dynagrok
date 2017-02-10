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
			switch i {
			case 0:
				break
			case 1:
				break loop
			default:
				continue loop
			}
			break
		}
		p, c = c, p+c
		continue loop
	}
	return c
}
