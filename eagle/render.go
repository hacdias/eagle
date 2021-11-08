package eagle

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	urlpkg "net/url"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/araddon/dateparse"
	"github.com/hacdias/eagle/v2/config"
)

const (
	TemplatesExtension string = ".html"
	TemplatesDirectory string = "templates"

	TemplateBase   string = "base"
	TemplateSingle string = "single"
	TemplateFeed   string = "feed"
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
		// TODO(future): perhaps make more type verifications.
		nrd := *data[0].(*RenderData)
		nrd.Entry = data[1].(*Entry)
		nrd.sidecar = nil
		err = templates[name].ExecuteTemplate(&buf, name, &nrd)
	} else {
		return "", errors.New("wrong parameters")
	}

	return template.HTML(buf.String()), err
}

func truncate(text string, size int) string {
	if len(text) <= size {
		return text
	}

	return strings.TrimSpace(text[:size]) + "..."
}

func domain(text string) string {
	u, err := urlpkg.Parse(text)
	if err != nil {
		return text
	}

	return u.Host
}

func safeHTML(text string) template.HTML {
	return template.HTML(text)
}

func dateFormat(date, template string) string {
	t, err := dateparse.ParseStrict(date)
	if err != nil {
		return date
	}
	return t.Format(template)
}

func (e *Eagle) getTemplateFuncMap(alwaysAbsolute bool) template.FuncMap {
	funcs := template.FuncMap{
		"include":    e.includeTemplate,
		"now":        time.Now,
		"md":         e.getRenderMarkdown(alwaysAbsolute),
		"truncate":   truncate,
		"domain":     domain,
		"safeHTML":   safeHTML,
		"dateFormat": dateFormat,
		"absURL":     e.AbsoluteURL,
		"relURL":     e.relativeURL,
	}

	if alwaysAbsolute {
		funcs["relURL"] = e.AbsoluteURL
	}

	return funcs
}

func (e *Eagle) AbsoluteURL(path string) string {
	url, _ := urlpkg.Parse(path)
	base, _ := urlpkg.Parse(e.Config.Site.BaseURL)
	return base.ResolveReference(url).String()
}

func (e *Eagle) relativeURL(path string) string {
	url, _ := urlpkg.Parse(path)
	base, _ := urlpkg.Parse(e.Config.Site.BaseURL)
	return base.ResolveReference(url).Path
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

	fns := e.getTemplateFuncMap(false)

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

		if id == TemplateFeed {
			absFns := e.getTemplateFuncMap(true)
			parsed[id], err = template.Must(baseTemplate.Clone()).New(id).Funcs(absFns).Parse(string(raw))
		} else {
			parsed[id], err = template.Must(baseTemplate.Clone()).New(id).Funcs(fns).Parse(string(raw))
		}

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
	Page        int
	NextPage    string
	IsListing   bool

	IsHome       bool
	LoggedIn     bool
	TorUsed      bool
	OnionAddress string

	eagle   *Eagle
	sidecar *Sidecar
}

func (rd *RenderData) HeadTitle() string {
	if rd.ID == "/" {
		return rd.Site.Title
	}

	// TODO(v2): create entry.Title() that gives the entry title based on the
	// content.

	if rd.Title != "" {
		return fmt.Sprintf("%s - %s", rd.Title, rd.Site.Title)
	}

	return rd.Site.Title
}

func (rd *RenderData) GetSidecar() *Sidecar {
	if rd.sidecar == nil {
		rd.sidecar, _ = rd.eagle.GetSidecar(rd.Entry)
	}
	return rd.sidecar
}

func (e *Eagle) Render(w io.Writer, data *RenderData, tpls []string) error {
	data.User = e.Config.User
	data.Site = e.Config.Site
	data.eagle = e

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

func (e *Eagle) getRenderMarkdown(absoluteURLs bool) func(string) template.HTML {
	if absoluteURLs {
		return func(source string) template.HTML {
			var buffer bytes.Buffer
			_ = e.absoluteMarkdown.Convert([]byte(source), &buffer)
			return template.HTML(buffer.Bytes())
		}
	} else {
		return func(source string) template.HTML {
			var buffer bytes.Buffer
			_ = e.markdown.Convert([]byte(source), &buffer)
			return template.HTML(buffer.Bytes())
		}
	}
}
