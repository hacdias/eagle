package renderer

import (
	"bytes"

	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting"
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

func newMarkdown(r *Renderer, absURLs bool) goldmark.Markdown {
	exts := []goldmark.Extender{}

	if r.c.Site.ChromaTheme != "" {
		exts = append(exts, highlighting.NewHighlighting(
			highlighting.WithStyle(r.c.Site.ChromaTheme),
		))
	}

	exts = append(exts, &markdown{
		r:       r,
		absURLs: absURLs,
	})

	return goldmark.New(append(defaultGoldmarkOptions, goldmark.WithExtensions(exts...))...)
}

type markdown struct {
	r       *Renderer
	absURLs bool
}

func (md *markdown) Extend(m goldmark.Markdown) {
	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(newMarkdownRenderer(md), 100),
	))
}

func newMarkdownRenderer(md *markdown) renderer.NodeRenderer {
	return &markdownRenderer{
		Config: html.Config{
			Writer: html.DefaultWriter,
		},
		markdown: md,
	}
}

type markdownRenderer struct {
	html.Config
	*markdown
}

func (r *markdownRenderer) SetOption(name renderer.OptionName, value interface{}) {
	r.Config.SetOption(name, value)
}

// RegisterFuncs implements NodeRenderer.RegisterFuncs.
func (r *markdownRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindLink, r.renderLink)
	reg.Register(ast.KindImage, r.renderImage)
	reg.Register(ast.KindAutoLink, r.renderAutoLink)
}

// https://github.com/yuin/goldmark/blob/5588d92a56fe1642791cf4aa8e9eae8227cfeecd/renderer/html/html.go#L439

func (r *markdownRenderer) renderLink(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.Link)
	if entering {
		_, _ = w.WriteString("<a href=\"")
		destination := util.URLEscape(n.Destination, true)
		if r.absURLs && r.r.c.Server.BaseURL != "" && bytes.HasPrefix(destination, []byte("/")) {
			_, _ = w.Write(util.EscapeHTML([]byte(r.r.c.Server.BaseURL)))
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
			_, _ = w.WriteString(` rel="noopener noreferrer" target="_blank" `)
		}
		_ = w.WriteByte('>')
	} else {
		_, _ = w.WriteString("</a>")
	}
	return ast.WalkContinue, nil
}

func (r *markdownRenderer) renderAutoLink(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
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
	if r.absURLs && r.r.c.Server.BaseURL != "" && bytes.HasPrefix(destination, []byte("/")) {
		_, _ = w.Write(util.EscapeHTML([]byte(r.r.c.Server.BaseURL)))
	}
	_, _ = w.Write(util.EscapeHTML(destination))
	if n.Attributes() != nil {
		_ = w.WriteByte('"')
		html.RenderAttributes(w, n, html.LinkAttributeFilter)
	} else {
		_, _ = w.WriteString(`"`)
	}

	if n.AutoLinkType == ast.AutoLinkURL && !(bytes.HasPrefix(url, []byte("/")) || bytes.HasPrefix(destination, []byte("#"))) {
		_, _ = w.WriteString(` rel="noopener noreferrer" target="_blank" `)
	}

	_ = w.WriteByte('>')
	_, _ = w.Write(util.EscapeHTML(label))
	_, _ = w.WriteString(`</a>`)
	return ast.WalkContinue, nil
}

// Hijack the image rendering and output <figure>!
func (r *markdownRenderer) renderImage(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	n := node.(*ast.Image)

	err := r.r.writeFigure(w, string(n.Destination), string(n.Text(source)), string(n.Title), r.absURLs, r.Unsafe, false)
	if err != nil {
		return ast.WalkStop, err
	}

	return ast.WalkSkipChildren, nil
}
