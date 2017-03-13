package views

import (
	"fmt"
	"strconv"
)

import ()

func (c *Context) indexIn(name string, collectionLength int) (int, error) {
	sid := c.p.ByName(name)
	id, err := strconv.Atoi(sid)
	if err != nil {
		return 0, fmt.Errorf("Expected an int got `%v` for :%v part. err: %v", sid, name, err)
	}
	if id < 0 {
		return 0, fmt.Errorf("%v was less than 0.", name)
	}
	if id >= collectionLength {
		return 0, fmt.Errorf("%v was out of range.", name)
	}
	return id, nil
}

func (v *Views) GenerateTest(c *Context) error {
	type data struct {
		ResultId  int
		ClusterId  int
		NodeId  int
		Test string
	}
	rid, err := c.indexIn("rid", len(v.result))
	if err != nil {
		return err
	}
	loc := &v.result[rid]
	cid, err := c.indexIn("cid", len(loc.Clusters))
	if err != nil {
		return err
	}
	cluster := loc.Clusters[cid]
	nid, err := c.indexIn("nid", len(cluster.Nodes))
	if err != nil {
		return err
	}
	node := cluster.Nodes[nid]
	if node.Test == nil {
		lat := v.opts.Lattice
		tests := v.opts.Tests
		for _, t := range tests {
			min, err := t.Minimize(lat, node.Node.SubGraph)
			if err != nil {
				return err
			}
			if min == nil {
				continue
			}
			node.Test = min
			break
		}
	}
	test := ""
	if node.Test != nil {
		test = string(node.Test.Case)
	}
	return v.tmpl.ExecuteTemplate(c.rw, "test", &data{
		rid,
		cid,
		nid,
		test,
	})
}

