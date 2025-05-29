package indienews

import (
	"context"
	"errors"

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

func NewIndieNews(co *core.Core, configMap map[string]interface{}) (server.Plugin, error) {
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
	for _, urlStr := range typed.New(e.Other).Strings(server.SyndicationField) {
		if urlStr == m.url {
			return true
		}
	}
	return false
}

func (m *IndieNews) Syndicate(ctx context.Context, e *core.Entry, _ *server.SyndicationContext) (string, bool, error) {
	return m.url, false, nil
}
