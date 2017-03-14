package views

import (
	"fmt"
)

import (
	"github.com/timtadh/dynagrok/localize/discflo/web/models"
)

func (v *Views) Block(c *Context) error {
	type data struct {
		Color        int
		FnName       string
		BasicBlockId int
		Clusters     []*models.Cluster
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
	bbid, fnName, _ := v.opts.Lattice.Info.Get(color)
	return v.tmpl.ExecuteTemplate(c.rw, "block", &data{
		Color: color,
		FnName: fnName,
		BasicBlockId: bbid,
		Clusters: colors[color],
	})
}

