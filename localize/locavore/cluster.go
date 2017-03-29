package locavore

import (
	"log"
)

type Clusterable interface {
	Dissimilar(Clusterable) float64
}

func KMedoids(numClusters int, nodes []Clusterable) map[Clusterable][]Clusterable {
	clusters := make(map[Clusterable][]Clusterable)
	medoids := make([]Clusterable, numClusters)
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

func totalCost(clusters map[Clusterable][]Clusterable) float64 {
	var cost float64 = 0
	for m, cluster := range clusters {
		cost += clusterCost(m, cluster)
	}
	return cost
}

func clusterCost(medoid Clusterable, nodes []Clusterable) float64 {
	cost := 0.0
	for _, node := range nodes {
		cost += medoid.Dissimilar(node)
	}
	return cost
}

func assignToMedoids(medoids []Clusterable, nodes []Clusterable) map[Clusterable][]Clusterable {
	clusters := make(map[Clusterable][]Clusterable)

	for _, node := range nodes {
		var nearestMedoid Clusterable = medoids[0]
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
			clusters[nearestMedoid] = []Clusterable{node}
		}
	}

	return clusters
}

func updateMedoids(clusters map[Clusterable][]Clusterable, medoids []Clusterable, nodes []Clusterable) {
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

func swapCluster(medoid Clusterable, node Clusterable, clusters map[Clusterable][]Clusterable) {
	medGroup := clusters[medoid]

	// from the list of nodes: add the medoid, remove the node
	medGroup = append(medGroup, medoid)
	medGroup = removeByValue(node, medGroup)
	// From the map of clusters, remove the medoid and add the node entry
	delete(clusters, medoid)
	clusters[node] = medGroup
}

func removeByValue(node Clusterable, list []Clusterable) []Clusterable {
	index := indexOf(node, list)
	list[index] = list[len(list)-1]
	list = list[:len(list)-1]
	return list
}

func swapLists(medoid int, node int, medoids []Clusterable, nodes []Clusterable) {
	nodes[node], medoids[medoid] = medoids[medoid], nodes[node]
}

func indexOf(n Clusterable, o []Clusterable) int {
	for i := range o {
		if o[i] == n {
			return i
		}
	}
	return -1
}
