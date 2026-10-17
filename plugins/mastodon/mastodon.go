package mastodon

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/mattn/go-mastodon"
	"github.com/samber/lo"
	"go.hacdias.com/eagle/core"
	"go.hacdias.com/eagle/log"
	"go.hacdias.com/eagle/server"
	"go.uber.org/zap"
)

var (
	_ server.SyndicationPlugin = &Mastodon{}
)

func init() {
	server.RegisterPlugin("mastodon", NewMastodon)
}

type Mastodon struct {
	core              *core.Core
	log               *zap.SugaredLogger
	client            *mastodon.Client
	maximumCharacters int
	maximumPhotos     int
}

type mastodonConfig struct {
	Server            string
	ClientKey         string
	ClientSecret      string
	AccessToken       string
	MaximumCharacters int
	MaximumPhotos     int
}

func NewMastodon(co *core.Core, config mastodonConfig) (server.Plugin, error) {
	if config.Server == "" {
		return nil, errors.New("server missing")
	}

	if config.ClientKey == "" {
		return nil, errors.New("clientKey missing")
	}

	if config.ClientSecret == "" {
		return nil, errors.New("clientSecret missing")
	}

	if config.AccessToken == "" {
		return nil, errors.New("accessToken missing")
	}

	if config.MaximumCharacters == 0 {
		config.MaximumCharacters = 500
	}

	if config.MaximumPhotos == 0 {
		config.MaximumPhotos = 5
	}

	return &Mastodon{
		core: co,
		log:  log.S().Named("mastodon"),
		client: mastodon.NewClient(&mastodon.Config{
			Server:       config.Server,
			ClientID:     config.ClientKey,
			ClientSecret: config.ClientSecret,
			AccessToken:  config.AccessToken,
		}),
		maximumCharacters: config.MaximumCharacters,
		maximumPhotos:     config.MaximumPhotos,
	}, nil
}

func (m *Mastodon) Syndicator() server.Syndicator {
	return server.Syndicator{
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
	for _, urlStr := range e.Syndications {
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

func (m *Mastodon) uploadPhotos(ctx context.Context, photos []*server.Photo) []mastodon.ID {
	mediaIDs := []mastodon.ID{}

	for i, photo := range photos {
		if i >= m.maximumPhotos {
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

func (m *Mastodon) Syndicate(ctx context.Context, e *core.Entry, sctx *server.SyndicationContext) error {
	url, id, err := m.getSyndication(e)
	if err != nil {
		return err
	}

	if id != "" {
		if e.Deleted() || e.Draft || e.Future() {
			err = m.client.DeleteStatus(ctx, id)
			if err != nil {
				return err
			}

			e.Syndications = lo.Without(e.Syndications, url)
			return nil
		}
	}

	if id != "" {
		// Existing Mastodon posts are not updated to avoid overwriting custom posts.
		return nil
	}

	toot := mastodon.Toot{
		Visibility: mastodon.VisibilityPublic,
		MediaIDs:   m.uploadPhotos(ctx, sctx.Photos),
	}

	if sctx.Status != "" {
		toot.Status = sctx.Status + " " + e.Permalink
	} else {
		statuses := e.Statuses(m.maximumCharacters, 1, len(e.Photos) > len(toot.MediaIDs))
		if len(statuses) != 1 {
			return fmt.Errorf("expected 1 status, got %d", len(statuses))
		}
		toot.Status = statuses[0]
	}

	status, err := m.client.PostStatus(ctx, &toot)
	if err != nil {
		return err
	}

	e.Syndications = append(e.Syndications, status.URL)
	return nil
}
