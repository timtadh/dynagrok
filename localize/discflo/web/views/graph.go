package views

import (
	"fmt"
	"sort"
)

import (
	"github.com/timtadh/dynagrok/localize/discflo"
	"github.com/timtadh/dynagrok/localize/mine"
	"github.com/timtadh/dynagrok/localize/test"
)

func (v *Views) Graph(c *Context) error {
	type location struct {
		Color, BasicBlockId int
		Position, FnName    string
		Score               float64
	}
	type data struct {
		ClusterId        int
		NodeId           int
		Score            float64
		MinimizedTests   map[int]*test.Testcase
		MinimizableTests map[int]*test.Testcase
		Locations        []*location
	}
	clusters, err := v.localization.Clusters()
	if err != nil {
		return err
	}
	cid, err := c.indexIn("cid", clusters.Has)
	if err != nil {
		return err
	}
	cluster := clusters.Get(cid)
	if cluster == nil {
		return fmt.Errorf("cluster %v was nil", cid)
	}
	nid, err := c.indexIn("nid", inSlice(len(cluster.Nodes)))
	if err != nil {
		return err
	}
	n := cluster.Nodes[nid]
	mt, err := clusters.MinimizableTests(cid, nid)
	if err != nil {
		return err
	}
	for tid, _ := range n.Tests {
		delete(mt, tid)
	}
	o := v.opts
	miner := mine.NewMiner(o.Miner, o.Lattice, o.Score, o.Opts...)
	colors := clusters.AllColors()
	sg := n.Node.SubGraph
	locations := make([]*location, 0, len(sg.V))
	for _, u := range sg.V {
		bbid, fnName, pos := v.opts.Lattice.Info.Get(u.Color)
		locations = append(locations, &location{
			Color:        u.Color,
			BasicBlockId: bbid,
			FnName:       fnName,
			Position:     pos,
			Score:        discflo.ScoreColor(miner, u.Color, clusters.AsDiscflo(colors[u.Color])),
		})
	}
	sort.Slice(locations, func(i, j int) bool {
		return locations[i].Score > locations[j].Score
	})
	return v.tmpl.ExecuteTemplate(c.rw, "graph", &data{
		ClusterId:        cid,
		NodeId:           nid,
		Score:            n.Score,
		MinimizedTests:   n.Tests,
		MinimizableTests: mt,
		Locations:        locations,
	})
}
