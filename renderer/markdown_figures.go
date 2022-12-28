package renderer

import (
	"bytes"
	urlpkg "net/url"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
)

// Pandoc syntax: https://pandoc.org/MANUAL.html#images
type figuresRenderer struct {
	*Renderer
	html.Config
	absoluteURLs bool
}

func newFiguresRenderer(renderer *Renderer, absoluteURLs bool) goldmark.Extender {
	e := &figuresRenderer{
		Renderer: renderer,
		Config: html.Config{
			Writer: html.DefaultWriter,
		},
		absoluteURLs: absoluteURLs,
	}
	return e
}

func (r *figuresRenderer) Extend(m goldmark.Markdown) {
	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(r, 100),
	))
}

func (r *figuresRenderer) SetOption(name renderer.OptionName, value interface{}) {
	r.Config.SetOption(name, value)
}

func (r *figuresRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindParagraph, r.renderParagraph)
	reg.Register(ast.KindImage, r.renderImage)
}

func (r *figuresRenderer) renderParagraph(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if isFigure(node) {
		return ast.WalkContinue, nil
	}

	if entering {
		if node.Attributes() != nil {
			_, _ = w.WriteString("<p")
			html.RenderAttributes(w, node, html.ParagraphAttributeFilter)
			_ = w.WriteByte('>')
		} else {
			_, _ = w.WriteString("<p>")
		}
	} else {
		_, _ = w.WriteString("</p>\n")
	}
	return ast.WalkContinue, nil
}

func (r *figuresRenderer) renderImage(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	isFigure := isFigure(node.Parent())

	if !entering {
		if isFigure {
			if node.HasChildren() {
				w.WriteString("</figcaption>")
			}

			_, _ = w.WriteString("</figure>\n")
		}

		return ast.WalkContinue, nil
	}

	n := node.(*ast.Image)

	url, err := urlpkg.Parse(string(n.Destination))
	if err != nil {
		return ast.WalkStop, err
	}
	query := url.Query()

	var (
		class   string
		caption = true
	)

	if v := query.Get("class"); v != "" {
		query.Del("class")
		class = v
	}

	if v := query.Get("caption"); v != "" {
		query.Del("caption")
		caption = v == "true"
	}

	var imgSrc []byte
	if isFigure {
		_, _ = w.WriteString("<figure")
		if class != "" {
			_, _ = w.WriteString(` class="`)
			_, _ = w.WriteString(class)
			_ = w.WriteByte('"')
		}
		_ = w.WriteByte('>')

		_, _ = w.WriteString("<picture>")
		if url.Scheme == "cdn" && r.m != nil {
			id := strings.TrimPrefix(url.Path, "/")
			imgSrc = []byte(r.m.ImageURL(id))

			for format, srcset := range r.m.ImageSourceSet(id) {
				_, _ = w.WriteString("<source srcset=\"")
				_, _ = w.WriteString(srcset)
				_, _ = w.WriteString("\" type=\"image/")
				_, _ = w.WriteString(format)
				_, _ = w.WriteString("\">")
			}
		} else {
			imgSrc = []byte(url.String())
		}
	} else {
		if url.Scheme == "cdn" && r.m != nil {
			id := strings.TrimPrefix(url.Path, "/")
			imgSrc = []byte(r.m.ImageURL(id))
		} else {
			imgSrc = []byte(url.String())
		}
	}

	w.WriteString("<img src=\"")
	if r.absoluteURLs && r.c.Server.BaseURL != "" && bytes.HasPrefix(imgSrc, []byte("/")) {
		_, _ = w.Write(util.EscapeHTML([]byte(r.c.Server.BaseURL)))
	}
	if r.Unsafe || !html.IsDangerousURL(imgSrc) {
		_, _ = w.Write(util.EscapeHTML(imgSrc))
	}
	w.WriteString(`" alt="`)
	w.Write(n.Text(source))
	w.WriteByte('"')
	if !isFigure && class != "" {
		_, _ = w.WriteString(` class="`)
		_, _ = w.WriteString(class)
		_ = w.WriteByte('"')
	}
	if n.Title != nil {
		w.WriteString(` title="`)
		r.Writer.Write(w, n.Title)
		w.WriteByte('"')
	}
	if n.Attributes() != nil {
		html.RenderAttributes(w, n, html.ImageAttributeFilter)
	}
	_, _ = w.WriteString(" loading=\"lazy\">")

	if isFigure {
		_, _ = w.WriteString("</picture>")

		// Special case for figure, to render the caption.
		if node.HasChildren() && caption {
			w.WriteString("\n<figcaption>")
			return ast.WalkContinue, nil
		}

		_, _ = w.WriteString("</figure>")
	}

	return ast.WalkSkipChildren, nil
}

func isFigure(node ast.Node) bool {
	var child = node.FirstChild()
	return node.Kind() == ast.KindParagraph &&
		child != nil &&
		child == node.LastChild() &&
		child.Kind() == ast.KindImage
	// child.HasChildren()
}
