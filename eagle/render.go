package eagle

import (
	"bytes"
	"errors"
	"html/template"
	"io"
	"io/fs"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/hacdias/eagle/v2/config"
)

type readDirFileFS interface {
	fs.ReadDirFS
	fs.ReadFileFS
}

const (
	TemplatesExtension string = ".html"
	TemplatesDirectory string = "templates"

	TemplateBase string = "base"
)

func (e *Eagle) GetTemplates() (map[string]*template.Template, error) {
	// TODO: cache templates

	baseTemplateFilename := path.Join(TemplatesDirectory, TemplateBase+TemplatesExtension)
	baseTemplateData, err := e.SrcFs.ReadFile(baseTemplateFilename)
	if err != nil {
		return nil, err
	}

	fns := template.FuncMap{
		"include": e.includeTemplate,
		"now":     time.Now,
	}

	baseTemplate, err := template.New("base").Funcs(fns).Parse(string(baseTemplateData))
	if err != nil {
		return nil, err
	}

	files, err := e.SrcFs.ReadDir(TemplatesDirectory)
	if err != nil {
		return nil, err
	}

	parsed := map[string]*template.Template{}
	for _, info := range files {
		if info.IsDir() {
			continue
		}

		basename := filepath.Base(info.Name())
		ext := filepath.Ext(basename)
		id := strings.TrimSuffix(basename, ext)

		if ext != TemplatesExtension || id == TemplateBase {
			continue
		}

		raw, err := e.SrcFs.ReadFile(filepath.Join(TemplatesDirectory, info.Name()))
		if err != nil {
			return nil, err
		}

		parsed[id], err = template.Must(baseTemplate.Clone()).New(id).Funcs(fns).Parse(string(raw))
		if err != nil {
			return nil, err
		}
	}

	return parsed, err
}

type RenderData struct {
	User *config.User
	Site *config.Site
	Data interface{}
}

func (e *Eagle) Render(w io.Writer, data *RenderData, tpls []string) error {
	// TODO: fill data

	data.User = &e.Config.Author
	data.Site = &e.Config.Site

	templates, err := e.GetTemplates()
	if err != nil {
		return err
	}

	var template *template.Template

	for _, tpl := range tpls {
		if t, ok := templates[tpl]; ok {
			template = t
			break
		}
	}

	if template == nil {
		return errors.New("unrecognized template")
	}

	return template.Execute(w, data)
}

func (e *Eagle) includeTemplate(name string, data interface{}) (template.HTML, error) {
	templates, err := e.GetTemplates()
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = templates[name].ExecuteTemplate(&buf, name, data)
	return template.HTML(buf.String()), err
}
