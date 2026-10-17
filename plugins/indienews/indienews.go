package indienews

import (
	"context"
	"errors"
	"slices"

	"go.hacdias.com/eagle/core"
	"go.hacdias.com/eagle/server"
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

type indieNewsConfig struct {
	Language string
}

func NewIndieNews(co *core.Core, config indieNewsConfig) (server.Plugin, error) {
	if config.Language == "" {
		return nil, errors.New("language missing")
	}

	return &IndieNews{
		lang: config.Language,
		url:  "https://news.indieweb.org/" + config.Language,
	}, nil
}

func (m *IndieNews) Syndicator() server.Syndicator {
	return server.Syndicator{
		UID:  "indienews",
		Name: "IndieNews",
	}
}

func (m *IndieNews) IsSyndicated(e *core.Entry) bool {
	return slices.Contains(e.Syndications, m.url)
}

func (m *IndieNews) Syndicate(ctx context.Context, e *core.Entry, _ *server.SyndicationContext) error {
	if !m.IsSyndicated(e) {
		e.Syndications = append(e.Syndications, m.url)
	}

	return nil
}
