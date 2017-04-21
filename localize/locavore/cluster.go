package locavore

import (
	"log"

	"github.com/timtadh/dynagrok/dgruntime/dgtypes"
)

func KMedoidsFunc(numClusters int, nodes []dgtypes.Clusterable, f func(dgtypes.Clusterable, dgtypes.Clusterable) float64) ([][]dgtypes.Clusterable, []dgtypes.Clusterable) {
	clusters := make([][]dgtypes.Clusterable, 0, numClusters)
	medoids := make([]dgtypes.Clusterable, 0, numClusters)
	if numClusters == 0 || len(nodes) < 3 {
		log.Printf("No profiles to cluster")
		return clusters, medoids
	}
	if len(nodes) < numClusters {
		numClusters = len(nodes) / 2
		log.Printf("Setting numBins temporarily to %v", numClusters)
	}

	// Initial step: add initial node to the medoids list
	for i := 0; i < numClusters; i++ {
		medoids = append(medoids, nodes[i])
	}
	//		And remove the medoids from the nodes list
	nodes = nodes[numClusters:]

	// Assignment step:
	clusters = assignToMedoids(medoids, nodes, f)
	cost := totalCost(clusters, medoids, f)

	// Update step:
	first := true
	for newcost := cost - 1; newcost < cost; {
		//	fmt.Printf("---\n")
		//	fmt.Printf("Clusters: \n")
		//	for m := range medoids {
		//		fmt.Printf("\tMedoid %v:\t Cluster:%v\n\n", medoids[m], clusters[m])
		//	}
		//	fmt.Printf("---\n")
		if first {
			newcost = cost
			first = false
		}
		cost = newcost
		updateMedoids(clusters, medoids, f)
		clusters = assignToMedoids(medoids, nodes, f)
		newcost = totalCost(clusters, medoids, f)
	}

	return clusters, medoids
}

func KMedoids(numClusters int, nodes []dgtypes.Clusterable) ([][]dgtypes.Clusterable, []dgtypes.Clusterable) {
	return KMedoidsFunc(numClusters, nodes, func(i, j dgtypes.Clusterable) float64 { return i.Dissimilar(j) })
}

func totalCost(clusters [][]dgtypes.Clusterable, medoids []dgtypes.Clusterable, f func(dgtypes.Clusterable, dgtypes.Clusterable) float64) float64 {
	var cost float64 = 0
	for m, cluster := range clusters {
		cost += clusterCost(medoids[m], cluster, f)
	}
	return cost
}

func clusterCost(medoid dgtypes.Clusterable, nodes []dgtypes.Clusterable, disFunc func(dgtypes.Clusterable, dgtypes.Clusterable) float64) float64 {
	cost := 0.0
	for _, node := range nodes {
		cost += disFunc(medoid, node)
	}
	return cost
}

func assignToMedoids(medoids []dgtypes.Clusterable, nodes []dgtypes.Clusterable, disFunc func(dgtypes.Clusterable, dgtypes.Clusterable) float64) [][]dgtypes.Clusterable {
	//log.Printf("\nAssigning to medoids...")
	//log.Printf("\nMedoids: [%v]", medoids)
	clusters := make([][]dgtypes.Clusterable, len(medoids))
	for cluster := range clusters {
		clusters[cluster] = make([]dgtypes.Clusterable, 0)
	}
	//log.Printf("Created %v clusters: %v", len(clusters), clusters)

	for _, node := range nodes {
		var nearestMedoid int = 0
		var minDist float64 = disFunc(node, medoids[0])
		for j := range medoids {
			dist := disFunc(node, medoids[j])
			if dist < minDist {
				minDist = dist
				nearestMedoid = j
				//	log.Printf("\t\t\tNot Equal: %v", dist)
				//} else if dist == minDist {
				//	log.Printf("\t\t\tEqual: %v", dist)
				//} else {
				//	log.Printf("\t\t\tNot Equal: %v", dist)
			}
		}
		//log.Printf("\n\t%v\n is the nearest medoid to \n\t%v", medoids[nearestMedoid], node)
		clusters[nearestMedoid] = append(clusters[nearestMedoid], node)
	}

	return clusters
}

func updateMedoids(clusters [][]dgtypes.Clusterable, medoids []dgtypes.Clusterable, f func(dgtypes.Clusterable, dgtypes.Clusterable) float64) {
	for m := range medoids {
		for x := range clusters {
			for y := range clusters[x] {
				cost := clusterCost(medoids[m], clusters[m], f)
				swapCluster(m, x, y, clusters, medoids)
				newcost := clusterCost(medoids[m], clusters[m], f)
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
