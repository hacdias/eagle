package render

import (
	"context"
	"fmt"
	"html/template"
	"io"

	"go.hacdias.com/eagle/config"
	"go.hacdias.com/eagle/entry"
	"go.hacdias.com/eagle/render/templates"
)

type Renderer struct {
	cfg       *config.Config
	assets    *assetsBuilder
	templates *templatesBuilder
}

func NewRenderer(cfg *config.Config) (*Renderer, error) {
	r := &Renderer{
		cfg:       cfg,
		assets:    newAssetsBuilder(cfg.Server.Source, cfg.Site.Assets),
		templates: newTemplatesBuilder(cfg.Server.Source),
	}

	err := r.assets.build()
	if err != nil {
		return nil, err
	}

	err = r.templates.load(nil)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (r *Renderer) Render(w io.Writer, data Data, tpls []string) error {
	err := templates.Single(&r.cfg.Site, data.Entry).Render(context.Background(), w)
	return err

	var layout *template.Template
	for _, layoutName := range tpls {
		if tpl, ok := r.templates.layouts[layoutName]; ok {
			layout = tpl
			break
		}
	}
	if layout == nil {
		return fmt.Errorf("template %v not found", tpls)
	}

	data.r = r
	data.Site = r.cfg.Site

	// layout.Execute(w, e)

	// md := newMarkdown()

	// return md.Convert([]byte(e.Content), w)

	return layout.Execute(w, data)
}

// type Alternate struct {
// 	Type string
// 	Href string
// }

type Data struct {
	r *Renderer

	// All pages must have some sort of Entry embedded.
	// This allows us to set generic information about
	// a page that may be needed.
	*entry.Entry

	Site config.SiteConfig

	// Assets *Assets
	// Me     eagle.User

	// // For page-specific variables.
	// Data interface{}

	// Alternates []Alternate
	// IsHome     bool
	// IsLoggedIn bool
	// NoIndex    bool

	// fs *fs.FS
}

func (d Data) AssetByName(name string) *Asset {
	return d.r.AssetByName(name)
}
