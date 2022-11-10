package renderer

import (
	"html/template"
	"io/fs"
	"path"
	"path/filepath"
	"strings"

	"github.com/hacdias/eagle/eagle"
)

const (
	TemplatesExtension string = ".html"
	TemplatesDirectory string = "templates"

	TemplateBase      string = "base"
	TemplateSingle    string = "single"
	TemplateFeed      string = "feed"
	TemplateList      string = "list"
	TemplateError     string = "error"
	TemplateLogin     string = "login"
	TemplateSearch    string = "search"
	TemplateEditor    string = "editor"
	TemplateNew       string = "new"
	TemplateIndex     string = "index"
	TemplateTags      string = "tags"
	TemplateEmojis    string = "emojis"
	TemplateAuth      string = "auth"
	TemplateDashboard string = "dashboard"
)

func EntryTemplates(e *eagle.Entry) []string {
	t := []string{}
	if e.Template != "" {
		t = append(t, e.Template)
	}
	t = append(t, TemplateSingle)
	return t
}

func (r *Renderer) LoadTemplates() error {
	baseTemplateFilename := path.Join(TemplatesDirectory, TemplateBase+TemplatesExtension)
	baseTemplateData, err := r.fs.ReadFile(baseTemplateFilename)
	if err != nil {
		return err
	}

	fns := r.getTemplateFuncMap(false)

	baseTemplate, err := template.New("base").Funcs(fns).Parse(string(baseTemplateData))
	if err != nil {
		return err
	}

	parsed := map[string]*template.Template{}

	err = r.fs.Walk(TemplatesDirectory, func(filename string, info fs.FileInfo, err error) error {
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

		raw, err := r.fs.ReadFile(filename)
		if err != nil {
			return err
		}

		if id == TemplateFeed {
			absFns := r.getTemplateFuncMap(true)
			parsed[id], err = template.Must(baseTemplate.Clone()).New(id).Funcs(absFns).Parse(string(raw))
		} else {
			parsed[id], err = template.Must(baseTemplate.Clone()).New(id).Funcs(fns).Parse(string(raw))
		}

		return err
	})

	if err != nil {
		return err
	}

	r.templates = parsed
	return nil
}
