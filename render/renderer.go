package render

import (
	"bytes"
	"fmt"
	"html/template"
	"io"

	"github.com/yuin/goldmark"
	"go.hacdias.com/eagle/config"
)

type Renderer struct {
	cfg       *config.Config
	assets    Assets
	templates map[string]*template.Template
	markdown  goldmark.Markdown
}

func NewRenderer(cfg *config.Config) (*Renderer, error) {
	r := &Renderer{
		cfg:      cfg,
		markdown: newMarkdown(),
	}

	var err error

	assetsBuilder := newAssetsBuilder(cfg.Server.Source)
	r.assets, err = assetsBuilder.build(cfg.Site.Assets)
	if err != nil {
		return nil, err
	}

	funcMap := r.getTemplateFuncMap(true)
	templatesBuilder := newTemplatesBuilder(cfg.Server.Source)
	r.templates, err = templatesBuilder.load(funcMap)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (r *Renderer) Render(w io.Writer, data Pagina, tpls []string) error {
	var layout *template.Template
	for _, layoutName := range tpls {
		if tpl, ok := r.templates[layoutName]; ok {
			layout = tpl
			break
		}
	}
	if layout == nil {
		return fmt.Errorf("template %v not found", tpls)
	}

	var b bytes.Buffer
	err := r.markdown.Convert([]byte(data.Entry.Content), &b)
	if err != nil {
		return err
	}

	data.Site = r.cfg.Site
	data.Assets = r.assets
	data.Content = template.HTML(b.String())

	return layout.Execute(w, data)
}

func (r *Renderer) Assets() Assets {
	return r.assets
}
