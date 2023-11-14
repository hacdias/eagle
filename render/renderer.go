package render

import (
	"io"

	"go.hacdias.com/eagle/config"
	"go.hacdias.com/eagle/entry"
)

type Renderer struct {
	cfg    *config.Config
	assets *assetsBuilder
}

func NewRenderer(cfg *config.Config) (*Renderer, error) {
	r := &Renderer{
		cfg:    cfg,
		assets: newAssetsBuilder(cfg.Server.Source, cfg.Site.Assets),
	}

	err := r.assets.build()

	return r, err
}

func (r *Renderer) Render(w io.Writer, e *entry.Entry) error {

	md := newMarkdown()

	return md.Convert([]byte(e.Content), w)
}
