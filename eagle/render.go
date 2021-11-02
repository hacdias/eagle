package eagle

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/hacdias/eagle/v2/config"
)

const (
	TemplatesExtension string = ".html"
	TemplatesDirectory string = "templates"

	TemplateBase   string = "base"
	TemplateSingle string = "single"
	TemplateList   string = "list"
)

func (e *Eagle) includeTemplate(name string, data interface{}) (template.HTML, error) {
	templates, err := e.getTemplates()
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = templates[name].ExecuteTemplate(&buf, name, data)
	return template.HTML(buf.String()), err
}

func (e *Eagle) getTemplateFuncMap() template.FuncMap {
	return template.FuncMap{
		"include": e.includeTemplate,
		"now":     time.Now,
		"md":      e.safeRenderMarkdownAsHTML,
	}
}

func (e *Eagle) getTemplates() (map[string]*template.Template, error) {
	// TODO: cache templates

	baseTemplateFilename := path.Join(TemplatesDirectory, TemplateBase+TemplatesExtension)
	baseTemplateData, err := e.SrcFs.ReadFile(baseTemplateFilename)
	if err != nil {
		return nil, err
	}

	fns := e.getTemplateFuncMap()

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

// Title          string    `yaml:"title,omitempty"`
// Description    string    `yaml:"description,omitempty"`
// Draft          bool      `yaml:"draft,omitempty"`
// Deleted        bool      `yaml:"deleted,omitempty"`
// Private        bool      `yaml:"private,omitempty"`
// NoInteractions bool      `yaml:"noInteractions,omitempty"`
// Emoji          string    `yaml:"emoji,omitempty"`
// Published      time.Time `yaml:"published,omitempty"`
// Updated        time.Time `yaml:"updated,omitempty"`
// Section        string    `yaml:"section,omitempty"`

type RenderData struct {
	// All pages must have some sort of Entry embedded.
	// This allows us to set generic information about
	// a page that may be needed.
	*Entry

	User *config.User
	Site *config.Site

	Entries []*Entry

	RenderedContent template.HTML
	// Data interface{}

	IsHome       bool
	LoggedIn     bool
	TorUsed      bool
	OnionAddress string
}

func (rd *RenderData) HeadTitle() string {
	if rd.ID == "/" {
		return rd.Site.Title
	}

	if rd.Title != "" {
		return fmt.Sprintf("%s - %s", rd.Title, rd.Site.Title)
	}

	return rd.Site.Title
}

func (e *Eagle) Render(w io.Writer, data *RenderData, tpls []string) error {
	// TODO: fill data

	data.User = &e.Config.Author
	data.Site = &e.Config.Site

	templates, err := e.getTemplates()
	if err != nil {
		return err
	}

	var tpl *template.Template

	for _, t := range tpls {
		if tt, ok := templates[t]; ok {
			tpl = tt
			break
		}
	}

	if tpl == nil {
		return errors.New("unrecognized template")
	}

	return tpl.Execute(w, data)
}

func (e *Eagle) renderMarkdown(source string) ([]byte, error) {
	var buffer bytes.Buffer
	err := e.markdown.Convert([]byte(source), &buffer)
	return buffer.Bytes(), err
}

func (e *Eagle) renderMarkdownAsHTML(source string) (rendered template.HTML, err error) {
	b, err := e.renderMarkdown(source)
	if err != nil {
		return "", err
	}
	return template.HTML(b), nil
}

func (e *Eagle) safeRenderMarkdownAsHTML(source string) template.HTML {
	h, _ := e.renderMarkdownAsHTML(source)
	return h
}
