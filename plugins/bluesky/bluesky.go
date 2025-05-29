package bluesky

import (
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
	"go.hacdias.com/eagle/server"
	"go.hacdias.com/indielib/micropub"
)

var (
	_ server.SyndicationPlugin = &Bluesky{}
)

const (
	apiUrl = "https://bsky.social"
	appUrl = "https://bsky.app"
)

func init() {
	server.RegisterPlugin("bluesky", NewBluesky)
}

type Bluesky struct {
	identifier string
	password   string
	userAgent  string
}

func NewBluesky(co *core.Core, configMap map[string]interface{}) (server.Plugin, error) {
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

func (m *Bluesky) Syndicate(ctx context.Context, e *core.Entry, photos []server.Photo) (string, bool, error) {
	xrpcc, err := m.getClient(ctx)
	if err != nil {
		return "", false, err
	}

	url, id, err := m.getSyndication(e)
	if err != nil {
		return "", false, err
	}

	if id != "" {
		if e.Deleted() || e.Draft {
			_, err := atproto.RepoDeleteRecord(ctx, xrpcc, &atproto.RepoDeleteRecord_Input{
				Collection: "app.bsky.feed.post",
				Repo:       xrpcc.Auth.Did,
				Rkey:       id,
			})

			return url, true, err
		}
	}

	text := []byte(e.Title + "\n\n" + e.Permalink)
	byteStart := int64(len(e.Title) + 2)
	byteEnd := byteStart + int64(len([]byte(e.Permalink)))

	post := bsky.FeedPost{
		Text:      string(text),
		CreatedAt: e.Date.Format(syntax.AtprotoDatetimeLayout),
		Facets: []*bsky.RichtextFacet{
			{
				Features: []*bsky.RichtextFacet_Features_Elem{
					{
						RichtextFacet_Link: &bsky.RichtextFacet_Link{
							Uri: e.Permalink,
						},
					},
				},
				Index: &bsky.RichtextFacet_ByteSlice{
					ByteStart: byteStart,
					ByteEnd:   byteEnd,
				},
			},
		},
	}

	resp, err := atproto.RepoCreateRecord(ctx, xrpcc, &atproto.RepoCreateRecord_Input{
		Collection: "app.bsky.feed.post",
		Repo:       xrpcc.Auth.Did,
		Record:     &lexutil.LexiconTypeDecoder{Val: &post},
	})
	if err != nil {
		return "", false, err
	}

	uri, err := syntax.ParseATURI(resp.Uri)
	if err != nil {
		return "", false, err
	}

	return fmt.Sprintf("%s/profile/%s/post/%s", appUrl, uri.Authority(), uri.RecordKey()), false, nil

}
