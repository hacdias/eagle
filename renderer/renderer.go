package renderer

import (
	"errors"
	"html/template"
	"io"
	"path"
	"path/filepath"

	"github.com/hacdias/eagle/v4/config"
	"github.com/hacdias/eagle/v4/eagle"
	"github.com/hacdias/eagle/v4/entry"
	"github.com/hacdias/eagle/v4/pkg/contenttype"
	"github.com/tdewolff/minify/v2"
	"github.com/yuin/goldmark"
)

type Renderer struct {
	c                *config.Config
	mediaBaseURL     string
	eagle            *eagle.Eagle //wip: remove/fs
	assets           *Assets
	templates        map[string]*template.Template
	markdown         goldmark.Markdown
	absoluteMarkdown goldmark.Markdown
	minify           *minify.M
}

func NewRenderer(c *config.Config, e *eagle.Eagle) (*Renderer, error) {
	r := &Renderer{
		c:         c,
		eagle:     e,
		templates: map[string]*template.Template{},
		minify:    getMinify(),
	}

	if c.BunnyCDN != nil {
		r.mediaBaseURL = c.BunnyCDN.Base // wip: change this
	}

	r.markdown = newMarkdown(r, false)
	r.absoluteMarkdown = newMarkdown(r, true)

	err := r.initAssets()
	if err != nil {
		return nil, err
	}

	err = r.initTemplates()
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (r *Renderer) Render(w io.Writer, data *RenderData, templates []string) error {
	data.Me = r.eagle.Config.User
	data.Site = r.eagle.Config.Site
	data.Assets = r.assets
	data.eagle = r.eagle

	if r.eagle.Config.Development {
		// Probably not very concurrent safe. But it's just
		// for development purposes.
		err := r.initAssets()
		if err != nil {
			return err
		}

		err = r.initTemplates()
		if err != nil {
			return err
		}
	}

	var tpl *template.Template

	for _, t := range templates {
		if tt, ok := r.templates[t]; ok {
			tpl = tt
			break
		}
	}

	if tpl == nil {
		return errors.New("unrecognized template")
	}

	mw := r.minify.Writer(contenttype.HTML, w)
	err := tpl.Execute(mw, data)
	if err != nil {
		return err
	}

	return mw.Close()
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

	Assets *Assets
	Me     config.User
	Site   config.Site

	// For page-specific variables.
	Data interface{}

	Alternates   []Alternate
	User         string
	IsHome       bool
	IsLoggedIn   bool
	IsAdmin      bool
	NoIndex      bool
	TorUsed      bool
	OnionAddress string

	eagle   *eagle.Eagle
	sidecar *eagle.Sidecar
}

func (rd *RenderData) GetSidecar() *eagle.Sidecar {
	if rd.sidecar == nil {
		rd.sidecar, _ = rd.eagle.GetSidecar(rd.Entry)
	}
	return rd.sidecar
}

func (rd *RenderData) GetJSON(path string) interface{} {
	filename := filepath.Join(eagle.ContentDirectory, rd.ID, path)
	var data interface{}
	_ = rd.eagle.FS.ReadJSON(filename, &data)
	return data
}

func (rd *RenderData) GetFile(path string) string {
	filename := filepath.Join(eagle.ContentDirectory, rd.ID, path)
	v, _ := rd.eagle.FS.ReadFile(filename)
	return string(v)
}

func (rd *RenderData) GetEntry(id string) *entry.Entry {
	entry, _ := rd.eagle.GetEntry(id)
	return entry
}

func (rd *RenderData) HasFile(path string) bool {
	filename := filepath.Join(eagle.ContentDirectory, rd.ID, path)
	stat, err := rd.eagle.FS.Stat(filename)
	return err == nil && stat.Mode().IsRegular()
}

func (rd *RenderData) TryFiles(filenames ...string) string {
	for _, filename := range filenames {
		if rd.HasFile(filename) {
			return path.Join(rd.ID, filename)
		}
	}

	return ""
}
