package runtime

// Get the current goroutine's id
func GoID() int64 {
	g := getg()
	return g.goid
}

