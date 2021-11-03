package eagle

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
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
	TemplateError  string = "error"
	TemplateLogin  string = "login"
	TemplateSearch string = "search"
	TemplateEditor string = "editor"
	TemplateIndex  string = "index"
)

func (e *Eagle) includeTemplate(name string, data ...interface{}) (template.HTML, error) {
	templates, err := e.getTemplates()
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer

	if len(data) == 1 {
		err = templates[name].ExecuteTemplate(&buf, name, data[0])
	} else if len(data) == 2 {
		// TODO: maybe some type verifications.
		nrd := *data[0].(*RenderData)
		nrd.Entry = data[1].(*Entry)
		err = templates[name].ExecuteTemplate(&buf, name, nrd)
	} else {
		return "", errors.New("wrong parameters")
	}

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
	if e.templates != nil {
		return e.templates, nil
	}

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

	parsed := map[string]*template.Template{}

	err = e.SrcFs.Walk(TemplatesDirectory, func(filename string, info fs.FileInfo, err error) error {
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

		raw, err := e.SrcFs.ReadFile(filename)
		if err != nil {
			return err
		}

		parsed[id], err = template.Must(baseTemplate.Clone()).New(id).Funcs(fns).Parse(string(raw))
		return err
	})

	if err != nil {
		return nil, err
	}

	if !e.Config.Development {
		e.templates = parsed
	}

	return parsed, nil
}

type RenderData struct {
	// All pages must have some sort of Entry embedded.
	// This allows us to set generic information about
	// a page that may be needed.
	*Entry

	User *config.User
	Site *config.Site

	Entries []*Entry

	SearchQuery string
	NextPage    string
	IsListing   bool

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

	data.User = e.Config.User
	data.Site = e.Config.Site

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
