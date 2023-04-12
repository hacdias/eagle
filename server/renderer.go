package server

import (
	"html/template"
	osfs "io/fs"
	"path/filepath"
	"strings"
)

type RenderData struct {
	Title    string
	LoggedIn bool
	Data     interface{}
}

func (s *Server) loadTemplates() error {
	parsed := map[string]*template.Template{}

	err := s.fs.Walk(templatesDirectory, func(filename string, info osfs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		basename := filepath.Base(info.Name())
		ext := filepath.Ext(basename)

		id := strings.TrimPrefix(filename, templatesDirectory)
		id = strings.TrimSuffix(id, ext)
		id = strings.TrimSuffix(id, "/")
		id = strings.TrimPrefix(id, "/")

		if ext != templatesExtension {
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
