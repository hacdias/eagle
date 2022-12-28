package renderer

import (
	"bytes"
	"errors"
	"html/template"
	"io"
	osfs "io/fs"
	"path"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/hacdias/eagle/eagle"
	"github.com/hacdias/eagle/fs"
	"github.com/hacdias/eagle/log"
	"github.com/hacdias/eagle/pkg/contenttype"
	"github.com/tdewolff/minify/v2"
	"github.com/yuin/goldmark"
)

type Renderer struct {
	c                 *eagle.Config
	fs                *fs.FS
	mediaBaseURL      string
	minify            *minify.M
	markdown          goldmark.Markdown
	absoluteMarkdown  goldmark.Markdown
	templates         map[string]*template.Template
	absoluteTemplates map[string]*template.Template
	assets            *Assets
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

	err := r.LoadAssets()
	if err != nil {
		return nil, err
	}

	err = r.LoadTemplates()
	if err != nil {
		return nil, err
	}

	if c.Development {
		go r.watch(TemplatesDirectory, r.LoadTemplates)
		go r.watch(AssetsDirectory, r.LoadAssets)
	}

	return r, nil
}

func (r *Renderer) watch(dir string, exec func() error) {
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

	err = r.fs.Walk(dir, func(filename string, info osfs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			return nil
		}

		return watcher.Add(filepath.Join(r.c.Source.Directory, filename))
	})
	if err != nil {
		log.Error(err)
		return
	}

	<-make(chan struct{})
}

func (r *Renderer) Render(w io.Writer, data *RenderData, templates []string, absoluteURLs bool) error {
	data.Me = r.c.User
	data.Site = r.c.Site
	data.Assets = r.assets
	data.fs = r.fs

	var htmlTemplates map[string]*template.Template
	if absoluteURLs {
		htmlTemplates = r.absoluteTemplates
	} else {
		htmlTemplates = r.templates
	}

	var tpl *template.Template

	for _, t := range templates {
		if tt, ok := htmlTemplates[t]; ok {
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

func (r *Renderer) RenderAbsoluteMarkdown(source string) template.HTML {
	var buffer bytes.Buffer
	_ = r.absoluteMarkdown.Convert([]byte(source), &buffer)
	return template.HTML(buffer.Bytes())
}

func (r *Renderer) RenderRelativeMarkdown(source string) template.HTML {
	var buffer bytes.Buffer
	_ = r.markdown.Convert([]byte(source), &buffer)
	return template.HTML(buffer.Bytes())
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

	Alternates []Alternate
	IsHome     bool
	IsLoggedIn bool
	NoIndex    bool

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

func (rd *RenderData) GetYAML(path string) interface{} {
	filename := filepath.Join(fs.ContentDirectory, rd.ID, path)
	var data interface{}
	_ = rd.fs.ReadYAML(filename, &data)
	return data
}

func (rd *RenderData) GetLogs(path string) interface{} {
	filename := filepath.Join(fs.ContentDirectory, rd.ID, path)
	var data eagle.Logs
	switch filepath.Ext(path) {
	case ".json":
		_ = rd.fs.ReadJSON(filename, &data)
	case ".yaml", ".yml":
		_ = rd.fs.ReadYAML(filename, &data)
	}
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
