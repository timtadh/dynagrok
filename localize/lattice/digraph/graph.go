package digraph

type Digraph struct {
	V       Vertices
	E       Edges
	Adj     [][]int
	Kids    [][]int
	Parents [][]int
	Graphs  int
}
