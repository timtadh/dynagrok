package views

import (
	"fmt"
	"net/http"
)


func (v *Views) ExcludeCluster(c *Context) error {
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
	gcid := -1
	for i, c := range v.clusters {
		if c == cluster {
			gcid = i
			break
		}
	}
	if gcid < 0 {
		return fmt.Errorf("could not find cluster")
	}
	dst := v.clusters[gcid : len(v.clusters)-1]
	src := v.clusters[gcid+1 : len(v.clusters)]
	copy(dst, src)
	v.clusters = v.clusters[:len(v.clusters)-1]
	v.result = v.clusters.RankColors(v.opts.Score, v.opts.Lattice)
	http.Redirect(c.rw, c.r, "/blocks", http.StatusFound)
	return nil
}

