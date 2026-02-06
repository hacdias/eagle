package standardsite

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/api/agnostic"
	"github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/xrpc"
	"github.com/karlseguin/typed"
	"go.hacdias.com/eagle/core"
	"go.hacdias.com/eagle/log"
	"go.hacdias.com/eagle/server"
	"go.hacdias.com/indielib/micropub"
	"go.uber.org/zap"
)

var (
	_ server.HandlerPlugin     = &StandardSite{}
	_ server.SyndicationPlugin = &StandardSite{}
)

const (
	apiUrl = "https://bsky.social"
)

func init() {
	server.RegisterPlugin("standardsite", NewStandardSite)
}

type StandardSite struct {
	co         *core.Core
	log        *zap.SugaredLogger
	identifier string
	password   string
	userAgent  string

	repositoryDid  string
	publicationUri string
}

func NewStandardSite(co *core.Core, configMap map[string]any) (server.Plugin, error) {
	config := typed.New(configMap)

	identifier := config.String("identifier")
	if identifier == "" {
		return nil, errors.New("identifier missing")
	}

	password := config.String("password")
	if password == "" {
		return nil, errors.New("password missing")
	}

	at := &StandardSite{
		co:         co,
		userAgent:  fmt.Sprintf("eagle/%s", co.BaseURL().String()),
		identifier: identifier,
		password:   password,
		log:        log.S().Named("standard.site"),
	}

	return at, at.init(context.Background())
}

func (at *StandardSite) getClient(ctx context.Context) (*xrpc.Client, error) {
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

func (at *StandardSite) init(ctx context.Context) error {
	xrpcc, err := at.getClient(ctx)
	if err != nil {
		return err
	}

	url := at.co.BaseURL()

	result, err := agnostic.RepoPutRecord(ctx, xrpcc, &agnostic.RepoPutRecord_Input{
		Collection: "site.standard.publication",
		Repo:       xrpcc.Auth.Did,
		Rkey:       url.Hostname(),
		Record: map[string]any{
			"$type":       "site.standard.publication",
			"url":         strings.TrimSuffix(url.String(), "/"),
			"name":        at.co.SiteConfig().Params.Site.Description,
			"description": at.co.SiteConfig().Title,
			"preferences": map[string]any{
				"showInDiscover": true,
			},
		},
	})
	if err != nil {
		return err
	}

	at.log.Infow("updated publication record", "uri", result.Uri)

	at.repositoryDid = xrpcc.Auth.Did
	at.publicationUri = result.Uri

	return nil
}

func (at *StandardSite) getSyndication(e *core.Entry) (string, string, error) {
	prefix := "at://" + at.repositoryDid + "/site.standard.document/"

	syndications := typed.New(e.Other).Strings(server.SyndicationField)
	for _, urlStr := range syndications {
		if strings.HasPrefix(urlStr, prefix) {
			id := strings.TrimPrefix(urlStr, prefix)

			if id == "" {
				return "", "", fmt.Errorf("invalid syndication url: %s", urlStr)
			}

			return urlStr, id, nil
		}
	}

	return "", "", nil
}

func (at *StandardSite) IsSyndicated(e *core.Entry) bool {
	_, _, err := at.getSyndication(e)
	return err == nil
}

func (at *StandardSite) Syndication() micropub.Syndication {
	return micropub.Syndication{
		UID:  "standard.site",
		Name: "Standard.site",
	}
}

func (at *StandardSite) Syndicate(ctx context.Context, e *core.Entry, sctx *server.SyndicationContext) (string, bool, error) {
	xrpcc, err := at.getClient(ctx)
	if err != nil {
		return "", false, err
	}

	uri, rkey, err := at.getSyndication(e)
	if err != nil {
		return "", false, err
	}

	// Handle deleted or draft entries
	if e.Deleted() || e.Draft {
		if rkey == "" {
			return "", false, errors.New("cannot syndicate a deleted or draft entry")
		} else {
			_, err = atproto.RepoDeleteRecord(ctx, xrpcc, &atproto.RepoDeleteRecord_Input{
				Collection: "site.standard.publication",
				Repo:       xrpcc.Auth.Did,
				Rkey:       rkey,
			})

			return uri, err == nil, err
		}
	}

	record := map[string]any{
		"$type":       "site.standard.document",
		"site":        at.publicationUri,
		"path":        e.RelPermalink,
		"title":       e.Title,
		"publishedAt": e.Date.Format(time.RFC3339),
	}

	if e.Description != "" {
		record["description"] = e.Description
	}

	if !e.Lastmod.IsZero() {
		record["updatedAt"] = e.Date.Format(time.RFC3339)
	}

	if rkey == "" {
		result, err := agnostic.RepoCreateRecord(ctx, xrpcc, &agnostic.RepoCreateRecord_Input{
			Collection: "site.standard.document",
			Repo:       xrpcc.Auth.Did,
			Record:     record,
		})
		if err != nil {
			return "", false, err
		}
		return result.Uri, false, nil
	} else {
		result, err := agnostic.RepoPutRecord(ctx, xrpcc, &agnostic.RepoPutRecord_Input{
			Collection: "site.standard.document",
			Repo:       xrpcc.Auth.Did,
			Rkey:       rkey,
			Record:     record,
		})
		if err != nil {
			return "", false, err
		}
		return result.Uri, false, nil
	}
}

func (at *StandardSite) HandlerRoute() string {
	return wellKnownStandardPublication
}

func (at *StandardSite) Handler(w http.ResponseWriter, r *http.Request, utils *server.PluginWebUtilities) {
	_, _ = w.Write([]byte(at.publicationUri))
}

const wellKnownStandardPublication = "/.well-known/site.standard.publication"
