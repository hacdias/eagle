package mastodon

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/karlseguin/typed"
	"github.com/mattn/go-mastodon"
	"go.hacdias.com/eagle/core"
	"go.hacdias.com/eagle/server"
	"go.hacdias.com/indielib/micropub"
)

var (
	_ server.SyndicationPlugin = &Mastodon{}
)

func init() {
	server.RegisterPlugin("mastodon", NewMastodon)
}

type Mastodon struct {
	core   *core.Core
	client *mastodon.Client
}

func NewMastodon(co *core.Core, config map[string]interface{}) (server.Plugin, error) {
	server := typed.New(config).String("server")
	if server == "" {
		return nil, errors.New("server missing")
	}

	clientKey := typed.New(config).String("clientkey")
	if clientKey == "" {
		return nil, errors.New("clientKey missing")
	}

	clientSecret := typed.New(config).String("clientsecret")
	if clientSecret == "" {
		return nil, errors.New("clientSecret missing")
	}

	accessToken := typed.New(config).String("accesstoken")
	if accessToken == "" {
		return nil, errors.New("accessToken missing")
	}

	return &Mastodon{
		core: co,
		client: mastodon.NewClient(&mastodon.Config{
			Server:       server,
			ClientID:     clientKey,
			ClientSecret: clientSecret,
			AccessToken:  accessToken,
		}),
	}, nil
}

func (ld *Mastodon) Syndication() micropub.Syndication {
	return micropub.Syndication{
		UID:  "mastodon",
		Name: "Mastodon",
	}
}

func (ld *Mastodon) extractID(urlStr string) (mastodon.ID, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}

	parts := strings.Split(u.Path, "/")
	if len(parts) != 3 {
		return "", fmt.Errorf("expected url to have 3 parts, has %d", len(parts))
	}

	return mastodon.ID(parts[2]), nil
}

func (ld *Mastodon) getSyndication(e *core.Entry) (string, mastodon.ID, error) {
	syndications := typed.New(e.Other).Strings(server.SyndicationField)
	for _, urlStr := range syndications {
		if strings.HasPrefix(urlStr, ld.client.Config.Server) {
			id, err := ld.extractID(urlStr)
			return urlStr, id, err
		}
	}
	return "", "", nil
}

func (ld *Mastodon) IsSyndicated(e *core.Entry) bool {
	_, id, err := ld.getSyndication(e)
	if err != nil {
		return false
	}
	return id != ""
}

func (ld *Mastodon) Syndicate(ctx context.Context, e *core.Entry) (string, bool, error) {
	url, id, err := ld.getSyndication(e)
	if err != nil {
		return "", false, err
	}

	if id != "" {
		if e.Deleted() || e.Draft {
			return url, true, ld.client.DeleteStatus(ctx, id)
		}
	}

	textContent := e.TextContent()
	addPermalink := false

	if textContent == "" || len(textContent) >= 500 {
		textContent = e.Title
		addPermalink = true
	} else if _, ok := e.Other["photos"]; ok {
		addPermalink = true
	}

	if addPermalink {
		textContent += "\n\n" + e.Permalink + "\n"
	}

	toot := mastodon.Toot{
		Visibility: mastodon.VisibilityPublic,
		Status:     textContent,
	}

	var status *mastodon.Status
	if id != "" {
		status, err = ld.client.UpdateStatus(ctx, &toot, id)
	} else {
		status, err = ld.client.PostStatus(ctx, &toot)
	}
	if err != nil {
		return "", false, err
	}

	return status.URL, false, nil
}
