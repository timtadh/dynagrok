package views

func (v *Views) Blocks(c *Context) error {
	return v.tmpl.ExecuteTemplate(c.rw, "blocks", map[string]interface{}{
		"result": v.result,
	})
}

