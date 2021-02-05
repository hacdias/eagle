package server

import (
	"bytes"
	"html/template"
	"io"

	"github.com/hacdias/eagle/eagle"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

var md = goldmark.New(
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
	),
)

func (s *Server) RenderHTML(entry *eagle.Entry, w io.Writer) error {
	tpl := template.Must(s.tpl.Clone())

	var buf bytes.Buffer
	err := md.Convert([]byte(entry.Content), &buf)
	if err != nil {
		return err
	}

	if entry.Metadata.Template == "" {
		entry.Metadata.Template = "page.tmpl"
	}

	// TODO: add context specific functions

	return tpl.ExecuteTemplate(w, entry.Metadata.Template, map[string]interface{}{
		"Content": template.HTML(buf.Bytes()),
		"Page":    entry.Metadata,
	})
}
