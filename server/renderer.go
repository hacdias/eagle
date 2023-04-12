package server

import (
	"errors"
	"html/template"
	"io"
	osfs "io/fs"
	"path"
	"path/filepath"
	"strings"
)

const (
	TemplatesExtension string = ".html"
	TemplatesDirectory string = "templates"

	// TemplateSearch    string = "search"
	TemplateBase      string = "base"
	TemplateError     string = "error"
	TemplateLogin     string = "login"
	TemplateAuth      string = "auth"
	TemplateNew       string = "new"
	TemplateEdit      string = "edit"
	TemplateDashboard string = "dashboard"
)

type RenderData struct {
	Title string
	Data  interface{}
}

func (s *Server) render(w io.Writer, data *RenderData, template string) error {
	tpl, ok := s.templates[template]
	if !ok {
		return errors.New("unrecognized template")
	}

	return tpl.Execute(w, data)
}

func (s *Server) loadTemplates() error {
	baseTemplateFilename := path.Join(TemplatesDirectory, TemplateBase+TemplatesExtension)
	baseTemplateData, err := s.fs.ReadFile(baseTemplateFilename)
	if err != nil {
		return err
	}

	baseTemplate, err := template.New("base").Parse(string(baseTemplateData))
	if err != nil {
		return err
	}
	parsed := map[string]*template.Template{}

	err = s.fs.Walk(TemplatesDirectory, func(filename string, info osfs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		basename := filepath.Base(info.Name())
		ext := filepath.Ext(basename)

		id := strings.TrimPrefix(filename, TemplatesDirectory)
		id = strings.TrimSuffix(id, ext)
		id = strings.TrimSuffix(id, "/")
		id = strings.TrimPrefix(id, "/")

		if ext != TemplatesExtension || id == TemplateBase {
			return nil
		}

		raw, err := s.fs.ReadFile(filename)
		if err != nil {
			return err
		}

		parsed[id], err = template.Must(baseTemplate.Clone()).New(id).Parse(string(raw))
		return err
	})

	if err != nil {
		return err
	}

	s.templates = parsed
	return nil
}
