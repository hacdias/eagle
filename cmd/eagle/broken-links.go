package main

import (
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/hacdias/eagle/core"
	"github.com/spf13/cobra"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
)

func init() {
	rootCmd.AddCommand(brokenLinksCmd)
}

var brokenLinksCmd = &cobra.Command{
	Use: "broken-links",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := core.ParseConfig()
		if err != nil {
			return err
		}

		fs := core.NewFS(c.SourceDirectory, c.BaseURL, &core.NopSync{})

		redirects, err := fs.LoadRedirects(false)
		if err != nil {
			return err
		}

		var resolveRedirect func(u string) string
		resolveRedirect = func(u string) string {
			if r, ok := redirects[u]; ok {
				return resolveRedirect(r)
			}

			return u
		}

		ee, err := fs.GetEntries(false)
		if err != nil {
			return err
		}

		getMarkdownURLs := func(e *core.Entry) ([]string, error) {
			r, md := newMarkdown()
			err = r.Convert([]byte(e.Content), io.Discard)
			if err != nil {
				return nil, err
			}

			urls := md.md.urls

			if e.Reply != "" {
				urls = append(urls, e.Reply)
			}
			if e.Context != nil {
				urls = append(urls, e.Context.URL)
			}

			return urls, nil
		}

		isBroken := func(urlStr string) (bool, string, error) {
			if strings.HasPrefix(urlStr, "/") || strings.HasPrefix(urlStr, c.BaseURL) {
				u, err := url.Parse(urlStr)
				if err != nil {
					return false, "", err
				}

				if strings.HasPrefix(u.Path, "/tags") {
					return false, "", nil
				}

				u.Path = strings.TrimSuffix(u.Path, "/")

				_, err = fs.GetEntry(u.Path)
				if err != nil {
					_, err := fs.ReadFile(filepath.Join("content", u.Path))
					if err != nil {
						return true, u.Path, nil
					}
				}
			}

			return false, "", nil
		}

		printBroken := func(e *core.Entry, what string, urls []string) {
			if len(urls) != 0 {
				fmt.Println(what, e.ID)
				for _, l := range urls {
					r := resolveRedirect(l)
					if r != l {
						fmt.Println("R", l, "->", r)
					} else {
						fmt.Println("D", l)
					}
				}

				fmt.Println("")
			}
		}

		for _, e := range ee {
			markdownURLs, err := getMarkdownURLs(e)
			if err != nil {
				return err
			}
			brokenLinks := []string{}
			for _, urlStr := range markdownURLs {
				broken, canonical, err := isBroken(urlStr)
				if err != nil {
					return err
				}
				if broken {
					brokenLinks = append(brokenLinks, canonical)
				}
			}
			printBroken(e, "Entry", brokenLinks)
		}

		return nil
	},
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

type markdown struct {
	md *markdownRenderer
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
