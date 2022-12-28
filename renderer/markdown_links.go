package renderer

import (
	"bytes"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
)

type linksRenderer struct {
	*Renderer
	html.Config
	absoluteURLs bool
}

func newLinksRenderer(renderer *Renderer, absoluteURLs bool) goldmark.Extender {
	e := &linksRenderer{
		Renderer: renderer,
		Config: html.Config{
			Writer: html.DefaultWriter,
		},
		absoluteURLs: absoluteURLs,
	}
	return e
}

func (md *linksRenderer) Extend(m goldmark.Markdown) {
	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(md, 0),
	))
}

func (r *linksRenderer) SetOption(name renderer.OptionName, value interface{}) {
	r.Config.SetOption(name, value)
}

func (r *linksRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindLink, r.renderLink)
	reg.Register(ast.KindAutoLink, r.renderAutoLink)
}

func (r *linksRenderer) renderLink(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.Link)
	if entering {
		_, _ = w.WriteString("<a href=\"")
		destination := util.URLEscape(n.Destination, true)
		if r.absoluteURLs && r.c.Server.BaseURL != "" && bytes.HasPrefix(destination, []byte("/")) {
			_, _ = w.Write(util.EscapeHTML([]byte(r.c.Server.BaseURL)))
		}
		if r.Unsafe || !html.IsDangerousURL(destination) {
			_, _ = w.Write(util.EscapeHTML(destination))
		}
		_ = w.WriteByte('"')
		if n.Title != nil {
			_, _ = w.WriteString(` title="`)
			r.Writer.Write(w, n.Title)
			_ = w.WriteByte('"')
		}
		if !(bytes.HasPrefix(destination, []byte("/")) || bytes.HasPrefix(destination, []byte("#"))) {
			_, _ = w.WriteString(` rel="noopener noreferrer" `)
		}
		_ = w.WriteByte('>')
	} else {
		_, _ = w.WriteString("</a>")
	}
	return ast.WalkContinue, nil
}

func (r *linksRenderer) renderAutoLink(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.AutoLink)
	if !entering {
		return ast.WalkContinue, nil
	}
	_, _ = w.WriteString(`<a href="`)
	url := n.URL(source)
	label := n.Label(source)
	if n.AutoLinkType == ast.AutoLinkEmail && !bytes.HasPrefix(bytes.ToLower(url), []byte("mailto:")) {
		_, _ = w.WriteString("mailto:")
	}
	destination := util.URLEscape(url, false)
	if r.absoluteURLs && r.c.Server.BaseURL != "" && bytes.HasPrefix(destination, []byte("/")) {
		_, _ = w.Write(util.EscapeHTML([]byte(r.c.Server.BaseURL)))
	}
	_, _ = w.Write(util.EscapeHTML(destination))
	if n.Attributes() != nil {
		_ = w.WriteByte('"')
		html.RenderAttributes(w, n, html.LinkAttributeFilter)
	} else {
		_, _ = w.WriteString(`"`)
	}

	if n.AutoLinkType == ast.AutoLinkURL && !(bytes.HasPrefix(url, []byte("/")) || bytes.HasPrefix(destination, []byte("#"))) {
		_, _ = w.WriteString(` rel="noopener noreferrer" `)
	}

	_ = w.WriteByte('>')
	_, _ = w.Write(util.EscapeHTML(label))
	_, _ = w.WriteString(`</a>`)
	return ast.WalkContinue, nil
}
