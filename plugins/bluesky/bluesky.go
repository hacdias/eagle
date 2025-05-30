package bluesky

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/atproto/syntax"
	lexutil "github.com/bluesky-social/indigo/lex/util"
	"github.com/bluesky-social/indigo/xrpc"
	"github.com/karlseguin/typed"
	"go.hacdias.com/eagle/core"
	"go.hacdias.com/eagle/log"
	"go.hacdias.com/eagle/server"
	"go.hacdias.com/indielib/micropub"
	"go.uber.org/zap"
)

var (
	_ server.SyndicationPlugin = &Bluesky{}
)

const (
	apiUrl            = "https://bsky.social"
	appUrl            = "https://bsky.app"
	maximumCharacters = 300
)

func init() {
	server.RegisterPlugin("bluesky", NewBluesky)
}

type Bluesky struct {
	log        *zap.SugaredLogger
	identifier string
	password   string
	userAgent  string
}

func NewBluesky(co *core.Core, configMap map[string]any) (server.Plugin, error) {
	config := typed.New(configMap)

	identifier := config.String("identifier")
	if identifier == "" {
		return nil, errors.New("identifier missing")
	}

	password := config.String("password")
	if password == "" {
		return nil, errors.New("password missing")
	}

	return &Bluesky{
		userAgent:  fmt.Sprintf("eagle/%s", co.BaseURL().String()),
		identifier: identifier,
		password:   password,
		log:        log.S().Named("bluesky"),
	}, nil
}

func (m *Bluesky) Syndication() micropub.Syndication {
	return micropub.Syndication{
		UID:  "bluesky",
		Name: "Bluesky",
	}
}

func (m *Bluesky) getClient(ctx context.Context) (*xrpc.Client, error) {
	client := &xrpc.Client{
		Host:      apiUrl,
		UserAgent: &m.userAgent,
	}

	sess, err := atproto.ServerCreateSession(context.Background(), client, &atproto.ServerCreateSession_Input{
		Identifier: m.identifier,
		Password:   m.password,
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

func (m *Bluesky) extractID(urlStr string) (string, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}

	parts := strings.Split(u.Path, "/")
	if len(parts) != 5 {
		return "", fmt.Errorf("expected url to have 5 parts, has %d", len(parts))
	}

	return parts[4], nil
}

func (m *Bluesky) getSyndication(e *core.Entry) (string, string, error) {
	syndications := typed.New(e.Other).Strings(server.SyndicationField)
	for _, urlStr := range syndications {
		if strings.HasPrefix(urlStr, appUrl) {
			id, err := m.extractID(urlStr)
			return urlStr, id, err
		}
	}

	return "", "", nil
}

func (m *Bluesky) IsSyndicated(e *core.Entry) bool {
	_, id, err := m.getSyndication(e)
	if err != nil {
		return false
	}
	return id != ""
}

func (b *Bluesky) uploadImage(ctx context.Context, xrpcc *xrpc.Client, photo *server.Photo) (*lexutil.LexBlob, error) {
	resp, err := atproto.RepoUploadBlob(ctx, xrpcc, bytes.NewReader(photo.Data))
	if err != nil {
		return nil, err
	}

	return &lexutil.LexBlob{
		Ref:      resp.Blob.Ref,
		MimeType: photo.MimeType,
		Size:     resp.Blob.Size,
	}, nil
}

func (b *Bluesky) uploadPhotos(ctx context.Context, xrpcc *xrpc.Client, photos []*server.Photo) []*bsky.EmbedImages_Image {
	embeddings := []*bsky.EmbedImages_Image{}

	for i, photo := range photos {
		if i >= 4 {
			break
		}

		blob, err := b.uploadImage(ctx, xrpcc, photo)
		if err != nil {
			b.log.Warnw("photo upload failed", "mimetype", photo.MimeType, "err", err)
			continue
		}

		embeddings = append(embeddings, &bsky.EmbedImages_Image{
			Image: blob,
			Alt:   photo.Title,
		})
	}

	return embeddings
}

func (b *Bluesky) deletePost(ctx context.Context, xrpcc *xrpc.Client, recordKey string) error {
	_, err := atproto.RepoDeleteRecord(ctx, xrpcc, &atproto.RepoDeleteRecord_Input{
		Collection: "app.bsky.feed.post",
		Repo:       xrpcc.Auth.Did,
		Rkey:       recordKey,
	})

	return err
}

func (b *Bluesky) getPost(ctx context.Context, xrpcc *xrpc.Client, recordKey string) (*bsky.FeedPost, *string, error) {
	resp, err := atproto.RepoGetRecord(ctx, xrpcc, "", "app.bsky.feed.post", xrpcc.Auth.Did, recordKey)
	if err != nil {
		return nil, nil, err
	}

	existentPost, ok := resp.Value.Val.(*bsky.FeedPost)
	if !ok {
		return nil, nil, fmt.Errorf("record %s is not a post", recordKey)
	}

	return existentPost, resp.Cid, nil
}

func (b *Bluesky) Syndicate(ctx context.Context, e *core.Entry, sctx *server.SyndicationContext) (string, bool, error) {
	xrpcc, err := b.getClient(ctx)
	if err != nil {
		return "", false, err
	}

	url, recordKey, err := b.getSyndication(e)
	if err != nil {
		return "", false, err
	}

	// Handle deleted or draft entries
	if e.Deleted() || e.Draft {
		if recordKey == "" {
			return "", false, errors.New("cannot syndicate a deleted or draft entry")
		} else {
			return url, true, b.deletePost(ctx, xrpcc, recordKey)
		}
	}

	// Initialize the post
	var post *bsky.FeedPost
	var cid *string
	if recordKey == "" {
		post = &bsky.FeedPost{
			CreatedAt: e.Date.Format(syntax.AtprotoDatetimeLayout),
			Embed:     &bsky.FeedPost_Embed{},
		}

		embeddings := b.uploadPhotos(ctx, xrpcc, sctx.Photos)
		if len(embeddings) > 0 {
			post.Embed.EmbedImages = &bsky.EmbedImages{
				Images: embeddings,
			}
		}
	} else {
		post, cid, err = b.getPost(ctx, xrpcc, recordKey)
		if err != nil {
			return "", false, err
		}

		if post.Embed == nil {
			post.Embed = &bsky.FeedPost_Embed{}
		}
	}

	// Determine how many images are embedded
	imagesCount := 0
	if post.Embed.EmbedImages != nil {
		imagesCount = len(post.Embed.EmbedImages.Images)
	}

	// Get text content and determine if it's too long
	post.Text = e.Status(maximumCharacters, len(e.Photos) > imagesCount)

	if byteStart := bytes.Index([]byte(post.Text), []byte(e.Permalink)); byteStart != -1 {
		// Include facet
		byteStart := bytes.Index([]byte(post.Text), []byte(e.Permalink))
		byteEnd := byteStart + len([]byte(e.Permalink))

		post.Facets = []*bsky.RichtextFacet{
			{
				Features: []*bsky.RichtextFacet_Features_Elem{
					{
						RichtextFacet_Link: &bsky.RichtextFacet_Link{
							Uri: e.Permalink,
						},
					},
				},
				Index: &bsky.RichtextFacet_ByteSlice{
					ByteStart: int64(byteStart),
					ByteEnd:   int64(byteEnd),
				},
			},
		}
	}

	// If there are no images, embed the post link
	if imagesCount == 0 {
		post.Embed = &bsky.FeedPost_Embed{
			EmbedExternal: &bsky.EmbedExternal{
				External: &bsky.EmbedExternal_External{
					Uri:         e.Permalink,
					Title:       e.Title,
					Description: e.Summary(),
				},
			},
		}

		if sctx.Thumbnail != nil {
			blob, err := b.uploadImage(ctx, xrpcc, sctx.Thumbnail)
			if err == nil {
				post.Embed.EmbedExternal.External.Thumb = blob
			}
		}
	}

	// Update or create the post
	if recordKey == "" {
		resp, err := atproto.RepoCreateRecord(ctx, xrpcc, &atproto.RepoCreateRecord_Input{
			Collection: "app.bsky.feed.post",
			Repo:       xrpcc.Auth.Did,
			Record:     &lexutil.LexiconTypeDecoder{Val: post},
		})
		if err != nil {
			return "", false, err
		}

		uri, err := syntax.ParseATURI(resp.Uri)
		if err != nil {
			return "", false, err
		}

		return fmt.Sprintf("%s/profile/%s/post/%s", appUrl, uri.Authority(), uri.RecordKey()), false, nil
	} else {
		_, err := atproto.RepoPutRecord(ctx, xrpcc, &atproto.RepoPutRecord_Input{
			Rkey:       recordKey,
			Collection: "app.bsky.feed.post",
			Repo:       xrpcc.Auth.Did,
			Record:     &lexutil.LexiconTypeDecoder{Val: post},
			SwapRecord: cid,
		})
		if err != nil {
			return "", false, err
		}

		return url, false, nil
	}
}
