package eagle

import (
	"bytes"

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
		&links{},
	),
}

func newMarkdown(absURLs bool, baseURL string) goldmark.Markdown {
	return goldmark.New(append(defaultGoldmarkOptions, goldmark.WithExtensions(
		&links{
			absURLs: absURLs,
			baseURL: baseURL,
		},
	))...)
}

// TODO: image rendering and absURLs
// TODO: figure rendering and absURLs

type links struct {
	baseURL string
	absURLs bool
}

// Extend implements goldmark.Extender.
func (l *links) Extend(m goldmark.Markdown) {
	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(newLinkRenderer(l), 100),
	))
}

func newLinkRenderer(l *links) renderer.NodeRenderer {
	r := &hookedRenderer{
		Config: html.Config{
			Writer: html.DefaultWriter,
		},
		baseURL: l.baseURL,
		absURLs: l.absURLs,
	}
	return r
}

type hookedRenderer struct {
	html.Config
	baseURL string
	absURLs bool
}

func (r *hookedRenderer) SetOption(name renderer.OptionName, value interface{}) {
	r.Config.SetOption(name, value)
}

// RegisterFuncs implements NodeRenderer.RegisterFuncs.
func (r *hookedRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindLink, r.renderLink)
	reg.Register(ast.KindAutoLink, r.renderAutoLink)
}

// https://github.com/yuin/goldmark/blob/5588d92a56fe1642791cf4aa8e9eae8227cfeecd/renderer/html/html.go#L439

func (r *hookedRenderer) renderLink(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
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

func (r *hookedRenderer) renderAutoLink(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
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
