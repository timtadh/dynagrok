package locavore

import (
	"github.com/timtadh/dynagrok/dgruntime/dgtypes"
	"log"
)

func KMedoids(numClusters int, nodes []dgtypes.Clusterable) ([][]dgtypes.Clusterable, []dgtypes.Clusterable) {
	clusters := make([][]dgtypes.Clusterable, numClusters)
	medoids := make([]dgtypes.Clusterable, numClusters)
	if len(nodes) < numClusters {
		log.Panic("Failed to cluster with KMedoids: not enough nodes")
	}
	if numClusters == 0 {
		return clusters, medoids
	}

	// Initial step: add initial node to the medoids list
	for i := 0; i < numClusters; i++ {
		medoids[i] = nodes[i]
	}
	//		And remove the medoids from the nodes list
	nodes = nodes[numClusters:]

	// Assignment step:
	clusters = assignToMedoids(medoids, nodes)
	cost := totalCost(clusters, medoids)

	// Update step:
	for newcost := cost - 1; newcost < cost; {
		cost = newcost
		updateMedoids(clusters, medoids)
		newcost = totalCost(clusters, medoids)
	}

	return clusters, medoids
}

func totalCost(clusters [][]dgtypes.Clusterable, medoids []dgtypes.Clusterable) float64 {
	var cost float64 = 0
	for m, cluster := range clusters {
		cost += clusterCost(medoids[m], cluster)
	}
	return cost
}

func clusterCost(medoid dgtypes.Clusterable, nodes []dgtypes.Clusterable) float64 {
	cost := 0.0
	for _, node := range nodes {
		cost += medoid.Dissimilar(node)
	}
	return cost
}

func assignToMedoids(medoids []dgtypes.Clusterable, nodes []dgtypes.Clusterable) [][]dgtypes.Clusterable {
	clusters := make([][]dgtypes.Clusterable, len(medoids))
	for cluster := range clusters {
		clusters[cluster] = make([]dgtypes.Clusterable, 0)
	}

	for _, node := range nodes {
		var nearestMedoid int = 0
		var minDist float64 = node.Dissimilar(medoids[0])
		for j := range medoids {
			dist := node.Dissimilar(medoids[j])
			if dist < minDist {
				minDist = dist
				nearestMedoid = j
			}
		}
		clusters[nearestMedoid] = append(clusters[nearestMedoid], node)
	}

	return clusters
}

func updateMedoids(clusters [][]dgtypes.Clusterable, medoids []dgtypes.Clusterable) {
	for m := range medoids {
		for x := range clusters {
			for y := range clusters[x] {
				cost := clusterCost(medoids[m], clusters[m])
				swapCluster(m, x, y, clusters, medoids)
				newcost := clusterCost(medoids[m], clusters[m])
				if newcost >= cost {
					swapCluster(m, x, y, clusters, medoids)
				}
			}
		}
	}
}

func swapCluster(m int, x int, y int, clusters [][]dgtypes.Clusterable, medoids []dgtypes.Clusterable) {
	medoids[m], clusters[x][y] = clusters[x][y], medoids[m]
}

func removeByValue(node dgtypes.Clusterable, list []dgtypes.Clusterable) []dgtypes.Clusterable {
	index := indexOf(node, list)
	list[index] = list[len(list)-1]
	list = list[:len(list)-1]
	return list
}

func swapLists(medoid int, node int, medoids []dgtypes.Clusterable, nodes []dgtypes.Clusterable) {
	nodes[node], medoids[medoid] = medoids[medoid], nodes[node]
}

func indexOf(n dgtypes.Clusterable, o []dgtypes.Clusterable) int {
	for i := range o {
		if o[i] == n {
			return i
		}
	}
	return -1
}
