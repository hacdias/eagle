package renderer

import (
	"errors"
	"html/template"
	"io"
	"path"
	"path/filepath"

	"github.com/hacdias/eagle/v4/eagle"
	"github.com/hacdias/eagle/v4/fs"
	"github.com/hacdias/eagle/v4/pkg/contenttype"
	"github.com/tdewolff/minify/v2"
	"github.com/yuin/goldmark"
)

type Renderer struct {
	c                *eagle.Config
	fs               *fs.FS
	mediaBaseURL     string
	minify           *minify.M
	markdown         goldmark.Markdown
	absoluteMarkdown goldmark.Markdown
	templates        map[string]*template.Template
	assets           *Assets
}

func NewRenderer(c *eagle.Config, fs *fs.FS, mediaBaseURL string) (*Renderer, error) {
	r := &Renderer{
		c:            c,
		fs:           fs,
		mediaBaseURL: mediaBaseURL,

		templates: map[string]*template.Template{},
		minify:    newMinify(),
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
	data.Me = r.c.User
	data.Site = r.c.Site
	data.Assets = r.assets
	data.fs = r.fs

	if r.c.Development {
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
	*eagle.Entry

	Assets *Assets
	Me     eagle.User
	Site   eagle.Site

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

	fs      *fs.FS
	sidecar *eagle.Sidecar
}

func (rd *RenderData) GetSidecar() *eagle.Sidecar {
	if rd.sidecar == nil {
		rd.sidecar, _ = rd.fs.GetSidecar(rd.Entry)
	}
	return rd.sidecar
}

func (rd *RenderData) GetJSON(path string) interface{} {
	filename := filepath.Join(fs.ContentDirectory, rd.ID, path)
	var data interface{}
	_ = rd.fs.ReadJSON(filename, &data)
	return data
}

func (rd *RenderData) GetFile(path string) string {
	filename := filepath.Join(fs.ContentDirectory, rd.ID, path)
	v, _ := rd.fs.ReadFile(filename)
	return string(v)
}

func (rd *RenderData) GetEntry(id string) *eagle.Entry {
	entry, _ := rd.fs.GetEntry(id)
	return entry
}

func (rd *RenderData) HasFile(path string) bool {
	filename := filepath.Join(fs.ContentDirectory, rd.ID, path)
	stat, err := rd.fs.Stat(filename)
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
