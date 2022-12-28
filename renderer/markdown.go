package renderer

import (
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

func newMarkdown(r *Renderer, absoluteURLs bool) goldmark.Markdown {
	exts := []goldmark.Extender{
		newLinksRenderer(r, absoluteURLs),
	}

	if r.c.Site.ChromaTheme != "" {
		exts = append(exts, highlighting.NewHighlighting(
			highlighting.WithStyle(r.c.Site.ChromaTheme),
		))
	}

	exts = append(exts, &markdown{
		r:       r,
		absURLs: absoluteURLs,
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
	reg.Register(ast.KindImage, r.renderImage)
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
