package atproto

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/bluesky-social/indigo/xrpc"
	"go.hacdias.com/eagle/core"
	"go.hacdias.com/eagle/server"
)

func (at *ATProto) initStandardPublication(ctx context.Context, xrpcc *xrpc.Client, co *core.Core) error {
	at.log.Infow("repository information found", "did", xrpcc.Auth.Did)

	record := map[string]any{
		"$type": "site.standard.publication",
		"url":   strings.TrimSuffix(co.BaseURL().String(), "/"),
		"name":  co.SiteConfig().Title,
		"preferences": map[string]any{
			"showInDiscover": true,
		},
	}

	if co.SiteConfig().Params.Site.Description != "" {
		record["description"] = co.SiteConfig().Params.Site.Description
	}

	uri, err := upsertRecord(ctx, xrpcc, "site.standard.publication", at.publicationRecordKey, record)
	if err != nil {
		return err
	}

	at.log.Infow("publication record upserted", "uri", uri)
	at.publicationUri = uri
	return nil
}

func (at *ATProto) upsertStandardDocument(ctx context.Context, client *xrpc.Client, documentUri *syntax.ATURI, e *core.Entry, post *blueskyPost) (string, error) {
	recordKey := ""
	if documentUri != nil {
		recordKey = documentUri.RecordKey().String()
	}

	// https://standard.site/
	record := map[string]any{
		"$type":       "site.standard.document",
		"site":        at.publicationUri,
		"path":        e.RelPermalink,
		"title":       e.Title,
		"publishedAt": e.Date.Format(time.RFC3339),
	}

	if post != nil {
		record["bskyPostRef"] = map[string]any{
			"$type": "com.atproto.repo.strongRef",
			"uri":   post.uri,
			"cid":   post.cid,
		}

		if post.Embed != nil && post.Embed.EmbedExternal != nil && post.Embed.EmbedExternal.External != nil && post.Embed.EmbedExternal.External.Thumb != nil {
			record["coverImage"] = post.Embed.EmbedExternal.External.Thumb
		}
	}

	if len(e.Tags) > 0 {
		record["tags"] = e.Tags
	}

	if textContent := e.TextContent(); textContent != "" {
		record["textContent"] = textContent
	}

	if e.Description != "" {
		record["description"] = e.Description
	} else if summary := e.Summary(); summary != "" {
		record["description"] = summary
	}

	if !e.Lastmod.IsZero() {
		record["updatedAt"] = e.Date.Format(time.RFC3339)
	}

	at.log.Infow("upserting site.standard.document", "rkey", recordKey, "record", record)
	documentUriStr, err := upsertRecord(ctx, client, "site.standard.document", recordKey, record)
	if err != nil {
		return "", fmt.Errorf("failed to upsert site.standard.document record: %w", err)
	}

	return documentUriStr, nil

}

func (at *ATProto) deleteStandardDocument(ctx context.Context, client *xrpc.Client, uri syntax.ATURI) error {
	at.log.Infow("deleting site.standard.document", "rkey", uri.RecordKey().String())
	return deleteRecord(ctx, client, "site.standard.document", uri.RecordKey().String())
}

func (at *ATProto) HandlerRoute() string {
	return wellKnownStandardPublication
}

func (at *ATProto) Handler(w http.ResponseWriter, r *http.Request, utils *server.PluginWebUtilities) {
	_, _ = w.Write([]byte(at.publicationUri))
}

const wellKnownStandardPublication = "/.well-known/site.standard.publication"
