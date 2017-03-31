package locavore

import (
	"github.com/timtadh/dynagrok/dgruntime/dgtypes"
	"log"
)

func KMedoids(numClusters int, nodes []dgtypes.Clusterable) map[dgtypes.Clusterable][]dgtypes.Clusterable {
	clusters := make(map[dgtypes.Clusterable][]dgtypes.Clusterable)
	medoids := make([]dgtypes.Clusterable, numClusters)
	if len(nodes) < numClusters {
		log.Panic("Failed to cluster with KMedoids: not enough nodes")
	}
	if numClusters == 0 {
		return clusters
	}

	// Initial step: add initial node to the medoids list
	for i := 0; i < numClusters; i++ {
		medoids[i] = nodes[i]
	}
	//		And remove the medoids from the nodes list
	nodes = nodes[numClusters:]

	// Assignment step:
	clusters = assignToMedoids(medoids, nodes)
	cost := totalCost(clusters)

	// Update step:
	for newcost := cost - 1; newcost < cost; {
		cost = newcost
		updateMedoids(clusters, medoids, nodes)
		newcost = totalCost(clusters)
	}

	return clusters
}

func totalCost(clusters map[dgtypes.Clusterable][]dgtypes.Clusterable) float64 {
	var cost float64 = 0
	for m, cluster := range clusters {
		cost += clusterCost(m, cluster)
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

func assignToMedoids(medoids []dgtypes.Clusterable, nodes []dgtypes.Clusterable) map[dgtypes.Clusterable][]dgtypes.Clusterable {
	clusters := make(map[dgtypes.Clusterable][]dgtypes.Clusterable)

	for _, node := range nodes {
		var nearestMedoid dgtypes.Clusterable = medoids[0]
		var minDist float64 = node.Dissimilar(medoids[0])
		for j := range medoids {
			dist := node.Dissimilar(medoids[j])
			if dist < minDist {
				minDist = dist
				nearestMedoid = medoids[j]
			}
		}
		if _, ok := clusters[nearestMedoid]; ok {
			clusters[nearestMedoid] = append(clusters[nearestMedoid], node)
		} else {
			clusters[nearestMedoid] = []dgtypes.Clusterable{node}
		}
	}

	return clusters
}

func updateMedoids(clusters map[dgtypes.Clusterable][]dgtypes.Clusterable, medoids []dgtypes.Clusterable, nodes []dgtypes.Clusterable) {
	for m, medoid := range medoids {
		for n, node := range nodes {
			cost := clusterCost(medoid, clusters[medoid])
			swapCluster(medoid, node, clusters)
			newcost := clusterCost(node, clusters[node])
			if newcost >= cost {
				swapCluster(node, medoid, clusters)
			} else {
				swapLists(m, n, medoids, nodes)
			}
		}
	}
}

func swapCluster(medoid dgtypes.Clusterable, node dgtypes.Clusterable, clusters map[dgtypes.Clusterable][]dgtypes.Clusterable) {
	medGroup := clusters[medoid]

	// from the list of nodes: add the medoid, remove the node
	medGroup = append(medGroup, medoid)
	medGroup = removeByValue(node, medGroup)
	// From the map of clusters, remove the medoid and add the node entry
	delete(clusters, medoid)
	clusters[node] = medGroup
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
