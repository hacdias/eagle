package eagle

import (
	"github.com/yuin/goldmark"
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

// publicAddress := ""
// if srv := a.cfg.Server; srv != nil {
// 	publicAddress = srv.PublicAddress
// }
// a.md = goldmark.New(append(defaultGoldmarkOptions, goldmark.WithExtensions(&customExtension{
// 	absoluteLinks: false,
// 	publicAddress: publicAddress,
// }))...)
