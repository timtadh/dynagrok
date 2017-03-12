package views

import (
	"log"
	"net/http"
	"time"
)

import (
)

type loggingRW struct {
	rw http.ResponseWriter
	total int
}

func (l *loggingRW) Header() http.Header {
	return l.rw.Header()
}

func (l *loggingRW) Write(bytes []byte) (int, error) {
	c, err := l.rw.Write(bytes)
	l.total += c
	return c, err
}

func (l *loggingRW) WriteHeader(code int) {
	l.rw.WriteHeader(code)
}

func (v *Views) Log(f View) View {
	return func(c *Context) {
		rw := c.rw
		lrw := &loggingRW{rw: rw}
		c.rw = lrw
		s := time.Now()
		f(c)
		e := time.Now()
		log.Printf("%v %-4v %v (%v) %v (%d) %v",
			c.r.RemoteAddr, c.r.Method, c.r.URL, c.r.ContentLength, c.s.Key, lrw.total, e.Sub(s))
	}
}

