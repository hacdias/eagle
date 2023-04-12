package server

import (
	"bytes"
	"fmt"
	"html/template"
	osfs "io/fs"
	"path/filepath"
	"strings"
)

const (
	TemplatesExtension string = ".html"
	TemplatesDirectory string = "eagle/templates"

	// TemplateSearch    string = "search"
	TemplateError     string = "error"
	TemplateLogin     string = "login"
	TemplateAuth      string = "auth"
	TemplateNew       string = "new"
	TemplateEdit      string = "edit"
	TemplateDashboard string = "dashboard"

	TemplateAdminBar string = "admin-bar"
)

type RenderData struct {
	Title    string
	LoggedIn bool
	Data     interface{}
}

func (s *Server) loadTemplates() error {
	parsed := map[string]*template.Template{}

	err := s.fs.Walk(TemplatesDirectory, func(filename string, info osfs.FileInfo, err error) error {
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

		if ext != TemplatesExtension {
			return nil
		}

		raw, err := s.fs.ReadFile(filename)
		if err != nil {
			return err
		}

		parsed[id], err = template.New(id).Parse(string(raw))
		return err
	})

	if err != nil {
		return err
	}

	s.templates = parsed
	return nil
}

func (s *Server) renderAdminBar(path string) ([]byte, error) {
	tpl, ok := s.templates[TemplateAdminBar]
	if !ok {
		return nil, fmt.Errorf("template %s not found", TemplateAdminBar)
	}

	var buf bytes.Buffer
	err := tpl.Execute(&buf, []string{})
	if err != nil {
		return nil, err
	}

	// data := &dashboardData{
	// 	HasAuth:  s.Config.Auth != nil,
	// 	BasePath: dashboardPath,
	// 	Data: map[string]interface{}{
	// 		"ID": path,
	// 	},
	// }

	return buf.Bytes(), nil
}
