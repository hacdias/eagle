package core

import (
	"bytes"
	"io"
	urlpkg "net/url"
	"path/filepath"

	"github.com/samber/lo"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
	"willnorris.com/go/webmention"
)

// GetEntryLinks gets the links found in the HTML rendered version of the entry.
// This uses the latest available build to check for the links. Entry must have
// .h-entry and .e-content classes.
func (co *Core) GetEntryLinks(permalink string) ([]string, error) {
	url, err := urlpkg.Parse(permalink)
	if err != nil {
		return nil, err
	}

	filename := filepath.Join(co.buildName, url.Path, "index.html")
	html, err := co.buildFS.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	targets, err := webmention.DiscoverLinksFromReader(bytes.NewBuffer(html), permalink, ".h-entry .e-content a, .h-entry .h-cite a")
	if err != nil {
		return nil, err
	}

	targets = (lo.Filter(targets, func(target string, _ int) bool {
		url, err := urlpkg.Parse(target)
		if err != nil {
			return false
		}

		return url.Scheme == "http" || url.Scheme == "https"
	}))

	return lo.Uniq(targets), nil
}

// IsLinkValid checks if the given link exists in the built version of the website.
func (co *Core) IsLinkValid(permalink string) (bool, error) {
	url, err := urlpkg.Parse(permalink)
	if err != nil {
		return false, err
	}

	_, err = co.buildFS.Stat(filepath.Join(co.buildName, url.Path))
	if err == nil {
		return true, nil
	}

	_, err = co.buildFS.Stat(filepath.Join(co.buildName, url.Path, "index.html"))
	if err == nil {
		return true, err
	}

	return false, nil
}

func GetMarkdownURLs(e *Entry) ([]string, error) {
	r, md := newMarkdown()
	err := r.Convert([]byte(e.Content), io.Discard)
	if err != nil {
		return nil, err
	}

	return lo.Uniq(md.md.urls), nil
}

type markdown struct {
	md *markdownRenderer
}

func newMarkdown() (goldmark.Markdown, *markdown) {
	exts := []goldmark.Extender{}
	md := &markdown{newMarkdownRenderer()}
	exts = append(exts, md)
	return goldmark.New(append([]goldmark.Option{
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
	}, goldmark.WithExtensions(exts...))...), md
}

func (md *markdown) Extend(m goldmark.Markdown) {
	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(md.md, 100),
	))
}

func newMarkdownRenderer() *markdownRenderer {
	return &markdownRenderer{
		Config: html.Config{
			Writer: html.DefaultWriter,
		},
	}
}

type markdownRenderer struct {
	html.Config
	urls []string
}

func (r *markdownRenderer) SetOption(name renderer.OptionName, value interface{}) {
	r.Config.SetOption(name, value)
}

func (r *markdownRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindLink, r.renderLink)
	reg.Register(ast.KindAutoLink, r.renderAutoLink)
}

func (r *markdownRenderer) renderLink(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.Link)
	if !entering {
		return ast.WalkContinue, nil
	}

	url := util.URLEscape(n.Destination, true)
	r.urls = append(r.urls, string(url))
	return ast.WalkContinue, nil
}

func (r *markdownRenderer) renderAutoLink(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.AutoLink)
	if !entering {
		return ast.WalkContinue, nil
	}
	url := n.URL(source)
	r.urls = append(r.urls, string(url))
	return ast.WalkContinue, nil
}
