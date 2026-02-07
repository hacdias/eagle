package atproto

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/xrpc"
	"go.hacdias.com/eagle/core"
	"go.hacdias.com/eagle/server"
)

func (at *ATProto) initStandardPublication(ctx context.Context, xrpcc *xrpc.Client, co *core.Core) error {
	at.log.Infof("repository did is %s", xrpcc.Auth.Did)

	record := map[string]any{
		"$type":       "site.standard.publication",
		"url":         strings.TrimSuffix(co.BaseURL().String(), "/"),
		"name":        co.SiteConfig().Params.Site.Description,
		"description": co.SiteConfig().Title,
		"preferences": map[string]any{
			"showInDiscover": true,
		},
	}

	uri, err := upsertRecord(ctx, xrpcc, "site.standard.publication", at.publicationRecordKey, record)
	if err != nil {
		return err
	}

	at.log.Infow("publication record upserted", "uri", uri)
	at.publicationUri = uri
	return nil
}

func (at *ATProto) upsertStandardDocument(ctx context.Context, client *xrpc.Client, recordKey string, e *core.Entry, post *blueskyPost) (string, error) {
	// https://standard.site/
	record := map[string]any{
		"$type":       "site.standard.document",
		"site":        at.publicationUri,
		"path":        e.RelPermalink,
		"title":       e.Title,
		"publishedAt": e.Date.Format(time.RFC3339),
		"bskyPostRef": map[string]any{
			"$type": "com.atproto.repo.strongRef",
			"uri":   post.uri,
			"cid":   post.cid,
		},
		"tags": post.Tags,
		// "textContent"
	}

	if post.Embed != nil && post.Embed.EmbedExternal != nil && post.Embed.EmbedExternal.External != nil && post.Embed.EmbedExternal.External.Thumb != nil {
		record["coverImage"] = post.Embed.EmbedExternal.External.Thumb
	}

	if e.Description != "" {
		record["description"] = e.Description
	}

	if !e.Lastmod.IsZero() {
		record["updatedAt"] = e.Date.Format(time.RFC3339)
	}

	uri, err := upsertRecord(ctx, client, "site.standard.document", recordKey, record)
	if err != nil {
		return "", err
	}

	return uri, nil

}

func (at *ATProto) deleteStandardDocument(ctx context.Context, client *xrpc.Client, recordKey string) error {
	return deleteRecord(ctx, client, "site.standard.publication", recordKey)
}

func (at *ATProto) HandlerRoute() string {
	return wellKnownStandardPublication
}

func (at *ATProto) Handler(w http.ResponseWriter, r *http.Request, utils *server.PluginWebUtilities) {
	_, _ = w.Write([]byte(at.publicationUri))
}

const wellKnownStandardPublication = "/.well-known/site.standard.publication"
