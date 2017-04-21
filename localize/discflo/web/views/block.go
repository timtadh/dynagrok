package views

import (
	"fmt"
)

import (
	"github.com/timtadh/dynagrok/localize/discflo/web/models"
	"github.com/timtadh/dynagrok/localize/mine"
	"github.com/timtadh/dynagrok/localize/test"
)

func (v *Views) Block(c *Context) error {
	type node struct {
		*mine.SearchNode
		MinimizableTests map[int]*test.Testcase
	}
	type cluster struct {
		Cluster *models.Cluster
		Score   float64
		Nodes   []*node
	}
	type data struct {
		Color        int
		FnName       string
		BasicBlockId int
		Clusters     []*cluster
	}
	clusters, err := v.localization.Clusters()
	if err != nil {
		return err
	}
	colors := clusters.AllColors()
	color, err := c.indexIn("color", inSlice(len(colors)))
	if err != nil {
		return err
	}
	if colors[color] == nil {
		return fmt.Errorf("no clusters for color %v (%v)", color, v.opts.Lattice.Labels.Label(color))
	}
	clstrs := make([]*cluster, 0, len(colors[color]))
	for _, c := range colors[color] {
		nodes := make([]*node, 0, len(c.Nodes))
		for nid, n := range c.Nodes {
			mt, err := clusters.MinimizableTests(c.Id, nid)
			if err != nil {
				return err
			}
			for tid, _ := range n.Tests {
				delete(mt, tid)
			}
			nodes = append(nodes, &node{
				SearchNode:       n,
				MinimizableTests: mt,
			})
		}
		clstrs = append(clstrs, &cluster{
			Cluster: c,
			Score:   c.Score,
			Nodes:   nodes,
		})
	}
	bbid, fnName, _ := v.opts.Lattice.Info.Get(color)
	return v.tmpl.ExecuteTemplate(c.rw, "block", &data{
		Color:        color,
		FnName:       fnName,
		BasicBlockId: bbid,
		Clusters:     clstrs,
	})
}
