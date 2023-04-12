package server

import (
	"errors"
	"html/template"
	"io"
	osfs "io/fs"
	"path"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/hacdias/eagle/log"
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

func (s *Server) watch(dir string, exec func() error) {
	log := log.S().Named("renderer")

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Error(err)
		return
	}
	defer watcher.Close()

	go func() {
		for {
			select {
			case evt, ok := <-watcher.Events:
				if !ok {
					return
				}

				// Ignore CHMOD only events.
				if evt.Op != fsnotify.Chmod {
					log.Infof("%s changed", evt.Name)
					err := exec()
					if err != nil {
						log.Error(err)
					}
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Error(err)
			}
		}
	}()

	err = s.fs.Walk(dir, func(filename string, info osfs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			return nil
		}

		return watcher.Add(filepath.Join(s.c.Source.Directory, filename))
	})
	if err != nil {
		log.Error(err)
		return
	}

	<-make(chan struct{})
}

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
