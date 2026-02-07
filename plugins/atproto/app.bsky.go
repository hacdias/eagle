package atproto

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"net/url"
	"strings"

	"github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/bluesky-social/indigo/lex/util"
	"github.com/bluesky-social/indigo/xrpc"
	"go.hacdias.com/eagle/core"
	"go.hacdias.com/eagle/server"
)

type blueskyPost struct {
	*bsky.FeedPost
	cid         string
	uri         string
	syndication string
}

func (at *ATProto) blueskySyndicationToPostId(urlStr string) (string, error) {
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

func (at *ATProto) blueskyUriToSyndication(raw string) (string, error) {
	uri, err := syntax.ParseATURI(raw)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/profile/%s/post/%s", appUrl, at.identifier, uri.RecordKey()), nil
}

func (at *ATProto) deleteBlueskyPost(ctx context.Context, client *xrpc.Client, recordKey string) error {
	return deleteRecord(ctx, client, "app.bsky.feed.post", recordKey)
}

func (at *ATProto) uploadBlueskyPhotos(ctx context.Context, client *xrpc.Client, photos []*server.Photo) ([]*bsky.EmbedImages_Image, error) {
	embeddings := []*bsky.EmbedImages_Image{}

	for _, photo := range photos {
		blob, err := uploadPhoto(ctx, client, photo)
		if err != nil {
			return nil, err
		}

		embedding := &bsky.EmbedImages_Image{
			Image: blob,
			Alt:   photo.Title,
		}

		if photo.Width > 0 && photo.Height > 0 {
			embedding.AspectRatio = &bsky.EmbedDefs_AspectRatio{
				Width:  int64(photo.Width),
				Height: int64(photo.Height),
			}
		}

		embeddings = append(embeddings, embedding)
	}

	return embeddings, nil
}

func (at *ATProto) getBlueskyPost(ctx context.Context, client *xrpc.Client, recordKey string) (*blueskyPost, error) {
	response, err := atproto.RepoGetRecord(ctx, client, "", "app.bsky.feed.post", client.Auth.Did, recordKey)
	if err != nil {
		return nil, err
	}

	post, ok := response.Value.Val.(*bsky.FeedPost)
	if !ok {
		return nil, fmt.Errorf("record %s is not a post", recordKey)
	}

	if response.Cid == nil {
		return nil, fmt.Errorf("record %s has no cid", recordKey)
	}

	syndication, err := at.blueskyUriToSyndication(response.Uri)
	if err != nil {
		return nil, err
	}

	return &blueskyPost{
		FeedPost:    post,
		cid:         *response.Cid,
		uri:         response.Uri,
		syndication: syndication,
	}, nil
}

func detectPermalinkFacet(post *bsky.FeedPost, permalinkStr string) {
	permalink := []byte(permalinkStr)

	if byteStart := bytes.Index([]byte(post.Text), permalink); byteStart != -1 {
		byteStart := bytes.Index([]byte(post.Text), permalink)
		byteEnd := byteStart + len(permalink)

		if post.Facets == nil {
			post.Facets = []*bsky.RichtextFacet{}
		}

		post.Facets = append(post.Facets, &bsky.RichtextFacet{
			Features: []*bsky.RichtextFacet_Features_Elem{
				{
					RichtextFacet_Link: &bsky.RichtextFacet_Link{
						Uri: permalinkStr,
					},
				},
			},
			Index: &bsky.RichtextFacet_ByteSlice{
				ByteStart: int64(byteStart),
				ByteEnd:   int64(byteEnd),
			},
		})
	}
}

func createBlueskyRecord(ctx context.Context, client *xrpc.Client, collection, repo string, record *util.LexiconTypeDecoder) (*atproto.RepoCreateRecord_Output, error) {
	return atproto.RepoCreateRecord(ctx, client, &atproto.RepoCreateRecord_Input{
		Collection: collection,
		Repo:       repo,
		Record:     record,
	})

}

func (at *ATProto) createPublishBlueskyPost(ctx context.Context, client *xrpc.Client, e *core.Entry, sctx *server.SyndicationContext) (*blueskyPost, error) {
	post := &bsky.FeedPost{
		CreatedAt: e.Date.Format(syntax.AtprotoDatetimeLayout),
		Text:      e.Title + " " + e.Permalink,
		Tags:      e.Taxonomy("tags"),
		Embed: &bsky.FeedPost_Embed{
			EmbedExternal: &bsky.EmbedExternal{
				External: &bsky.EmbedExternal_External{
					Uri:         e.Permalink,
					Title:       e.Title,
					Description: e.Summary(),
				},
			},
		},
	}

	detectPermalinkFacet(post, e.Permalink)

	if sctx.Thumbnail != nil {
		blob, err := uploadPhoto(ctx, client, sctx.Thumbnail)
		if err != nil {
			return nil, err
		}
		post.Embed.EmbedExternal.External.Thumb = blob
	}

	record, err := createBlueskyRecord(ctx, client, "app.bsky.feed.post", client.Auth.Did, &util.LexiconTypeDecoder{Val: post})
	if err != nil {
		return nil, err
	}

	syndication, err := at.blueskyUriToSyndication(record.Uri)
	if err != nil {
		return nil, err
	}

	return &blueskyPost{
		FeedPost:    post,
		cid:         record.Cid,
		uri:         record.Uri,
		syndication: syndication,
	}, nil
}

func (at *ATProto) createPublishBlueskyPostThread(ctx context.Context, xrpcc *xrpc.Client, e *core.Entry, sctx *server.SyndicationContext) ([]*blueskyPost, error) {
	// Infer how many posts needed from photos count
	postsNeeded := 1
	if len(sctx.Photos) > 0 {
		postsNeeded = int(math.Ceil(float64(len(sctx.Photos)) / maximumPhotos))
	}

	statuses := e.Statuses(maximumCharacters, postsNeeded)

	embeddings, err := at.uploadBlueskyPhotos(ctx, xrpcc, sctx.Photos)
	if err != nil {
		return nil, err
	}

	posts := []*blueskyPost{}
	for i := 0; i < postsNeeded; i++ {

		text := ""
		if i < len(statuses) {
			text = statuses[i]
		}

		post := &bsky.FeedPost{
			CreatedAt: e.Date.Format(syntax.AtprotoDatetimeLayout),
			Text:      text,
			Embed:     &bsky.FeedPost_Embed{},
			Tags:      e.Taxonomy("tags"),
		}

		embeddingsStart := i * 4
		embeddingsEnd := (i + 1) * 4
		if embeddingsEnd > len(embeddings) {
			embeddingsEnd = len(embeddings)
		}

		post.Embed.EmbedImages = &bsky.EmbedImages{
			Images: embeddings[embeddingsStart:embeddingsEnd],
		}

		detectPermalinkFacet(post, e.Permalink)

		if i != 0 {
			post.Reply = &bsky.FeedPost_ReplyRef{
				Root: &atproto.RepoStrongRef{
					Uri: posts[0].uri,
					Cid: posts[0].cid,
				},
				Parent: &atproto.RepoStrongRef{
					Uri: posts[i-1].uri,
					Cid: posts[i-1].cid,
				},
			}
		}

		record, err := createBlueskyRecord(ctx, xrpcc, "app.bsky.feed.post", xrpcc.Auth.Did, &util.LexiconTypeDecoder{Val: post})
		if err != nil {
			return nil, err
		}

		syndication, err := at.blueskyUriToSyndication(record.Uri)
		if err != nil {
			return nil, err
		}

		posts = append(posts, &blueskyPost{
			FeedPost:    post,
			cid:         record.Cid,
			uri:         record.Uri,
			syndication: syndication,
		})
	}

	return posts, nil
}
