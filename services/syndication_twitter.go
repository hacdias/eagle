package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/dghubble/oauth1"
	"github.com/hacdias/eagle/config"
	"github.com/hacdias/eagle/middleware/micropub"
)

type Twitter struct {
	*config.Twitter
	client *http.Client
}

type twitterResp struct {
	ID string `json:"id_str"`
}

func NewTwitter(opts *config.Twitter) *Twitter {
	config := oauth1.NewConfig(opts.Key, opts.Secret)
	token := oauth1.NewToken(opts.Token, opts.TokenSecret)

	client := config.Client(oauth1.NoContext, token)

	return &Twitter{
		Twitter: opts,
		client:  client,
	}
}

func (t *Twitter) Syndicate(entry *HugoEntry, typ micropub.Type, related string) (string, error) {
	switch typ {
	case micropub.TypeReply, micropub.TypeNote, micropub.TypeArticle:
		// ok
	default:
		return "", fmt.Errorf("unsupported post type for twitter: %s", typ)
	}

	status := entry.Content
	if len(entry.Content) > 280 {
		status = strings.TrimSpace(entry.Content[0:230]) + "... " + entry.Permalink
	}

	u, err := url.Parse("https://api.twitter.com/1.1/statuses/update.json")
	if err != nil {
		return "", err
	}

	u.Query().Set("status", status)
	if typ == micropub.TypeReply {
		if strings.HasSuffix(related, "/") {
			related = strings.TrimSuffix(related, "/")
		}
		parts := strings.Split(related, "/")
		u.Query().Set("in_reply_to_status_id", parts[len(parts)-1])
		u.Query().Set("auto_populate_reply_metadata", "true")
		// TODO: add attachment_url for retweet with status
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", err
	}

	res, err := t.client.Do(req)
	if err != nil {
		return "", err
	}

	var tid twitterResp
	err = json.NewDecoder(res.Body).Decode(&tid)
	if err != nil {
		return "", err
	}

	return "https://twitter.com/" + t.User + "/status/" + tid.ID, nil
}

func (t *Twitter) IsRelated(url string) bool {
	return strings.HasPrefix(url, "https://twitter.com")
}
