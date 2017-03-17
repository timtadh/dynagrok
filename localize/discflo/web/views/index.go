package views

func (v *Views) Index(c *Context) error {
	return v.tmpl.ExecuteTemplate(c.rw, "index", map[string]interface{}{})
}
