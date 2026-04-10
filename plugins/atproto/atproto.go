package atproto

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/bluesky-social/indigo/xrpc"
	"github.com/go-viper/mapstructure/v2"
	"github.com/samber/lo"
	"go.hacdias.com/eagle/core"
	"go.hacdias.com/eagle/log"
	"go.hacdias.com/eagle/server"
	"go.uber.org/zap"
)

var (
	_ server.SyndicationPlugin = &ATProto{}
	_ server.HandlerPlugin     = &ATProto{}
	_ server.CronPlugin        = &ATProto{}
)

const (
	maximumCharacters = 300
	maximumPhotos     = 4
)

func init() {
	server.RegisterPlugin("atproto", NewATProto)
}

type atprotoConfig struct {
	Host            string
	Identifier      string
	Password        string
	ArabicaFilename string
	StandardSite    standardSite
}

type ATProto struct {
	core       *core.Core
	log        *zap.SugaredLogger
	host       string
	identifier string
	password   string
	userAgent  string

	// site.standard
	standardSite               standardSite
	standardSitePublicationUri string

	// alpha.arabica.social
	arabicaFilename string
}

func NewATProto(co *core.Core, configMap map[string]any) (server.Plugin, error) {
	var config atprotoConfig

	err := mapstructure.Decode(configMap, &config)
	if err != nil {
		return nil, err
	}

	if config.Host == "" {
		config.Host = "https://bsky.social"
	}

	if config.Identifier == "" {
		return nil, errors.New("identifier missing")
	}

	if config.Password == "" {
		return nil, errors.New("password missing")
	}

	if config.StandardSite.RecordKey == "" {
		return nil, errors.New("standardSite.recordKey missing")
	}

	at := &ATProto{
		core:            co,
		userAgent:       fmt.Sprintf("eagle/%s", co.BaseURL().String()),
		host:            config.Host,
		identifier:      config.Identifier,
		password:        config.Password,
		log:             log.S().Named("atproto"),
		standardSite:    config.StandardSite,
		arabicaFilename: config.ArabicaFilename,
	}

	return at, at.init()
}

func (at *ATProto) init() error {
	ctx := context.Background()

	client, err := at.getClient(ctx)
	if err != nil {
		return err
	}

	return at.initStandardPublication(ctx, client)
}

func (at *ATProto) Syndicator() server.Syndicator {
	return server.Syndicator{
		UID:     "atproto",
		Name:    "ATProto",
		Default: true,
	}
}

func (at *ATProto) getClient(ctx context.Context) (*xrpc.Client, error) {
	client := &xrpc.Client{
		Host:      at.host,
		UserAgent: &at.userAgent,
	}

	sess, err := atproto.ServerCreateSession(context.Background(), client, &atproto.ServerCreateSession_Input{
		Identifier: at.identifier,
		Password:   at.password,
	})
	if err != nil {
		return nil, err
	}

	client.Auth = &xrpc.AuthInfo{
		AccessJwt:  sess.AccessJwt,
		RefreshJwt: sess.RefreshJwt,
		Handle:     sess.Handle,
		Did:        sess.Did,
	}

	return client, nil
}

type syndications struct {
	feedPosts    []syntax.ATURI
	document     *syntax.ATURI
	grainGallery *syntax.ATURI
}

func (at *ATProto) getSyndications(e *core.Entry) (*syndications, error) {
	s := &syndications{
		feedPosts: []syntax.ATURI{},
	}

	for _, syndication := range e.Syndications {
		if !strings.HasPrefix(syndication, "at://") {
			continue
		}

		uri, err := syntax.ParseATURI(syndication)
		if err != nil {
			return nil, err
		}

		switch uri.Collection() {
		case "app.bsky.feed.post":
			s.feedPosts = append(s.feedPosts, uri)
		case "site.standard.document":
			s.document = &uri
		case "social.grain.gallery":
			s.grainGallery = &uri
		}
	}

	return s, nil
}

func (at *ATProto) IsSyndicated(e *core.Entry) bool {
	s, err := at.getSyndications(e)
	if err != nil {
		return false
	}
	return len(s.feedPosts) > 0 || s.document != nil || s.grainGallery != nil
}

func (at *ATProto) deleteBlueskyPosts(ctx context.Context, client *xrpc.Client, uris []syntax.ATURI) error {
	for _, uri := range uris {
		err := at.deleteBlueskyPost(ctx, client, uri.RecordKey().String())
		if err != nil {
			return err
		}
	}

	return nil
}

func (at *ATProto) getBlueskyPosts(ctx context.Context, client *xrpc.Client, uris []syntax.ATURI) ([]*blueskyPost, error) {
	posts := []*blueskyPost{}

	for _, uri := range uris {
		post, err := at.getBlueskyPost(ctx, client, uri.RecordKey().String())
		if err != nil {
			return nil, err
		}

		posts = append(posts, post)
	}

	// Sort to ensure that the first post is the root of the thread.
	slices.SortFunc(posts, func(a *blueskyPost, b *blueskyPost) int {
		if a.Reply == nil && b.Reply == nil {
			return 0
		}

		if a.Reply == nil && b.Reply != nil {
			return -1
		}

		if a.Reply != nil && b.Reply == nil {
			return 1
		}

		if a.Reply.Parent != nil && a.Reply.Parent.Uri == b.uri {
			return 1
		}

		if b.Reply.Parent != nil && b.Reply.Parent.Uri == a.uri {
			return -1
		}

		return 0
	})

	return posts, nil
}

func (at *ATProto) Syndicate(ctx context.Context, e *core.Entry, sctx *server.SyndicationContext) error {
	s, err := at.getSyndications(e)
	if err != nil {
		return err
	}

	client, err := at.getClient(ctx)
	if err != nil {
		return err
	}

	if e.Deleted() || e.Draft {
		if s.grainGallery != nil {
			err = at.deleteGrainGallery(ctx, client, *s.grainGallery)
			if err != nil {
				return err
			}
			e.Syndications = lo.Without(e.Syndications, s.grainGallery.String())
		}

		if s.document != nil {
			err = at.deleteStandardDocument(ctx, client, *s.document)
			if err != nil {
				return err
			}
			e.Syndications = lo.Without(e.Syndications, s.document.String())
		}

		err = at.deleteBlueskyPosts(ctx, client, s.feedPosts)
		if err != nil {
			return err
		}

		e.Syndications = lo.Without(e.Syndications, lo.Map(s.feedPosts, func(uri syntax.ATURI, i int) string {
			return uri.String()
		})...)

		return nil
	}

	posts, err := at.getBlueskyPosts(ctx, client, s.feedPosts)
	if err != nil {
		return err
	}

	if lo.Contains(e.Categories, "writings") {
		var post *blueskyPost

		if len(posts) > 0 {
			// Existing Bluesky posts are not updated to avoid overwriting custom posts.
			// First post (root of thread) is selected to be linked on the standard.site
			// document.
			post = posts[0]
		} else {
			var thumbnail *photoBlob
			if sctx.Thumbnail != nil {
				thumbnail, err = uploadPhoto(ctx, client, sctx.Thumbnail)
				if err != nil {
					return err
				}
			}

			post, err = at.createPublishBlueskyPost(ctx, client, e, sctx, thumbnail)
			if err != nil {
				return err
			}

			e.Syndications = append(e.Syndications, post.uri)
		}

		// Upsert standard.site document to ensure that it is up to date (tags, content,
		// link to Bluesky post, etc).
		documentUriStr, err := at.upsertStandardDocument(ctx, client, s.document, e, post)
		if err != nil {
			return err
		}

		if s.document == nil {
			// Only add the syndication if we didn't have a documentURI before, otherwise it means we already had the syndication and we just updated the record.
			e.Syndications = append(e.Syndications, documentUriStr)
		}

		return nil
	}

	if lo.Contains(e.Categories, "photos") {
		// Get photos photos: either extract from existing posts or upload fresh.
		var photos []*photoBlob
		if len(posts) > 0 {
			photos = blueskyPostToPhotoBlobs(posts)
		} else {
			photos, err = uploadPhotos(ctx, client, sctx.Photos)
			if err != nil {
				return err
			}
		}

		// Create Bluesky thread if it doesn't exist yet.
		if len(posts) == 0 {
			newPosts, err := at.createPublishBlueskyPostThread(ctx, client, e, sctx, photos)
			if err != nil {
				return err
			}

			e.Syndications = append(e.Syndications, lo.Map(newPosts, func(post *blueskyPost, i int) string {
				return post.uri
			})...)
		}

		// Create Grain gallery if it doesn't exist yet.
		if s.grainGallery == nil {
			galleryURI, err := at.createGrainGallery(ctx, client, e, photos)
			if err != nil {
				return err
			}
			e.Syndications = append(e.Syndications, galleryURI)
		}

		return nil
	}

	return errors.New("atproto syndication only supports writings and photos categories")
}

func (at *ATProto) DailyCron() error {
	return at.UpdateCoffee(context.Background())
}
