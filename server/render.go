package server

import (
	"bytes"
	"net/http"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/hacdias/eagle/dashboard/templates"
)

const dashboardPath = "/dashboard"

type dashboardData struct {
	Base       string
	Content    string
	ID         string
	DraftsList []interface{}
}

func (s *Server) renderDashboard(w http.ResponseWriter, tpl string, data *dashboardData) {
	data.Base = dashboardPath

	tpls, err := getTemplates()
	if err != nil {
		s.serveError(w, http.StatusInternalServerError, err)
		return
	}

	var buf bytes.Buffer
	err = tpls[tpl].ExecuteTemplate(&buf, tpl, data)
	if err != nil {
		s.serveError(w, http.StatusInternalServerError, err)
		return
	}

	w.Header().Set("Content-type", "text/html; charset=urf-8")
	_, _ = w.Write(buf.Bytes())
}

// TODO: only load templates once.
func getTemplates() (map[string]*template.Template, error) {
	parsed := map[string]*template.Template{}
	baseTpl := template.Must(template.New("base").Parse(templates.Base))

	files, err := templates.FS.ReadDir(".")
	if err != nil {
		return nil, err
	}

	for _, info := range files {
		if info.IsDir() {
			continue
		}

		basename := filepath.Base(info.Name())
		if basename == "base" {
			continue
		}

		ext := filepath.Ext(basename)
		id := strings.TrimSuffix(basename, ext)

		raw, err := templates.FS.ReadFile(info.Name())
		if err != nil {
			return nil, err
		}

		parsed[id] = template.Must(template.Must(baseTpl.Clone()).New(id).Parse(string(raw)))
	}

	return parsed, err
}
