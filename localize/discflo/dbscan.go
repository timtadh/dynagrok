package discflo

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
)

import (
	"github.com/timtadh/data-structures/errors"
	"github.com/timtadh/data-structures/exc"
	"github.com/timtadh/data-structures/set"
	"github.com/timtadh/data-structures/types"
)

import (
	"github.com/timtadh/dynagrok/localize/lattice"
	"github.com/timtadh/dynagrok/localize/mine"
)

type Clusters []*Cluster

type Cluster struct {
	Score float64
	Nodes []*mine.SearchNode
}

func (c *Cluster) String() string {
	return fmt.Sprintf("cluster %v %v %v", c.Score, len(c.Nodes), c.Nodes[0])
}

type clusterNode struct {
	n      *mine.SearchNode
	name   string
	labels types.Set
}

func newClusterNode(n *mine.SearchNode) (*clusterNode, error) {
	labels, err := labelset(n.Node)
	if err != nil {
		return nil, err
	}
	cn := &clusterNode{n, n.String(), labels}
	return cn, nil
}

func labelSimilarity(a, b *clusterNode) float64 {
	return jaccardSetSimilarity(a.labels, b.labels)
}

func jaccardSetSimilarity(a, b types.Set) float64 {
	i, err := a.Intersect(b)
	exc.ThrowOnError(err)
	inter := float64(i.Size())
	return 1.0 - (inter / (float64(a.Size()) + float64(b.Size()) - inter))
}

type cluster []*clusterNode

func correlation(clusters []cluster, metric func(a, b *clusterNode) float64) float64 {
	var totalDist float64
	var totalIncidence float64
	var totalItems float64
	for x, X := range clusters {
		for i := 0; i < len(X); i++ {
			for y := x; y < len(clusters); y++ {
				Y := clusters[y]
				for j := i + 1; j < len(Y); j++ {
					totalDist += metric(X[i], Y[j])
					if x == y {
						totalIncidence++
					}
					totalItems++
				}
			}
		}
	}
	meanDist := totalDist / totalItems
	meanIncidence := totalIncidence / totalItems
	var sumOfSqDist float64
	var sumOfSqIncidence float64
	var sumOfProduct float64
	for x, X := range clusters {
		for i := 0; i < len(X); i++ {
			for y := x; y < len(clusters); y++ {
				Y := clusters[y]
				for j := i + 1; j < len(Y); j++ {
					dist := metric(X[i], Y[j])
					var incidence float64
					if x == y {
						incidence = 1.0
					}
					distDiff := (dist - meanDist)
					incidenceDiff := (incidence - meanIncidence)
					sumOfSqDist += distDiff * distDiff
					sumOfSqIncidence += incidenceDiff * incidenceDiff
					sumOfProduct += distDiff * incidenceDiff
				}
			}
		}
	}
	return sumOfProduct / (sumOfSqDist * sumOfSqIncidence)
}

func intradist(clusters []cluster, metric func(a, b *clusterNode) float64) float64 {
	var totalDist float64
	for _, X := range clusters {
		var dist float64
		for i := 0; i < len(X); i++ {
			for j := i + 1; j < len(X); j++ {
				dist += metric(X[i], X[j])
			}
		}
		if len(X) > 0 {
			totalDist += dist / float64(len(X))
		}
	}
	return totalDist
}

func interdist(clusters []cluster, metric func(a, b *clusterNode) float64) float64 {
	var totalDist float64
	for x, X := range clusters {
		var dist float64
		for i := 0; i < len(X); i++ {
			for y := x; y < len(clusters); y++ {
				Y := clusters[y]
				var to float64
				for j := i + 1; j < len(Y); j++ {
					if y == x {
						continue
					}
					for j := 0; j < len(Y); j++ {
						to += metric(X[i], Y[j])
					}
				}
				if len(Y) > 0 {
					dist += to / float64(len(Y))
				}
			}
		}
		if len(X) > 0 {
			totalDist += dist / float64(len(X))
		}
	}
	return totalDist
}

func noNan(x float64) interface{} {
	if math.IsNaN(x) {
		return "nan"
	}
	return x
}

type DbScan struct {
	clusters []cluster
	items    int
	epsilon  float64
	seen     map[string]bool
}

func NewDbScan(epsilon float64) *DbScan {
	r := &DbScan{
		epsilon: epsilon,
		seen:    make(map[string]bool),
	}
	return r
}

func (r *DbScan) Count() int {
	return r.items
}

func (r *DbScan) Add(n *mine.SearchNode) error {
	cn, err := newClusterNode(n)
	if err != nil {
		return err
	}
	r.items++
	r.clusters = add(r.clusters, cn, r.epsilon, labelSimilarity)
	return nil
}

func (r *DbScan) Clusters() []*Cluster {
	clstrs := make([]*Cluster, 0, len(r.clusters))
	for _, cluster := range r.clusters {
		clstr := make([]*mine.SearchNode, 0, len(cluster))
		sum := 0.0
		for _, cn := range cluster {
			clstr = append(clstr, cn.n)
			sum += cn.n.Score
		}
		lc := float64(len(clstr))
		score := (sum / lc) * math.Sqrt(lc/(lc+1))
		clstrs = append(clstrs, &Cluster{
			Score: score,
			Nodes: clstr,
		})
	}
	return clstrs
}

func (r *DbScan) WriteClusters(dir string) error {
	f, err := os.Create(filepath.Join(dir, "clusters"))
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "")
	random := make([]cluster, rand.Intn(int(float64(len(r.clusters))*1.2))+2)
	for i, cluster := range r.clusters {
		for _, cn := range cluster {
			x := map[string]interface{}{
				"cluster": i,
				"name":    cn.name,
				"labels":  fmt.Sprintf("%v", cn.labels),
			}
			err := enc.Encode(x)
			if err != nil {
				return err
			}
			n := rand.Intn(len(random))
			random[n] = append(random[n], cn)
		}
	}
	return r.metrics(filepath.Join(dir, "metrics"), random)
}

func (r *DbScan) metrics(filePath string, random []cluster) error {
	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "")
	if len(r.clusters) >= r.items || len(r.clusters) <= 1 {
		x := map[string]interface{}{
			"items":    r.items,
			"clusters": len(r.clusters),
		}
		return enc.Encode(x)
	}
	intraLabel := intradist(r.clusters, labelSimilarity)
	interLabel := interdist(r.clusters, labelSimilarity)
	intraLabelRand := intradist(random, labelSimilarity)
	interLabelRand := interdist(random, labelSimilarity)
	stderr := func(a, b float64) float64 {
		return math.Sqrt((a - b) * (a - b))
	}
	x := map[string]interface{}{
		"items": r.items,
		"cluster-metrics": map[string]interface{}{
			"count":                len(r.clusters),
			"label-correlation":    noNan(correlation(r.clusters, labelSimilarity)),
			"label-intra-distance": noNan(intraLabel),
			"label-inter-distance": noNan(interLabel),
			"label-distance-ratio": noNan(intraLabel / interLabel),
		},
		"random-metrics": map[string]interface{}{
			"count":                len(random),
			"label-correlation":    noNan(correlation(random, labelSimilarity)),
			"label-intra-distance": noNan(intraLabelRand),
			"label-inter-distance": noNan(interLabelRand),
			"label-distance-ratio": noNan(intraLabelRand / interLabelRand),
		},
		"standard-error": map[string]interface{}{
			"count":                noNan(stderr(float64(len(r.clusters)), float64(len(random)))),
			"label-correlation":    noNan(stderr(correlation(r.clusters, labelSimilarity), correlation(random, labelSimilarity))),
			"label-intra-distance": noNan(stderr(intraLabel, intraLabelRand)),
			"label-inter-distance": noNan(stderr(interLabel, interLabelRand)),
			"label-distance-ratio": noNan(stderr(intraLabel/interLabel, intraLabelRand/interLabelRand)),
		},
	}
	err = enc.Encode(x)
	if err != nil && strings.Contains(err.Error(), "NaN") {
		x := map[string]interface{}{
			"items":    r.items,
			"clusters": len(r.clusters),
		}
		return enc.Encode(x)
	} else if err != nil {
		return err
	}
	return nil
}

func add(clusters []cluster, cn *clusterNode, epsilon float64, sim func(a, b *clusterNode) float64) []cluster {
	near := set.NewSortedSet(len(clusters))
	min_near := -1
	min_sim := -1.0
	var min_item *clusterNode = nil
	for i := len(clusters) - 1; i >= 0; i-- {
		for _, b := range clusters[i] {
			s := sim(cn, b)
			if s <= epsilon {
				near.Add(types.Int(i))
				if min_near == -1 || s < min_sim {
					min_near = i
					min_sim = s
					min_item = b
				}
			}
		}
	}
	if near.Size() <= 0 {
		return append(clusters, cluster{cn})
	}
	if false {
		errors.Logf("DBSCAN", "%v %v %v", min_sim, cn.n, min_item.n)
	}
	clusters[min_near] = append(clusters[min_near], cn)
	prev := -1
	for x, next := near.ItemsInReverse()(); next != nil; x, next = next() {
		cur := int(x.(types.Int))
		if prev >= 0 {
			clusters[cur] = append(clusters[cur], clusters[prev]...)
			clusters = remove(clusters, prev)
		}
		prev = cur
	}
	return clusters
}

func remove(list []cluster, i int) []cluster {
	if i >= len(list) {
		panic(fmt.Errorf("out of range (i (%v) >= len(list) (%v))", i, len(list)))
	} else if i < 0 {
		panic(fmt.Errorf("out of range (i (%v) < 0)", i))
	}
	for ; i < len(list)-1; i++ {
		list[i] = list[i+1]
	}
	list[len(list)-1] = nil
	return list[:len(list)-1]
}

func labelset(n *lattice.Node) (types.Set, error) {
	s := set.NewSortedSet(len(n.SubGraph.V) + len(n.SubGraph.E))
	for i := range n.SubGraph.V {
		s.Add(types.Int(n.SubGraph.V[i].Color))
	}
	for i := range n.SubGraph.E {
		s.Add(types.Int(n.SubGraph.E[i].Color))
	}
	return s, nil
}
