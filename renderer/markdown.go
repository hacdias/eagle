package renderer

import (
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
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
		newFiguresRenderer(r, absoluteURLs),
	}

	if r.c.Site.ChromaTheme != "" {
		exts = append(exts, highlighting.NewHighlighting(
			highlighting.WithStyle(r.c.Site.ChromaTheme),
		))
	}

	return goldmark.New(append(defaultGoldmarkOptions, goldmark.WithExtensions(exts...))...)
}
