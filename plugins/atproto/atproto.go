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
	"github.com/karlseguin/typed"
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
	apiUrl            = "https://bsky.social"
	maximumCharacters = 300
	maximumPhotos     = 4
)

func init() {
	server.RegisterPlugin("atproto", NewATProto)
}

type ATProto struct {
	core       *core.Core
	log        *zap.SugaredLogger
	identifier string
	password   string
	userAgent  string

	// site.standard
	publicationRecordKey string
	publicationUri       string

	// alpha.arabica.social
	arabicaFilename string
}

func NewATProto(co *core.Core, configMap map[string]any) (server.Plugin, error) {
	config := typed.New(configMap)

	identifier := config.String("identifier")
	if identifier == "" {
		return nil, errors.New("identifier missing")
	}

	password := config.String("password")
	if password == "" {
		return nil, errors.New("password missing")
	}

	publicationRecordKey := config.String("publicationrecordkey")
	if publicationRecordKey == "" {
		return nil, errors.New("publicationRecordKey missing")
	}

	at := &ATProto{
		core:                 co,
		userAgent:            fmt.Sprintf("eagle/%s", co.BaseURL().String()),
		identifier:           identifier,
		password:             password,
		log:                  log.S().Named("atproto"),
		publicationRecordKey: publicationRecordKey,
		arabicaFilename:      config.String("arabicafilename"),
	}

	return at, at.init(co)
}

func (at *ATProto) init(co *core.Core) error {
	ctx := context.Background()

	client, err := at.getClient(ctx)
	if err != nil {
		return err
	}

	return at.initStandardPublication(ctx, client, co)
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
		Host:      apiUrl,
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

func (at *ATProto) getSyndications(e *core.Entry) ([]syntax.ATURI, *syntax.ATURI, error) {
	var documentURI *syntax.ATURI
	feedPostsURIs := []syntax.ATURI{}

	for _, syndication := range e.Syndications {
		if !strings.HasPrefix(syndication, "at://") {
			continue
		}

		uri, err := syntax.ParseATURI(syndication)
		if err != nil {
			return nil, nil, err
		}

		if uri.Collection() == "app.bsky.feed.post" {
			feedPostsURIs = append(feedPostsURIs, uri)
		}

		if uri.Collection() == "site.standard.document" {
			documentURI = &uri
		}

	}

	return feedPostsURIs, documentURI, nil
}

func (at *ATProto) IsSyndicated(e *core.Entry) bool {
	feedPostsURIs, documentURI, err := at.getSyndications(e)
	if err != nil {
		return false
	}
	return len(feedPostsURIs) > 0 || documentURI != nil
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
	feedPostsURIs, documentURI, err := at.getSyndications(e)
	if err != nil {
		return err
	}

	client, err := at.getClient(ctx)
	if err != nil {
		return err
	}

	if e.Deleted() || e.Draft {
		if documentURI != nil {
			err = at.deleteStandardDocument(ctx, client, *documentURI)
			if err != nil {
				return err
			}

			e.Syndications = lo.Without(e.Syndications, documentURI.String())
		}

		err = at.deleteBlueskyPosts(ctx, client, feedPostsURIs)
		if err != nil {
			return err
		}

		e.Syndications = lo.Without(e.Syndications, lo.Map(feedPostsURIs, func(uri syntax.ATURI, i int) string {
			return uri.String()
		})...)

		return nil
	}

	posts, err := at.getBlueskyPosts(ctx, client, feedPostsURIs)
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
			post, err = at.createPublishBlueskyPost(ctx, client, e, sctx)
			if err != nil {
				return err
			}

			e.Syndications = append(e.Syndications, post.uri)
		}

		// Upsert standard.site document to ensure that it is up to date (tags, content,
		// link to Bluesky post, etc).
		documentUriStr, err := at.upsertStandardDocument(ctx, client, documentURI, e, post)
		if err != nil {
			return err
		}

		if documentURI == nil {
			// Only add the syndication if we didn't have a documentURI before, otherwise it means we already had the syndication and we just updated the record.
			e.Syndications = append(e.Syndications, documentUriStr)
		}

		return nil
	}

	if lo.Contains(e.Categories, "photos") {
		if len(posts) > 0 {
			// Existing Bluesky posts are not updated to avoid overwriting custom posts.
			return nil
		}

		posts, err := at.createPublishBlueskyPostThread(ctx, client, e, sctx)
		if err != nil {
			return err
		}

		e.Syndications = append(e.Syndications, lo.Map(posts, func(post *blueskyPost, i int) string {
			return post.uri
		})...)

		return nil
	}

	return errors.New("atproto syndication only supports writings and photos categories")
}

func (at *ATProto) DailyCron() error {
	return at.UpdateCoffee(context.Background())
}
