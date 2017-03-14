package views

import (
	"fmt"
	"strconv"
)

import ()

func inSlice(length int) func(idx int) bool {
	return func(idx int) bool {
		return idx >= 0 && idx < length
	}
}

func (c *Context) indexIn(name string, has func(int) bool) (int, error) {
	sid := c.p.ByName(name)
	id, err := strconv.Atoi(sid)
	if err != nil {
		return 0, fmt.Errorf("Expected an int got `%v` for :%v part. err: %v", sid, name, err)
	}
	if !has(id) {
		return 0, fmt.Errorf("%v was less out of range.", name)
	}
	return id, nil
}

func (v *Views) GenerateTest(c *Context) error {
	type data struct {
		ClusterId  int
		NodeId  int
		Test string
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
	testBytes, err := clusters.Test(cid, nid)
	if err != nil {
		return err
	}
	test := ""
	if testBytes != nil {
		test = string(testBytes)
	}
	return v.tmpl.ExecuteTemplate(c.rw, "test", &data{
		ClusterId: cid,
		NodeId: nid,
		Test: test,
	})
}

