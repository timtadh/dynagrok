package main


func main() {
	for i := 10; i >= -2; i-- {
		println(fib(i))
	}
	println("done")
}

func fib(x int) int {
	if x < 0 {
		return 0
	}
	p, c := 0, 1
	for i := 0; i < x; i++ {
		n := p + c
		p, c = c, n
	}
	return c
}

// func main() {
// 	println(fibLoop(5))
// }
// 
// func fibLoop(x int) int {
// 	p := 0
// 	c := 1
// loop:
// 	for i := 0; ; i++ {
// 		if i >= x {
// 			switch i {
// 			case 0:
// 				break
// 			case 1:
// 				break loop
// 			default:
// 				continue loop
// 			}
// 			break
// 		}
// 		p, c = c, p+c
// 		continue loop
// 	}
// 	return c
// }
