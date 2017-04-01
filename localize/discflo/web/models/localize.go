package models

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"sort"
	"sync"
	"time"
)

import (
	"github.com/timtadh/dynagrok/localize/discflo"
	"github.com/timtadh/dynagrok/localize/lattice"
	"github.com/timtadh/dynagrok/localize/mine"
	"github.com/timtadh/dynagrok/localize/stat"
	"github.com/timtadh/dynagrok/localize/test"
)

type Localization struct {
	lock     sync.Mutex
	opts     *discflo.Options
	clusters *Clusters
}

type Clusters struct {
	lock      sync.Mutex
	tests     []*test.Testcase
	lat       *lattice.Lattice
	miner     *mine.Miner
	opts      *discflo.Options
	included  []*Cluster
	excluded  []*Cluster
	clusters  map[int]*Cluster
	allColors [][]*Cluster
	colors    map[int][]*Cluster
	imgs      map[sgid][]byte
	blocks    Blocks
}

type sgid struct {
	ClusterId, NodeId int
}

type Cluster struct {
	discflo.Cluster
	Id               int
	IncludedIdx      int
	ExcludedIdx      int
	MinimizableTests map[int]map[int]*test.Testcase
	                 // node-id -> test-id -> original-test
}

type Blocks []*Block

type Block struct {
	stat.Location
	In []*Cluster
}

func (b Blocks) Sort() {
	sort.SliceStable(b, func(i, j int) bool {
		return b[i].Score > b[j].Score
	})
}

func (b Blocks) Group() []Blocks {
	b.Sort()
	groups := make([]Blocks, 0, 10)
	for _, n := range b {
		lg := len(groups)
		if lg > 0 && n.Score == groups[lg-1][0].Score {
			groups[lg-1] = append(groups[lg-1], n)
		} else {
			groups = append(groups, make(Blocks, 0, 10))
			groups[lg] = append(groups[lg], n)
		}
	}
	return groups
}

func Localize(opts *discflo.Options) *Localization {
	return &Localization{
		opts: opts,
	}
}

func (l *Localization) Test(tid int) (from string, test []byte) {
	l.lock.Lock()
	defer l.lock.Unlock()
	return l.opts.Failing[tid].From, l.opts.Failing[tid].Case
}

func (l *Localization) Lattice() (*lattice.Lattice) {
	l.lock.Lock()
	defer l.lock.Unlock()
	return l.opts.Lattice
}

func (l *Localization) Clusters() (*Clusters, error) {
	l.lock.Lock()
	defer l.lock.Unlock()

	if l.clusters != nil {
		return l.clusters, nil
	}
	o := l.opts
	miner := mine.NewMiner(o.Miner, o.Lattice, o.Score, o.Opts...)
	clusters, err := discflo.Localizer(l.opts)(miner)
	if err != nil {
		return nil, err
	}
	l.clusters = l.newClusters(miner, clusters)
	return l.clusters, nil
}

func (l *Localization) Exclude(id int) error {
	clusters, err := l.Clusters()
	if err != nil {
		return err
	}
	return clusters.Exclude(id)
}

func (l *Localization) newClusters(miner *mine.Miner, clusters discflo.Clusters) *Clusters {
	c := &Clusters{
		lat:      l.opts.Lattice,
		miner:    miner,
		tests:    l.opts.Failing,
		included: make([]*Cluster, 0, len(clusters)),
		excluded: make([]*Cluster, 0, len(clusters)),
		clusters: make(map[int]*Cluster, len(clusters)),
		imgs:     make(map[sgid][]byte),
	}
	for i, x := range clusters {
		cluster := &Cluster{
			Cluster:     *x,
			Id:          i,
			IncludedIdx: i,
			ExcludedIdx: -1,
			MinimizableTests: make(map[int]map[int]*test.Testcase),
		}
		c.included = append(c.included, cluster)
		c.clusters[cluster.Id] = cluster
	}
	colors := c.Colors()
	c.allColors = make([][]*Cluster, len(c.lat.Labels.Labels()))
	for color := range c.lat.Labels.Labels() {
		if clusters, has := colors[color]; has {
			c.allColors[color] = clusters
		}
	}
	return c
}

func (c *Clusters) AsDiscflo(clusters []*Cluster) []*discflo.Cluster {
	df := make([]*discflo.Cluster, 0, len(clusters))
	for _, c := range clusters {
		df = append(df, &c.Cluster)
	}
	return df
}

func (c *Clusters) AllColors() [][]*Cluster {
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.allColors
}

func (c *Clusters) Colors() map[int][]*Cluster {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.colors != nil {
		return c.colors
	}

	colors := make(map[int][]*Cluster)
	for _, clstr := range c.included {
		added := make(map[int]bool)
		for _, n := range clstr.Nodes {
			for j := range n.Node.SubGraph.V {
				if added[n.Node.SubGraph.V[j].Color] {
					continue
				}
				colors[n.Node.SubGraph.V[j].Color] = append(colors[n.Node.SubGraph.V[j].Color], clstr)
				added[n.Node.SubGraph.V[j].Color] = true
			}
		}
	}
	c.colors = colors
	return c.colors
}

func (c *Clusters) Blocks() Blocks {
	colors := c.Colors()

	c.lock.Lock()
	defer c.lock.Unlock()

	if c.blocks != nil {
		return c.blocks
	}

	blocks := make(Blocks, 0, len(colors))
	for color, clusters := range colors {
		bbid, fnName, pos := c.lat.Info.Get(color)
		blocks = append(blocks, &Block{
			In: clusters,
			Location: stat.Location{
				Score:        discflo.ScoreColor(c.miner, color, c.AsDiscflo(clusters)),
				Color:        color,
				Position:     pos,
				FnName:       fnName,
				BasicBlockId: bbid,
			},
		})
	}
	blocks.Sort()
	c.blocks = blocks
	return c.blocks
}

func (c *Clusters) Has(id int) bool {
	c.lock.Lock()
	defer c.lock.Unlock()
	_, has := c.clusters[id]
	return has
}

func (c *Clusters) HasTest(id int) bool {
	c.lock.Lock()
	defer c.lock.Unlock()
	if id >= 0 && id < len(c.tests) {
		return true
	}
	return false
}

func (c *Clusters) Get(id int) *Cluster {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.clusters[id]
}

func (c *Clusters) Exclude(id int) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	cluster, has := c.clusters[id]
	if !has {
		return fmt.Errorf("Could not find cluster %v in the clusters map.", id)
	}
	if cluster.IncludedIdx < 0 {
		return nil
	}

	{ // remove from included
		dst := c.included[cluster.IncludedIdx : len(c.included)-1]
		src := c.included[cluster.IncludedIdx+1 : len(c.included)]
		copy(dst, src)
		c.included = c.included[:len(c.included)-1]
		cluster.IncludedIdx = -1
	}

	// add to excluded
	cluster.ExcludedIdx = len(c.excluded)
	c.excluded = append(c.excluded, cluster)

	// renumber
	for i, clstr := range c.included {
		clstr.IncludedIdx = i
	}
	for i, clstr := range c.excluded {
		clstr.ExcludedIdx = i
	}

	c.colors = nil
	c.blocks = nil
	return nil
}

func (c *Clusters) MinimizableTests(cid, nid int) (map[int]*test.Testcase, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	cluster, has := c.clusters[cid]
	if !has {
		return nil, fmt.Errorf("Could not find cluster %v in the clusters map.", cid)
	}
	if nid < 0 || nid >= len(cluster.Nodes) {
		return nil, fmt.Errorf("Could not find node %v in the for cluster %v.", nid, cid)
	}
	node := cluster.Nodes[nid]

	if cluster.MinimizableTests[nid] != nil {
		return cluster.MinimizableTests[nid], nil
	}

	tests := make(map[int]*test.Testcase)

	for i, t := range c.tests {
		can, err := t.CanMinimize(c.lat, node.Node.SubGraph)
		if err != nil {
			return nil, err
		}
		if can {
			tests[i] = t
		}
	}

	cluster.MinimizableTests[nid] = tests
	return cluster.MinimizableTests[nid], nil
}

func (c *Clusters) Test(tid, cid, nid int) (*test.Testcase, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	cluster, has := c.clusters[cid]
	if !has {
		return nil, fmt.Errorf("Could not find cluster %v in the clusters map.", cid)
	}
	if nid < 0 || nid >= len(cluster.Nodes) {
		return nil, fmt.Errorf("Could not find node %v in the for cluster %v.", nid, cid)
	}
	node := cluster.Nodes[nid]

	if tid < 0 || tid >= len(c.tests) {
		for i, t := range c.tests {
			if node.Tests[i] != nil {
				return node.Tests[i], nil
			}
			min, err := t.Minimize(c.lat, node.Node.SubGraph)
			if err != nil {
				return nil, err
			}
			if min == nil {
				continue
			}
			node.Tests[i] = min
			return node.Tests[i], nil
		}
		return nil, nil
	} else {
		if node.Tests[tid] != nil {
			return node.Tests[tid], nil
		}
		t := c.tests[tid]
		min, err := t.Minimize(c.lat, node.Node.SubGraph)
		if err != nil {
			return nil, err
		}
		if min == nil {
			return nil, nil
		}
		node.Tests[tid] = min
		return min, nil
	}
}

func (c *Clusters) Img(id, nid int) ([]byte, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if img, has := c.imgs[sgid{id, nid}]; has {
		return img, nil
	}

	cluster, has := c.clusters[id]
	if !has {
		return nil, fmt.Errorf("Could not find cluster %v in the clusters map.", id)
	}
	if nid < 0 || nid >= len(cluster.Nodes) {
		return nil, fmt.Errorf("Could not find node %v in the for cluster %v.", nid, id)
	}
	node := cluster.Nodes[nid]

	dotty := node.Node.SubGraph.Dotty(c.lat.Labels)
	var outbuf, errbuf bytes.Buffer
	inbuf := bytes.NewBuffer([]byte(dotty))
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "dot", "-Tpng")
	cmd.Stdin = inbuf
	cmd.Stdout = &outbuf
	cmd.Stderr = &errbuf
	err := cmd.Run()
	if err != nil {
		return nil, err
	}
	img := outbuf.Bytes()
	c.imgs[sgid{id, nid}] = img
	return img, nil
}

func (c *Clusters) Dotty(id, nid int) (string, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	cluster, has := c.clusters[id]
	if !has {
		return "", fmt.Errorf("Could not find cluster %v in the clusters map.", id)
	}
	if nid < 0 || nid >= len(cluster.Nodes) {
		return "", fmt.Errorf("Could not find node %v in the for cluster %v.", nid, id)
	}
	node := cluster.Nodes[nid]

	return node.Node.SubGraph.Dotty(c.lat.Labels), nil
}

func (c *Cluster) Included() bool {
	return c.IncludedIdx >= 0
}
