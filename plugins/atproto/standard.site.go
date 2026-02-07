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
		"$type":       "site.standard.publication",
		"url":         strings.TrimSuffix(co.BaseURL().String(), "/"),
		"name":        co.SiteConfig().Title,
		"description": co.SiteConfig().Params.Site.Description,
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

func (at *ATProto) upsertStandardDocument(ctx context.Context, client *xrpc.Client, documentUri string, e *core.Entry, post *blueskyPost) (string, error) {
	recordKey := ""
	if documentUri != "" {
		uri, err := syntax.ParseATURI(documentUri)
		if err != nil {
			return "", fmt.Errorf("failed to parse site.standard.document URI: %w", err)
		}

		recordKey = uri.RecordKey().String()
	}

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
	}

	if tags := e.Taxonomy("tags"); len(tags) > 0 {
		record["tags"] = tags
	}

	if post.Embed != nil && post.Embed.EmbedExternal != nil && post.Embed.EmbedExternal.External != nil && post.Embed.EmbedExternal.External.Thumb != nil {
		record["coverImage"] = post.Embed.EmbedExternal.External.Thumb
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

	documentUri, err := upsertRecord(ctx, client, "site.standard.document", recordKey, record)
	if err != nil {
		return "", fmt.Errorf("failed to upsert site.standard.document record: %w", err)
	}

	return documentUri, nil

}

func (at *ATProto) deleteStandardDocument(ctx context.Context, client *xrpc.Client, document string) error {
	uri, err := syntax.ParseATURI(document)
	if err != nil {
		return fmt.Errorf("failed to parse site.standard.document URI: %w", err)
	}

	return deleteRecord(ctx, client, "site.standard.publication", uri.RecordKey().String())
}

func (at *ATProto) HandlerRoute() string {
	return wellKnownStandardPublication
}

func (at *ATProto) Handler(w http.ResponseWriter, r *http.Request, utils *server.PluginWebUtilities) {
	_, _ = w.Write([]byte(at.publicationUri))
}

const wellKnownStandardPublication = "/.well-known/site.standard.publication"
