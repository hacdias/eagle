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
	"go.hacdias.com/eagle/log"
	"go.hacdias.com/eagle/server"
	"go.hacdias.com/indielib/micropub"
	"go.uber.org/zap"
)

var (
	_ server.SyndicationPlugin = &Mastodon{}
)

func init() {
	server.RegisterPlugin("mastodon", NewMastodon)
}

type Mastodon struct {
	core   *core.Core
	log    *zap.SugaredLogger
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
		log:  log.S().Named("mastodon"),
		client: mastodon.NewClient(&mastodon.Config{
			Server:       server,
			ClientID:     clientKey,
			ClientSecret: clientSecret,
			AccessToken:  accessToken,
		}),
	}, nil
}

func (m *Mastodon) Syndication() micropub.Syndication {
	return micropub.Syndication{
		UID:  "mastodon",
		Name: "Mastodon",
	}
}

func (m *Mastodon) extractID(urlStr string) (mastodon.ID, error) {
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

func (m *Mastodon) getSyndication(e *core.Entry) (string, mastodon.ID, error) {
	syndications := typed.New(e.Other).Strings(server.SyndicationField)
	for _, urlStr := range syndications {
		if strings.HasPrefix(urlStr, m.client.Config.Server) {
			id, err := m.extractID(urlStr)
			return urlStr, id, err
		}
	}
	return "", "", nil
}

func (m *Mastodon) IsSyndicated(e *core.Entry) bool {
	_, id, err := m.getSyndication(e)
	if err != nil {
		return false
	}
	return id != ""
}

func (m *Mastodon) uploadPhotos(ctx context.Context, photos []server.Photo) []mastodon.ID {
	mediaIDs := []mastodon.ID{}

	for i, photo := range photos {
		if i >= 5 {
			break
		}

		attachment, err := m.client.UploadMediaFromBytes(ctx, photo.Data)
		if err != nil {
			m.log.Warnw("photo upload failed", "mimetype", photo.MimeType, "err", err)
			continue
		}

		mediaIDs = append(mediaIDs, attachment.ID)
	}

	return mediaIDs
}

func (m *Mastodon) Syndicate(ctx context.Context, e *core.Entry, photos []server.Photo) (string, bool, error) {
	url, id, err := m.getSyndication(e)
	if err != nil {
		return "", false, err
	}

	if id != "" {
		if e.Deleted() || e.Draft {
			return url, true, m.client.DeleteStatus(ctx, id)
		}
	}

	toot := mastodon.Toot{
		Visibility: mastodon.VisibilityPublic,
	}

	if id == "" {
		toot.MediaIDs = m.uploadPhotos(ctx, photos)
	} else {
		status, err := m.client.GetStatus(ctx, id)
		if err != nil {
			return "", false, err
		}

		for _, attachment := range status.MediaAttachments {
			toot.MediaIDs = append(toot.MediaIDs, attachment.ID)
		}
	}

	textContent := e.TextContent()
	addPermalink := len(photos) != len(toot.MediaIDs)

	if textContent == "" || len(textContent) >= 500-(len(e.Permalink)+3) {
		textContent = e.Title
		addPermalink = true
	}

	if addPermalink {
		textContent += "\n\n" + e.Permalink + "\n"
	}

	toot.Status = textContent

	var status *mastodon.Status
	if id != "" {
		status, err = m.client.UpdateStatus(ctx, &toot, id)
	} else {
		status, err = m.client.PostStatus(ctx, &toot)
	}
	if err != nil {
		return "", false, err
	}

	return status.URL, false, nil
}
