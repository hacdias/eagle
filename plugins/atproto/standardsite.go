package atproto

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/atproto/syntax"
	lexutil "github.com/bluesky-social/indigo/lex/util"
	"github.com/bluesky-social/indigo/xrpc"
	"github.com/gabriel-vasile/mimetype"
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
	Icon        string
	Preferences struct {
		ShowInDiscover bool
	}
	BasicTheme *standardSiteBasicTheme
}

func (at *ATProto) uploadIcon(ctx context.Context, client *xrpc.Client) (*lexutil.LexBlob, error) {
	data, err := at.core.ReadFile(at.standardSite.Icon)
	if err != nil {
		return nil, fmt.Errorf("failed to read icon file %q: %w", at.standardSite.Icon, err)
	}

	if len(data) > 1_000_000 {
		return nil, fmt.Errorf("icon file %q exceeds 1MB limit (%d bytes)", at.standardSite.Icon, len(data))
	}

	mime := mimetype.Detect(data)
	if !strings.HasPrefix(mime.String(), "image/") {
		return nil, fmt.Errorf("icon file %q has non-image mimetype %q", at.standardSite.Icon, mime.String())
	}

	resp, err := atproto.RepoUploadBlob(ctx, client, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	return &lexutil.LexBlob{
		Ref:      resp.Blob.Ref,
		MimeType: mime.String(),
		Size:     resp.Blob.Size,
	}, nil
}

func (at *ATProto) initStandardPublication(ctx context.Context, client *xrpc.Client) error {
	at.log.Infow("repository information found", "did", client.Auth.Did)

	record := map[string]any{
		"$type": "site.standard.publication",
		"url":   strings.TrimSuffix(at.core.BaseURL().String(), "/"),
		"name":  at.core.SiteConfig().Title,
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

	if at.core.SiteConfig().Params.Site.Description != "" {
		record["description"] = at.core.SiteConfig().Params.Site.Description
	}

	if at.standardSite.Icon != "" {
		iconBlob, err := at.uploadIcon(ctx, client)
		if err != nil {
			return fmt.Errorf("failed to upload icon: %w", err)
		}
		record["icon"] = iconBlob
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
