package standardsite

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
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

	at.log.Infof("repository did is %s", xrpcc.Auth.Did)
	at.repositoryDid = xrpcc.Auth.Did

	url := at.co.BaseURL()
	recordKey := url.Hostname()

	expectedRecord := map[string]any{
		"$type":       "site.standard.publication",
		"url":         strings.TrimSuffix(url.String(), "/"),
		"name":        at.co.SiteConfig().Params.Site.Description,
		"description": at.co.SiteConfig().Title,
		"preferences": map[string]any{
			"showInDiscover": true,
		},
	}

	getResult, err := agnostic.RepoGetRecord(ctx, xrpcc, "", "site.standard.publication", at.repositoryDid, recordKey)
	if err == nil {
		at.log.Info("publication record found")
		var currentRecord map[string]any
		err = json.Unmarshal(*getResult.Value, &currentRecord)
		if err != nil {
			return err
		}

		if reflect.DeepEqual(expectedRecord, currentRecord) {
			at.log.Infow("document record is up to date", "uri", getResult.Uri)
			at.publicationUri = getResult.Uri
			return nil
		}
	}

	putRecord, err := agnostic.RepoPutRecord(ctx, xrpcc, &agnostic.RepoPutRecord_Input{
		Collection: "site.standard.publication",
		Repo:       at.repositoryDid,
		Rkey:       recordKey,
		Record:     expectedRecord,
	})
	if err != nil {
		return err
	}

	at.log.Infow("publication record updated", "uri", putRecord.Uri)
	at.publicationUri = putRecord.Uri
	return nil
}

func (at *StandardSite) getSyndication(e *core.Entry) (string, string, error) {
	prefix := "at://" + at.repositoryDid + "/site.standard.document/"

	syndications := typed.New(e.Other).Strings(server.SyndicationField)
	for _, uri := range syndications {
		if strings.HasPrefix(uri, prefix) {
			recordKey := strings.TrimPrefix(uri, prefix)
			if recordKey == "" {
				return "", "", fmt.Errorf("invalid syndication uri: %s", uri)
			}

			return uri, recordKey, nil
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

	uri, recordKey, err := at.getSyndication(e)
	if err != nil {
		return "", false, err
	}

	// Handle deleted or draft entries
	if e.Deleted() || e.Draft {
		if recordKey == "" {
			return "", false, errors.New("cannot syndicate a deleted or draft entry")
		} else {
			_, err = atproto.RepoDeleteRecord(ctx, xrpcc, &atproto.RepoDeleteRecord_Input{
				Collection: "site.standard.publication",
				Repo:       at.repositoryDid,
				Rkey:       recordKey,
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

	// Create new record
	if recordKey == "" {
		result, err := agnostic.RepoCreateRecord(ctx, xrpcc, &agnostic.RepoCreateRecord_Input{
			Collection: "site.standard.document",
			Repo:       at.repositoryDid,
			Record:     record,
		})
		if err != nil {
			return "", false, err
		}
		return result.Uri, false, nil
	}

	// Get existing record and check if needs to be updated
	getResult, err := agnostic.RepoGetRecord(ctx, xrpcc, "", "site.standard.document", at.repositoryDid, recordKey)
	if err == nil {
		at.log.Info("document record found")
		var currentRecord map[string]any
		err = json.Unmarshal(*getResult.Value, &currentRecord)
		if err != nil {
			return "", false, err
		}

		if reflect.DeepEqual(record, currentRecord) {
			at.log.Infow("publication record is up to date", "uri", getResult.Uri)
			return getResult.Uri, false, nil
		}
	}

	// Update existing record
	putResult, err := agnostic.RepoPutRecord(ctx, xrpcc, &agnostic.RepoPutRecord_Input{
		Collection: "site.standard.document",
		Repo:       at.repositoryDid,
		Rkey:       recordKey,
		Record:     record,
	})
	if err != nil {
		return "", false, err
	}

	return putResult.Uri, false, nil

}

func (at *StandardSite) HandlerRoute() string {
	return wellKnownStandardPublication
}

func (at *StandardSite) Handler(w http.ResponseWriter, r *http.Request, utils *server.PluginWebUtilities) {
	_, _ = w.Write([]byte(at.publicationUri))
}

const wellKnownStandardPublication = "/.well-known/site.standard.publication"
