package indienews

import (
	"context"
	"errors"
	"slices"

	"github.com/karlseguin/typed"
	"go.hacdias.com/eagle/core"
	"go.hacdias.com/eagle/server"
	"go.hacdias.com/indielib/micropub"
)

var (
	_ server.SyndicationPlugin = &IndieNews{}
)

func init() {
	server.RegisterPlugin("indienews", NewIndieNews)
}

type IndieNews struct {
	url  string
	lang string
}

func NewIndieNews(co *core.Core, configMap map[string]any) (server.Plugin, error) {
	config := typed.New(configMap)

	language := config.String("language")
	if language == "" {
		return nil, errors.New("language missing")
	}

	return &IndieNews{
		lang: language,
		url:  "https://news.indieweb.org/" + language,
	}, nil
}

func (m *IndieNews) Syndication() micropub.Syndication {
	return micropub.Syndication{
		UID:  "indienews",
		Name: "IndieNews",
	}
}

func (m *IndieNews) IsSyndicated(e *core.Entry) bool {
	return slices.Contains(typed.New(e.Other).Strings(server.SyndicationField), m.url)
}

func (m *IndieNews) Syndicate(ctx context.Context, e *core.Entry, _ *server.SyndicationContext) ([]string, []string, error) {
	return []string{m.url}, []string{m.url}, nil
}
