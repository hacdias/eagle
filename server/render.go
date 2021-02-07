package server

import (
	"bytes"
	"encoding/json"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	fhtml "github.com/alecthomas/chroma/formatters/html"
	"github.com/hacdias/eagle"
	"github.com/hacdias/eagle/config"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

var md = goldmark.New(
	goldmark.WithRendererOptions(
		html.WithUnsafe(),
	),
	goldmark.WithParserOptions(
		parser.WithAutoHeadingID(),
	),
	goldmark.WithExtensions(
		extension.Table,
		extension.Strikethrough,
		extension.Footnote,
		extension.Typographer,
		extension.Linkify,
		highlighting.NewHighlighting(
			highlighting.WithStyle("dracula"),
			highlighting.WithFormatOptions(
				fhtml.WithAllClasses(true),
			),
		),
	),
)

type page struct {
	*eagle.Entry
	*eagle.EntryMetadata
	Site    *config.Site
	IsHome  bool
	Content template.HTML
}

func (s *Server) serveHTML(w http.ResponseWriter, entry *eagle.Entry) {
	// Only for testing purposes, remove
	layouts, err := getTemplates(s.c.Source)
	if err != nil {
		panic(err)
	}
	s.layouts = layouts

	var buf bytes.Buffer
	err = md.Convert([]byte(entry.Content), &buf)
	if err != nil {
		s.serveHTMLError(w, http.StatusInternalServerError, err)
		return
	}

	if entry.Metadata.Layout == "" {
		entry.Metadata.Layout = "single"
	}

	// TODO: add context specific functions
	tpl := template.Must(s.layouts[entry.Metadata.Layout].Clone())
	err = tpl.ExecuteTemplate(w, entry.Metadata.Layout, &page{
		Entry:         entry,
		EntryMetadata: &entry.Metadata,
		Site:          &s.c.Site,
		Content:       template.HTML(buf.Bytes()),
		IsHome:        entry.ID == "/",
	})

	if err != nil {
		// TODO: this causes superfluous call if tpl fails in the middle
		s.serveHTMLError(w, http.StatusInternalServerError, err)
		return
	}
}

func (s *Server) serveHTMLError(w http.ResponseWriter, code int, err error) {
	// TODO
	w.WriteHeader(code)
	w.Write([]byte(err.Error()))
}

func (s *Server) serveJSON(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		s.Errorf("error while serving json: %s", err)
	}
}

func (s *Server) serveJSONError(w http.ResponseWriter, code int, err error) {
	s.serveJSON(w, code, map[string]interface{}{
		"error":             http.StatusText(code),
		"error_description": err.Error(),
	})
}

func getTemplates(source string) (map[string]*template.Template, error) {
	var includes *template.Template

	includesDir := filepath.Join(source, "templates", "includes")
	err := filepath.Walk(includesDir, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		basename := filepath.Base(info.Name())
		ext := filepath.Ext(basename)
		id := strings.TrimSuffix(basename, ext)

		raw, err := ioutil.ReadFile(p)
		if err != nil {
			return err
		}

		if includes == nil {
			includes = template.Must(template.New(id).Funcs(funcMap).Parse(string(raw)))
		} else {
			includes = template.Must(includes.New(id).Parse(string(raw)))
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	layouts := map[string]*template.Template{}
	layoutsDir := filepath.Join(source, "templates", "layouts")
	err = filepath.Walk(layoutsDir, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		basename := filepath.Base(info.Name())
		ext := filepath.Ext(basename)
		id := strings.TrimSuffix(basename, ext)

		raw, err := ioutil.ReadFile(p)
		if err != nil {
			return err
		}

		layouts[id] = template.Must(template.Must(includes.Clone()).New(id).Parse(string(raw)))
		return nil
	})

	return layouts, err
}

var funcMap = template.FuncMap{
	"relURL": func(page *page, url string) string {
		return url
	},
}
