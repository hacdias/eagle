package eagle

import (
	"bytes"
	"io"
	urlpkg "net/url"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
)

var defaultGoldmarkOptions = []goldmark.Option{
	goldmark.WithRendererOptions(
		html.WithUnsafe(),
	),
	goldmark.WithParserOptions(
		parser.WithAutoHeadingID(),
		parser.WithAttribute(),
	),
	goldmark.WithExtensions(
		extension.Table,
		extension.Strikethrough,
		extension.Footnote,
		extension.Typographer,
		extension.Linkify,
		extension.TaskList,
	),
}

func newMarkdown(absURLs bool, baseURL string) goldmark.Markdown {
	return goldmark.New(append(defaultGoldmarkOptions, goldmark.WithExtensions(
		&customMarkdown{
			absURLs: absURLs,
			baseURL: baseURL,
		},
	))...)
}

type customMarkdown struct {
	baseURL string
	absURLs bool
}

func (c *customMarkdown) Extend(m goldmark.Markdown) {
	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(newCustomRenderer(c), 100),
	))
}

func newCustomRenderer(l *customMarkdown) renderer.NodeRenderer {
	r := &customRenderer{
		Config: html.Config{
			Writer: html.DefaultWriter,
		},
		baseURL: l.baseURL,
		absURLs: l.absURLs,
	}
	return r
}

type customRenderer struct {
	html.Config
	baseURL string
	absURLs bool
}

func (r *customRenderer) SetOption(name renderer.OptionName, value interface{}) {
	r.Config.SetOption(name, value)
}

// RegisterFuncs implements NodeRenderer.RegisterFuncs.
func (r *customRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindLink, r.renderLink)
	reg.Register(ast.KindImage, r.renderImage)
	reg.Register(ast.KindAutoLink, r.renderAutoLink)
}

// https://github.com/yuin/goldmark/blob/5588d92a56fe1642791cf4aa8e9eae8227cfeecd/renderer/html/html.go#L439

func (r *customRenderer) renderLink(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.Link)
	if entering {
		_, _ = w.WriteString("<a href=\"")
		destination := util.URLEscape(n.Destination, true)
		if r.absURLs && r.baseURL != "" && bytes.HasPrefix(destination, []byte("/")) {
			_, _ = w.Write(util.EscapeHTML([]byte(r.baseURL)))
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
		if !bytes.HasPrefix(destination, []byte("/")) {
			_, _ = w.WriteString(` rel="noopener noreferrer" target="_blank" `)
		}
		_ = w.WriteByte('>')
	} else {
		_, _ = w.WriteString("</a>")
	}
	return ast.WalkContinue, nil
}

func (r *customRenderer) renderAutoLink(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
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
	if r.absURLs && r.baseURL != "" && bytes.HasPrefix(destination, []byte("/")) {
		_, _ = w.Write(util.EscapeHTML([]byte(r.baseURL)))
	}
	_, _ = w.Write(util.EscapeHTML(destination))
	if n.Attributes() != nil {
		_ = w.WriteByte('"')
		html.RenderAttributes(w, n, html.LinkAttributeFilter)
	} else {
		_, _ = w.WriteString(`"`)
	}

	if n.AutoLinkType == ast.AutoLinkURL && !bytes.HasPrefix(url, []byte("/")) {
		_, _ = w.WriteString(` rel="noopener noreferrer" target="_blank" `)
	}

	_ = w.WriteByte('>')
	_, _ = w.Write(util.EscapeHTML(label))
	_, _ = w.WriteString(`</a>`)
	return ast.WalkContinue, nil
}

// Hijack the image rendering and output <figure>!
//
// Syntax
//	![Alt text](url "Title")
//	url?class=my+class									--> Add class.
//	url?id=someid												--> Add id.
//	url?caption=false							  		--> Do not print "Title" as <figcaption>.
//
// URL should be either:
//	- cdn:/slug-at-cdn									--> Renders <figure> with many <source>.
// 	- /relative/to/image.jpeg						--> Renders an <img> by default.
//	- http://example.com/example.jpg		-->	Renders an <img> by default.
func (r *customRenderer) renderImage(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	n := node.(*ast.Image)

	err := writeFigure(w, r.baseURL, string(n.Destination), string(n.Text(source)), string(n.Title), r.absURLs, r.Unsafe, false)
	if err != nil {
		return ast.WalkStop, err
	}

	return ast.WalkSkipChildren, nil
}

type figureWriter interface {
	io.Writer
	WriteByte(c byte) error
	WriteString(s string) (int, error)
	WriteRune(r rune) (size int, err error)
}

// TODO(v2): cleanup this
func writeFigure(w figureWriter, baseURL, imgURL, alt, title string, absURLs, unsafe, uPhoto bool) error {
	url, err := urlpkg.Parse(imgURL)
	if err != nil {
		return err
	}

	query := url.Query()

	_, _ = w.WriteString("<figure")

	if class := query.Get("class"); class != "" {
		query.Del("class")
		_, _ = w.WriteString(" class=\"")
		_, _ = w.WriteString(class)
		_ = w.WriteByte('"')
	}

	if id := query.Get("id"); id != "" {
		query.Del("id")
		_, _ = w.WriteString(" id=\"")
		_, _ = w.WriteString(id)
		_ = w.WriteByte('"')
	}

	caption := true
	if c := query.Get("caption"); c != "" {
		caption = c == "true"
		query.Del("caption")
	}

	_ = w.WriteByte('>')

	url.RawQuery = query.Encode()

	var imgSrc []byte

	_, _ = w.WriteString("<picture>")

	if url.Scheme == "cdn" {
		id := strings.TrimPrefix(url.Path, "/")
		imgSrc = []byte("https://cdn.hacdias.com/i/t/" + id + "-2000x.jpeg")

		_, _ = w.WriteString("<source srcset=\"")
		_, _ = w.WriteString(makePictureSourceSet(id, "webp"))
		_, _ = w.WriteString("\" type=\"image/webp\">")

		_, _ = w.WriteString("<source srcset=\"")
		_, _ = w.WriteString(makePictureSourceSet(id, "jpeg"))
		_, _ = w.WriteString("\">")
	} else {
		imgSrc = []byte(url.String())
	}

	_, _ = w.WriteString("<img src=\"")
	if absURLs && baseURL != "" && bytes.HasPrefix(imgSrc, []byte("/")) {
		_, _ = w.Write(util.EscapeHTML([]byte(baseURL)))
	}
	if unsafe || !html.IsDangerousURL(imgSrc) {
		_, _ = w.Write(util.EscapeHTML(imgSrc))
	}
	_, _ = w.WriteRune('"')

	if uPhoto {
		_, _ = w.WriteString(" class=\"u-photo\"")
	}

	if alt != "" {
		_, _ = w.WriteString(` alt="`)
		_, _ = w.Write(util.EscapeHTML([]byte(alt)))
		_, _ = w.WriteRune('"')
	}
	_, _ = w.WriteString(" loading=\"lazy\">")
	_, _ = w.WriteString("</picture>")

	if caption && title != "" {
		_, _ = w.WriteString("<figcaption>")
		_, _ = w.Write(util.EscapeHTML([]byte(title)))
		_, _ = w.WriteString("</figcaption>")
	}

	_, _ = w.WriteString("</figure>")
	return nil
}

// TODO(future): perhaps make this customizable.
func makePictureSourceSet(id, format string) string {
	return "https://cdn.hacdias.com/i/t/" + id + "-250x." + format + " 250w" +
		", https://cdn.hacdias.com/i/t/" + id + "-500x." + format + " 500w" +
		", https://cdn.hacdias.com/i/t/" + id + "-1000x." + format + " 1000w" +
		", https://cdn.hacdias.com/i/t/" + id + "-2000x." + format + " 2000w"
}
