package syndicator

import (
	"encoding/json"
	"fmt"
	"net/http"
	urlpkg "net/url"
	"strings"
	"time"

	"github.com/dghubble/oauth1"
	"github.com/hacdias/eagle/v2/config"
	"github.com/hacdias/eagle/v2/entry"
	"github.com/hacdias/eagle/v2/entry/mf2"
)

type Twitter struct {
	conf   *config.Twitter
	client *http.Client
}

func NewTwitter(opts *config.Twitter) *Twitter {
	config := oauth1.NewConfig(opts.Key, opts.Secret)
	token := oauth1.NewToken(opts.Token, opts.TokenSecret)

	client := config.Client(oauth1.NoContext, token)
	client.Timeout = time.Second * 30

	return &Twitter{
		conf:   opts,
		client: client,
	}
}

func (t *Twitter) Syndicate(entry *entry.Entry) (url string, err error) {
	mm := entry.Helper()
	typ := mm.PostType()
	urlStr := mm.String(mm.TypeProperty())

	if typ == mf2.TypeLike {
		id, err := t.idFromUrl(urlStr)
		if err != nil {
			return "", err
		}

		return t.like(id)
	}

	if typ == mf2.TypeRepost {
		id, err := t.idFromUrl(urlStr)
		if err != nil {
			return "", err
		}

		return t.repost(id)
	}

	var replyUrl string
	if typ == mf2.TypeReply {
		replyUrl = urlStr
	}

	status := entry.TextContent()
	if len(status) > 280 {
		status = strings.TrimSpace(status[0:275-len(entry.Permalink)]) + "... " + entry.Permalink
	}

	return t.tweet(status, replyUrl)
}

func (t *Twitter) IsByContext(entry *entry.Entry) bool {
	mm := entry.Helper()
	typ := mm.PostType()

	switch typ {
	case mf2.TypeReply, mf2.TypeLike, mf2.TypeRepost:
	default:
		return false
	}

	urlStr := mm.String(mm.TypeProperty())
	url, err := urlpkg.Parse(urlStr)
	if err != nil {
		return false
	}

	return strings.Contains(url.Host, "twitter.com")
}

func (t *Twitter) Name() string {
	return fmt.Sprintf("Twitter (%s)", t.conf.User)
}

func (t *Twitter) Identifier() string {
	return fmt.Sprintf("twitter-%s", t.conf.User)
}

func (t *Twitter) tweet(status, replyTo string) (string, error) {
	values := urlpkg.Values{}
	values.Set("status", status)

	if replyTo != "" {
		id, err := t.idFromUrl(replyTo)
		if err != nil {
			return "", err
		}
		values.Set("in_reply_to_status_id", id)
		values.Set("auto_populate_reply_metadata", "true")
	}

	return t.post("https://api.twitter.com/1.1/statuses/update.json", values)
}

func (t *Twitter) like(id string) (url string, err error) {
	return t.post("https://api.twitter.com/1.1/favorites/create.json", urlpkg.Values{
		"id": []string{id},
	})
}

func (t *Twitter) repost(id string) (url string, err error) {
	return t.post(fmt.Sprintf("https://api.twitter.com/1.1/statuses/retweet/%s.json", id), nil)
}

func (t *Twitter) post(urlStr string, values urlpkg.Values) (string, error) {
	url, err := urlpkg.Parse(urlStr)
	if err != nil {
		return "", err
	}

	if values != nil {
		url.RawQuery = values.Encode()
	}

	req, err := http.NewRequest(http.MethodPost, url.String(), nil)
	if err != nil {
		return "", err
	}

	res, err := t.client.Do(req)
	if err != nil {
		return "", err
	}

	var tid map[string]interface{}
	err = json.NewDecoder(res.Body).Decode(&tid)
	if err != nil {
		return "", err
	}

	iid, ok := tid["id_str"]
	if !ok {
		return "", fmt.Errorf("got invalid response: %x", tid)
	}

	id, ok := iid.(string)
	if !ok {
		return "", fmt.Errorf("got invalid response: %x", tid)
	}

	return fmt.Sprintf("https://twitter.com/%s/status/%s", t.conf.User, id), nil
}

func (t *Twitter) idFromUrl(urlStr string) (string, error) {
	replyTo, err := urlpkg.Parse(urlStr)
	if err != nil {
		return "", err
	}

	user := strings.TrimSuffix(replyTo.Path, "/")
	user = strings.TrimPrefix(user, "/")
	parts := strings.Split(user, "/")
	return parts[len(parts)-1], nil
}
