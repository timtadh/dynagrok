package views

import (
	"fmt"
)

func (v *Views) Dotty(c *Context) error {
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
	bytes, err := clusters.Dotty(cid, nid)
	if err != nil {
		return err
	}
	c.rw.Header().Set("Content-Type", "text/graphviz")
	_, err = c.rw.Write([]byte(bytes))
	return err
}
