package core

import (
	"io"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/samber/lo"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
	"golang.org/x/net/publicsuffix"
)

const (
	ExternalLinksFile = "external-links.json"
)

type Link struct {
	SourceURL string `json:"sourceUrl"`
	TargetURL string `json:"targetUrl"`
}

type Links struct {
	Domain string `json:"domain"`
	Count  int    `json:"count"`
	Links  []Link `json:"links"`
}

func (f *FS) LoadExternalLinks() ([]Links, error) {
	var links []Links
	err := f.ReadJSON(filepath.Join(DataDirectory, ExternalLinksFile), &links)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return links, nil
}

func (f *FS) UpdateExternalLinks() error {
	ee, err := f.GetEntries(false)
	if err != nil {
		return err
	}

	linksMap := map[string][]Link{}
	for _, e := range ee {
		urls, err := GetMarkdownURLs(e)
		if err != nil {
			return err
		}

		for _, urlStr := range urls {
			if strings.HasPrefix(urlStr, "/") || strings.HasPrefix(urlStr, f.parser.baseURL) {
				continue
			}

			u, err := url.Parse(urlStr)
			if err != nil {
				return err
			}

			hostname := u.Hostname()
			if hostname == "" {
				continue
			}

			hostname, err = publicsuffix.EffectiveTLDPlusOne(hostname)
			if err != nil {
				return err
			}

			if strings.HasSuffix(f.parser.baseURL, hostname) {
				continue
			}

			if _, ok := linksMap[hostname]; !ok {
				linksMap[hostname] = []Link{}
			}

			linksMap[hostname] = append(linksMap[hostname], Link{
				SourceURL: e.Permalink,
				TargetURL: u.String(),
			})
		}
	}

	newLinks := []Links{}
	for domain, domainLinks := range linksMap {
		sort.Slice(domainLinks, func(i, j int) bool {
			return domainLinks[i].SourceURL < domainLinks[j].SourceURL
		})

		newLinks = append(newLinks, Links{
			Domain: domain,
			Count:  len(domainLinks),
			Links:  domainLinks,
		})
	}

	sort.Slice(newLinks, func(i, j int) bool {
		if newLinks[i].Count == newLinks[j].Count {
			return newLinks[i].Domain < newLinks[j].Domain
		}

		return newLinks[i].Count > newLinks[j].Count
	})

	oldLinks, err := f.LoadExternalLinks()
	if err != nil {
		return err
	}

	if reflect.DeepEqual(oldLinks, newLinks) {
		return nil
	}

	return f.WriteJSON(filepath.Join(DataDirectory, ExternalLinksFile), newLinks, "update external links file")
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
