package render

import (
	"io"

	"go.hacdias.com/eagle/entry"
)

type Renderer struct {
}

func NewRenderer() *Renderer {
	r := &Renderer{}

	return r
}

func (r *Renderer) Render(w io.Writer, e *entry.Entry) error {

	md := newMarkdown()

	return md.Convert([]byte(e.Content), w)
}
