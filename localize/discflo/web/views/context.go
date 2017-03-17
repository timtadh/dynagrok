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

type View func(*Context) error

type Context struct {
	views *Views
	s     *models.Session
	rw    http.ResponseWriter
	r     *http.Request
	p     httprouter.Params
}

func (v *Views) Context(f View) httprouter.Handle {
	return func(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
		defer func() {
			if e := recover(); e != nil {
				log.Println(e)
				log.Println(string(debug.Stack()))
				rw.WriteHeader(500)
				n, err := rw.Write([]byte("Internal Error"))
				if err != nil {
					log.Println("err:", n, err)
				}
			}
			return
		}()
		c := &Context{
			views: v,
			rw:    rw, r: r, p: p,
		}
		err := c.Session(v.Log(f))
		if err != nil {
			log.Println(err)
			rw.WriteHeader(500)
			n, err := rw.Write([]byte("Internal Error"))
			if err != nil {
				log.Println("err:", n, err)
			}
		}
	}
}

func (c *Context) Session(f View) error {
	s, err := models.GetSession(c.views.sessions, c.rw, c.r)
	if err != nil {
		return err
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
	return f(c)
}
