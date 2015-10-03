package express

import (
	"html/template"
	"path/filepath"
	"os"
	"regexp"
	"github.com/gin-gonic/gin/render"
	"net/http"
	"fmt"
	"strings"
)

var (
	layoutpath = regexp.MustCompile(`.*layouts.*`)
	htmlfile = regexp.MustCompile(`.*\.html$`)
)

type HTMLTemplate map[string]*template.Template

type HTMLMaster struct {
	templates map[string]HTMLTemplate
}

type HTMLMasterInstance struct {
	Templates map[string]HTMLTemplate
	Layout    string
	Name      string
	Data      interface{}
}

func (r HTMLMaster) Instance(name string, data interface{}) render.Render {

	layout := "application"

	p := strings.Split(name, ":")
	if len(p) > 1 {
		layout = p[0]
		name = p[1]
	}

	return HTMLMasterInstance{
		Templates: r.templates,
		Layout: layout,
		Name: name,
		Data: data,
	}
}

func (r HTMLMasterInstance) Render(w http.ResponseWriter) error {

	layout, ok := r.Templates[r.Layout]
	if !ok {
		return fmt.Errorf("Layout %v not found", r.Layout)
	}

	template, ok := layout[r.Name]
	if !ok {
		return fmt.Errorf("Template %v not found", r.Name)
	}

	header := w.Header()
	if val := header["Content-Type"]; len(val) == 0 {
		header["Content-Type"] = []string{"text/html; charset=utf-8"}
	}

	return template.Execute(w, r.Data)
}

func NewHtmlMaster(path string) *HTMLMaster {

	debug("Loading templates")

	var (
		layouts []string
		views []string
	)

	templates := make(map[string]HTMLTemplate)

	err := filepath.Walk(path, func(path string, file os.FileInfo, err error) error {
		if htmlfile.MatchString(file.Name()) {

			if layoutpath.MatchString(path) {
				layouts = append(layouts, path)
				return nil
			}

			views = append(views, path)
			return nil
		}

		return nil
	})

	if err != nil {
		debug("[ERROR] %s", err.Error())
	}

	for _, layout := range layouts {
		k := strings.Replace(filepath.Base(layout), ".html", "", -1)
		for _, view := range views {
			_, ok := templates[k];
			if !ok {
				templates[k] = make(HTMLTemplate)
			}

			key := filepath.Base(filepath.Dir(view)) + "/" + filepath.Base(view)
			templates[k][key] = template.Must(template.ParseFiles(layout, view))
		}
	}

	return &HTMLMaster{templates}
}