package views

import (
	"log"
	"net/http"
	"runtime/debug"
)

import (
	"github.com/julienschmidt/httprouter"
)

import (
	"github.com/timtadh/dynagrok/localize/discflo/web/models"
)

type View func(*Context)

type Context struct {
	views *Views
	s *models.Session
	rw http.ResponseWriter
	r *http.Request
	p httprouter.Params
}

func (v *Views) Context(f View) httprouter.Handle {
	return func(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
		defer func() {
			if e := recover(); e != nil {
				log.Println(e)
				log.Println(string(debug.Stack()))
				rw.WriteHeader(500)
				rw.Write([]byte("Internal Error"))
			}
			return
		}()
		c := &Context{
			views: v,
			rw: rw, r: r, p: p,
		}
		c.Session(v.Log(f))
	}
}

func (c *Context) Session(f View) {
	doErr := func(c *Context, err error) {
		log.Println(err)
		c.rw.WriteHeader(500)
		c.rw.Write([]byte("error processing request"))
	}
	s, err := models.GetSession(c.views.sessions, c.rw, c.r)
	if err != nil {
		doErr(c, err)
		return
	}
	c.s = s
	// if s.User != "" {
	// 	u, err := c.views.users.Get(s.User)
	// 	if err != nil {
	// 		doErr(c, err)
	// 		return
	// 	}
	// 	c.u = u
	// }
	f(c)
}

func (v *Views) Index(c *Context) {
	err := v.tmpl.ExecuteTemplate(c.rw, "index", nil)
	if err != nil {
		log.Panic(err)
	}
}

