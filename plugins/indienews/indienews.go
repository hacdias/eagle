package mastodon

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

	lang := config.String("lang")
	if lang == "" {
		return nil, errors.New("lang missing")
	}

	return &IndieNews{
		lang: lang,
		url:  "https://news.indieweb.org/" + lang,
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

func (m *IndieNews) Syndicate(ctx context.Context, e *core.Entry, photos []server.Photo) (string, bool, error) {
	return m.url, false, nil
}
