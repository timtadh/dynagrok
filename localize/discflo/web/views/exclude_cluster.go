package views

import (
	"net/http"
)

func (v *Views) ExcludeCluster(c *Context) error {
	clusters, err := v.localization.Clusters()
	if err != nil {
		return err
	}
	cid, err := c.indexIn("cid", clusters.Has)
	if err != nil {
		return err
	}
	err = clusters.Exclude(cid)
	if err != nil {
		return err
	}
	http.Redirect(c.rw, c.r, "/blocks", http.StatusFound)
	return nil
}
