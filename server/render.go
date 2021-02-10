package server

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

const dashboardPath = "/dashboard"

type dashboardData struct {
	Base      string
	Content   string
	ID        string
	DefaultID string
	Reply     string
	Edit      string
	Delete    string
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

func getTemplates() (map[string]*template.Template, error) {
	templates := map[string]*template.Template{}
	baseTpl := template.Must(template.ParseFiles("dashboard/templates/base.html"))

	err := filepath.Walk("dashboard/templates", func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		basename := filepath.Base(info.Name())
		if basename == "base" {
			return nil
		}

		ext := filepath.Ext(basename)
		id := strings.TrimSuffix(basename, ext)

		raw, err := ioutil.ReadFile(p)
		if err != nil {
			return err
		}

		templates[id] = template.Must(template.Must(baseTpl.Clone()).New(id).Parse(string(raw)))
		return nil
	})

	return templates, err
}
