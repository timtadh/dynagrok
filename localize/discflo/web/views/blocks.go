package views

import (
	"github.com/timtadh/dynagrok/localize/discflo/web/models"
)

func (v *Views) Blocks(c *Context) error {
	type data struct {
		Blocks models.Blocks
	}
	clusters, err := v.localization.Clusters()
	if err != nil {
		return err
	}
	return v.tmpl.ExecuteTemplate(c.rw, "blocks", data{
		Blocks: clusters.Blocks(),
	})
}

