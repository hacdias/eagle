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

type standardSiteBasicTheme struct {
	Background       [3]int
	Foreground       [3]int
	Accent           [3]int
	AccentForeground [3]int
}

type standardSite struct {
	RecordKey   string
	Preferences struct {
		ShowInDiscover bool
	}
	BasicTheme *standardSiteBasicTheme
}

func (at *ATProto) initStandardPublication(ctx context.Context, client *xrpc.Client, co *core.Core) error {
	at.log.Infow("repository information found", "did", client.Auth.Did)

	record := map[string]any{
		"$type": "site.standard.publication",
		"url":   strings.TrimSuffix(co.BaseURL().String(), "/"),
		"name":  co.SiteConfig().Title,
		"preferences": map[string]any{
			"showInDiscover": at.standardSite.Preferences.ShowInDiscover,
		},
	}

	if at.standardSite.BasicTheme != nil {
		record["basicTheme"] = map[string]any{
			"$type": "site.standard.theme.basic",
			"background": map[string]any{
				"$type": "site.standard.theme.color#rgb",
				"r":     at.standardSite.BasicTheme.Background[0],
				"g":     at.standardSite.BasicTheme.Background[1],
				"b":     at.standardSite.BasicTheme.Background[2],
			},
			"foreground": map[string]any{
				"$type": "site.standard.theme.color#rgb",
				"r":     at.standardSite.BasicTheme.Foreground[0],
				"g":     at.standardSite.BasicTheme.Foreground[1],
				"b":     at.standardSite.BasicTheme.Foreground[2],
			},
			"accent": map[string]any{
				"$type": "site.standard.theme.color#rgb",
				"r":     at.standardSite.BasicTheme.Accent[0],
				"g":     at.standardSite.BasicTheme.Accent[1],
				"b":     at.standardSite.BasicTheme.Accent[2],
			},
			"accentForeground": map[string]any{
				"$type": "site.standard.theme.color#rgb",
				"r":     at.standardSite.BasicTheme.AccentForeground[0],
				"g":     at.standardSite.BasicTheme.AccentForeground[1],
				"b":     at.standardSite.BasicTheme.AccentForeground[2],
			},
		}
	}

	if co.SiteConfig().Params.Site.Description != "" {
		record["description"] = co.SiteConfig().Params.Site.Description
	}

	uri, err := putRecord(ctx, client, "site.standard.publication", at.standardSite.RecordKey, record)
	if err != nil {
		return err
	}

	at.log.Infow("publication record upserted", "uri", uri)
	at.standardSitePublicationUri = uri
	return nil
}

func (at *ATProto) upsertStandardDocument(ctx context.Context, client *xrpc.Client, documentUri *syntax.ATURI, e *core.Entry, post *blueskyPost) (string, error) {
	// https://standard.site/
	record := map[string]any{
		"$type":       "site.standard.document",
		"site":        at.standardSitePublicationUri,
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

	if !e.LastMod.IsZero() {
		record["updatedAt"] = e.Date.Format(time.RFC3339)
	}

	var (
		documentUriStr string
		err            error
	)

	if documentUri == nil {
		// Generate record key based on the entry's date. Ensures sortability.
		recordKey := syntax.NewTID(e.Date.UnixMicro(), clockId).String()
		at.log.Infow("creating site.standard.document", "rkey", recordKey, "record", record)
		documentUriStr, err = createRecord(ctx, client, "site.standard.document", &recordKey, record)
	} else {
		recordKey := documentUri.RecordKey().String()
		at.log.Infow("updating site.standard.document", "rkey", recordKey, "record", record)
		documentUriStr, err = putRecord(ctx, client, "site.standard.document", recordKey, record)
	}
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
	_, _ = w.Write([]byte(at.standardSitePublicationUri))
}

const wellKnownStandardPublication = "/.well-known/site.standard.publication"
