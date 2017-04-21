package locavore

import (
	"math"
	"testing"

	"github.com/timtadh/data-structures/test"
)

type Vec struct {
	x int
	y int
}

func (v Vec) Dissimilar(o Clusterable) float64 {
	if v2, ok := o.(Vec); ok {
		return math.Abs(float64(v.x-v2.x)) + math.Abs(float64(v.y-v2.y))
	}
	return math.Inf(1)
}

var TestVecs []Clusterable

func initialize() {
	TestVecs = []Clusterable{Vec{0, 10},
		Vec{10, 0},
		Vec{10, 10},
		Vec{10, 1},
		Vec{10, 11},
		Vec{10, 12}}
}

func TestAssignToMedoids(x *testing.T) {
	t := (*test.T)(x)
	initialize()
	medoids := make([]Clusterable, 3)
	medoids[0] = TestVecs[0]
	medoids[1] = TestVecs[1]
	medoids[2] = TestVecs[2]
	TestVecs = TestVecs[3:]

	clusters := assignToMedoids(medoids, TestVecs)
	nodes, ok := clusters[medoids[2]]
	t.Assert(ok && len(nodes) == 2, "assignToMedoids is not working as expected: %v", clusters)
}

func TestSwap(x *testing.T) {
	t := (*test.T)(x)
	initialize()
	medoid := TestVecs[2]
	nodes := TestVecs[4:]

	clusters := make(map[Clusterable][]Clusterable)
	clusters[medoid] = nodes
	cost := clusterCost(medoid, nodes)
	t.Assert(cost == 3, "cost of %f  unexpected: %v", cost, clusters)
	swapCluster(medoid, nodes[0], clusters)

	medoid, nodes[0] = nodes[0], medoid

	cost = clusterCost(medoid, nodes)
	t.Assert(cost == 2, "Lower cost of %f unexpected: %v", cost, clusters)
}
