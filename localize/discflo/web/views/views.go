package views

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"io/ioutil"
	"html/template"
)

import (
	"github.com/timtadh/data-structures/errors"
	"github.com/julienschmidt/httprouter"
)

import (
	"github.com/timtadh/dynagrok/localize/discflo/web/models"
	"github.com/timtadh/dynagrok/localize/discflo/web/models/mem"
)

type Views struct {
	assets string
	tmpl *template.Template
	sessions models.SessionStore
}

func Routes(assetPath string) (http.Handler, error) {
	mux := httprouter.New()
	v := &Views{
		assets: filepath.Clean(assetPath),
		sessions: mem.NewSessionMapStore("session"),
	}
	mux.GET("/", v.Context(v.Index))
	mux.ServeFiles("/static/*filepath", http.Dir(filepath.Join(assetPath, "static")))
	err := v.Init()
	if err != nil {
		return nil, err
	}
	return mux, nil
}
func (v *Views) Init() error {
	return v.loadTemplates()
}

func (v *Views) loadTemplates() error {
	s, err := os.Stat(v.assets)
	if os.IsNotExist(err) {
		return errors.Errorf("Could not load assets from %v. Path does not exist.", v.assets)
	} else if err != nil {
		return err
	}
	v.tmpl = template.New("!")
	if s.IsDir() {
		return v.loadTemplatesFromDir("", filepath.Join(v.assets, "templates"), v.tmpl)
	} else {
		return errors.Errorf("Could not load assets from %v. Unknown file type", v.assets)
	}
}

func (v *Views) loadTemplatesFromDir(ctx, path string, t *template.Template) error {
	dir, err := ioutil.ReadDir(path)
	if err != nil {
		return err
	}
	for _, info := range dir {
		c := filepath.Join(ctx, info.Name())
		p := filepath.Join(path, info.Name())
		if info.IsDir() {
			err := v.loadTemplatesFromDir(c, p, t)
			if err != nil {
				return err
			}
		} else {
			err := v.loadTemplateFile(ctx, p, t)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (v *Views) loadTemplateFile(ctx, path string, t *template.Template) error {
	name := filepath.Base(path)
	if strings.HasPrefix(name, ".") {
		return nil
	}
	ext := filepath.Ext(name)
	if ext != "" {
		name = strings.TrimSuffix(name, ext)
	}
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return v.loadTemplate(filepath.Join(ctx, name), string(content), t)
}

func (v *Views) loadTemplate(name, content string, t *template.Template) error {
	log.Println("loaded template", name)
	_, err := t.New(name).Parse(content)
	if err != nil {
		return err
	}
	return nil
}
