package main

func panicWhen(cond bool) {
	if cond {
		panic("The right thing.")
	} else {
		panic("The wrong thing.")
	}
}

func main() {
	e := (*string)(nil)
	panicWhen(e == e)
	// Should never reach this line.
	panicWhen(*e == *e)
}

