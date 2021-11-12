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
	"github.com/hacdias/eagle/v2/contenttype"
	"github.com/hacdias/eagle/v2/entry"
	"github.com/hacdias/eagle/v2/util"
	"github.com/thoas/go-funk"
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
	TemplateNew    string = "new"
	TemplateIndex  string = "index"
	TemplateTags   string = "tags"
	TemplateAuth   string = "auth"
)

func (e *Eagle) includeTemplate(name string, data ...interface{}) (template.HTML, error) {
	var (
		buf bytes.Buffer
		err error
	)

	if len(data) == 1 {
		err = e.templates[name].ExecuteTemplate(&buf, name, data[0])
	} else if len(data) == 2 {
		// TODO(future): perhaps make more type verifications.
		nrd := *data[0].(*RenderData)
		nrd.Entry = data[1].(*entry.Entry)
		nrd.sidecar = nil
		err = e.templates[name].ExecuteTemplate(&buf, name, &nrd)
	} else {
		return "", errors.New("wrong parameters")
	}

	return template.HTML(buf.String()), err
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

func safeCSS(text string) template.CSS {
	return template.CSS(text)
}

func dateFormat(date, template string) string {
	t, err := dateparse.ParseStrict(date)
	if err != nil {
		return date
	}
	return t.Format(template)
}

func (e *Eagle) getTemplateFuncMap(alwaysAbsolute bool) template.FuncMap {
	// TODO(v2): cleanup this
	figure := func(url, alt string) template.HTML {
		var w strings.Builder
		err := writeFigure(&w, e.Config.Site.BaseURL, url, alt, "", alwaysAbsolute, true)
		if err != nil {
			return template.HTML("")
		}
		return template.HTML(w.String())
	}

	funcs := template.FuncMap{
		"truncate":    util.TruncateString,
		"contains":    funk.Contains,
		"domain":      domain,
		"strContains": strings.Contains,
		"safeHTML":    safeHTML,
		"safeCSS":     safeCSS,
		"figure":      figure,
		"dateFormat":  dateFormat,
		"now":         time.Now,
		"include":     e.includeTemplate,
		"md":          e.getRenderMarkdown(alwaysAbsolute),
		"absURL":      e.AbsoluteURL,
		"relURL":      e.relativeURL,
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

func (e *Eagle) updateTemplates() error {
	baseTemplateFilename := path.Join(TemplatesDirectory, TemplateBase+TemplatesExtension)
	baseTemplateData, err := e.fs.ReadFile(baseTemplateFilename)
	if err != nil {
		return err
	}

	fns := e.getTemplateFuncMap(false)

	baseTemplate, err := template.New("base").Funcs(fns).Parse(string(baseTemplateData))
	if err != nil {
		return err
	}

	parsed := map[string]*template.Template{}

	err = e.fs.Walk(TemplatesDirectory, func(filename string, info fs.FileInfo, err error) error {
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

		raw, err := e.fs.ReadFile(filename)
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
		return err
	}

	e.templates = parsed
	return nil
}

type Alternate struct {
	Type string
	Href string
}

type RenderData struct {
	// All pages must have some sort of Entry embedded.
	// This allows us to set generic information about
	// a page that may be needed.
	*entry.Entry

	User config.User
	Site config.Site

	// For page-specific variables.
	Data interface{}

	Alternates   []Alternate
	IsHome       bool
	IsListing    bool
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

	title := rd.Title
	if title == "" {
		title = rd.Summary()
		title = strings.TrimSuffix(title, "…")

		if len(title) > 100 {
			title = util.TruncateString(title, 100) + "…"
		}
	}

	if title != "" {
		return fmt.Sprintf("%s - %s", title, rd.Site.Title)
	}

	return rd.Site.Title
}

func (rd *RenderData) GetSidecar() *Sidecar {
	if rd.sidecar == nil {
		rd.sidecar, _ = rd.eagle.GetSidecar(rd.Entry)
	}
	return rd.sidecar
}

func (rd *RenderData) GetJSON(path string) interface{} {
	filename := filepath.Join(ContentDirectory, rd.ID, path)
	var data interface{}
	_ = rd.eagle.fs.ReadJSON(filename, &data)
	return data
}

func (rd *RenderData) GetFile(path string) string {
	filename := filepath.Join(ContentDirectory, rd.ID, path)
	v, _ := rd.eagle.fs.ReadFile(filename)
	return string(v)
}

func (e *Eagle) Render(w io.Writer, data *RenderData, tpls []string) error {
	data.User = e.Config.User
	data.Site = e.Config.Site
	data.eagle = e

	if e.Config.Development {
		err := e.updateTemplates()
		if err != nil {
			return err
		}
	}

	var tpl *template.Template

	for _, t := range tpls {
		if tt, ok := e.templates[t]; ok {
			tpl = tt
			break
		}
	}

	if tpl == nil {
		return errors.New("unrecognized template")
	}

	mw := e.minifier.Writer(contenttype.HTML, w)
	err := tpl.Execute(mw, data)
	if err != nil {
		return err
	}

	return mw.Close()
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
