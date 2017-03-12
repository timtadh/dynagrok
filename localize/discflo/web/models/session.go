package models

import (
	"crypto"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)


type Session struct {
	Key string
	CsrfKey []byte
	Addr string
	UsrAgent string
	Created time.Time
	Accessed time.Time
	User string
}

type SessionStore interface {
	Name() string
	Get(key string) (*Session, error)
	Update(*Session) (error)
	Invalidate(*Session) (error)
}

func randBytes(length int) []byte {
	if urandom, err := os.Open("/dev/urandom"); err != nil {
		log.Fatal(err)
	} else {
		slice := make([]byte, length)
		if _, err := urandom.Read(slice); err != nil {
			log.Fatal(err)
		}
		urandom.Close()
		return slice
	}
	panic("unreachable")
}

func randUint64() uint64 {
	b := randBytes(8)
	return binary.LittleEndian.Uint64(b)
}

func userAgent(r *http.Request) string {
	if agent, has := r.Header["User-Agent"]; has {
		return strings.Join(agent, "; ")
	}
	return "None"
}

func ip(r *http.Request) string {
	return strings.SplitN(r.RemoteAddr, ":", 2)[0]
}

func key(name string, r *http.Request) (string, error) {
	c, err := r.Cookie(name)
	if err == nil {
		return c.Value, nil
	}
	return "", fmt.Errorf("Failed to extract session key")
}

func GetSession(store SessionStore, rw http.ResponseWriter, r *http.Request) (s *Session, err error) {
	name := store.Name()
	k, err := key(name, r)
	if err != nil {
		s = newSession(r)
	} else {
		s, err = store.Get(k)
		if err != nil {
			s = newSession(r)
		} else {
			err := s.update(name, r)
			if err != nil {
				log.Println(err)
				s = newSession(r)
			}
		}
	}
	err = store.Update(s)
	if err != nil {
		return nil, err
	}
	s.write(name, rw, r)
	return s, nil
}


func newSession(r *http.Request) *Session {
	return &Session{
		Key: hex.EncodeToString(randBytes(16)),
		CsrfKey: randBytes(64),
		Addr: ip(r),
		UsrAgent: userAgent(r),
		Created: time.Now().UTC(),
		Accessed: time.Now().UTC(),
	}
}

func (s *Session) Copy() *Session {
	c := *s
	return &c
}

func (s *Session) Csrf(obj string) string {
	h := crypto.SHA512.New()
	h.Write([]byte(obj))
	h.Write([]byte(s.CsrfKey))
	for i := 0; i < 100; i++ {
		h.Write(h.Sum(nil))
	}
	return base64.URLEncoding.EncodeToString(h.Sum(nil))
}

func (s *Session) ValidCsrf(obj, token string) bool {
	return s.Csrf(obj) == token
}

func (s *Session) Invalidate(store SessionStore, rw http.ResponseWriter) error {
	delete(rw.Header(), store.Name())
	return store.Invalidate(s)
}

func (s *Session) valid(name string, r *http.Request) bool {
	k, err := key(name, r)
	if err != nil {
		return false
	}
	ua := userAgent(r)
	addr := ip(r)
	return ua == s.UsrAgent && addr == s.Addr && k == s.Key
}

func (s *Session) update(name string, r *http.Request) error {
	if s.valid(name, r) {
		s.Accessed = time.Now().UTC()
		return nil
	}
	return fmt.Errorf("session was invalid")
}

func (s *Session) write(name string, rw http.ResponseWriter, r *http.Request) {
	secure := r.URL.Scheme == "https" || r.TLS != nil
	http.SetCookie(rw, &http.Cookie{
		Name: name,
		Value: s.Key,
		Path: "/",
		Secure: secure,
		HttpOnly: true,
	})
}

