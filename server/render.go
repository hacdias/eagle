package server

import (
	"bytes"
	"html/template"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/hacdias/eagle/dashboard/templates"
	"github.com/hacdias/eagle/eagle"
	"github.com/spf13/afero"
)

type dashboardData struct {
	// Common To All Pages
	LoggedIn bool

	// Cleanup
	Content string
	ID      string

	// ReShare Page Only
	Targets []string

	// Root Page Only
	Entries      []*eagle.SearchEntry
	Drafts       bool
	Query        string
	NextPage     string
	PreviousPage string
}

func (s *Server) renderDashboard(w http.ResponseWriter, tpl string, data *dashboardData) {
	tpls, err := s.getTemplates()
	if err != nil {
		s.serveError(w, http.StatusInternalServerError, err)
		return
	}

	data.LoggedIn = tpl != "login"

	var buf bytes.Buffer
	err = tpls[tpl].ExecuteTemplate(&buf, tpl, data)
	if err != nil {
		s.serveError(w, http.StatusInternalServerError, err)
		return
	}

	w.Header().Set("Content-type", "text/html; charset=urf-8")
	_, _ = w.Write(buf.Bytes())
}

type readDirFileFS interface {
	fs.ReadDirFS
	fs.ReadFileFS
}

func (s *Server) getTemplates() (map[string]*template.Template, error) {
	if s.templates != nil {
		return s.templates, nil
	}

	var fs readDirFileFS

	if s.c.Development {
		fs = afero.NewIOFS(afero.NewBasePathFs(afero.NewOsFs(), "./dashboard/templates"))
	} else {
		fs = templates.FS
	}

	baseRaw, err := fs.ReadFile("base.html")
	if err != nil {
		return nil, err
	}

	baseTpl := template.Must(template.New("base").Parse(string(baseRaw)))

	files, err := fs.ReadDir(".")
	if err != nil {
		return nil, err
	}

	parsed := map[string]*template.Template{}
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

		raw, err := fs.ReadFile(info.Name())
		if err != nil {
			return nil, err
		}

		parsed[id] = template.Must(template.Must(baseTpl.Clone()).New(id).Parse(string(raw)))
	}

	if !s.c.Development {
		s.templates = parsed
	}

	return parsed, err
}
