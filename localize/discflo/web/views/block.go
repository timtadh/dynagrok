package views

import (
	"github.com/timtadh/dynagrok/localize/discflo"
)

func (v *Views) Block(c *Context) error {
	type data struct {
		Id  int
		Loc *discflo.Location
	}
	id, err := c.indexIn("rid", len(v.result))
	if err != nil {
		return err
	}
	return v.tmpl.ExecuteTemplate(c.rw, "block", &data{
		id,
		&v.result[id],
	})
}

