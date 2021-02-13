package eagle

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/dghubble/oauth1"
	"github.com/hacdias/eagle/config"
)

type Twitter struct {
	conf   *config.Twitter
	client *http.Client
}

func NewTwitter(opts *config.Twitter) *Twitter {
	config := oauth1.NewConfig(opts.Key, opts.Secret)
	token := oauth1.NewToken(opts.Token, opts.TokenSecret)
	client := config.Client(oauth1.NoContext, token)

	return &Twitter{
		conf:   opts,
		client: client,
	}
}

func (t *Twitter) Syndicate(entry *Entry) (string, error) {
	status := entry.RawContent
	if len(status) > 280 {
		status = strings.TrimSpace(status[0:270-len(entry.Permalink)]) + "... " + entry.Permalink
	}

	u, err := url.Parse("https://api.twitter.com/1.1/statuses/update.json")
	if err != nil {
		return "", err
	}

	q := u.Query()
	q.Set("status", status)

	if entry.Metadata.ReplyTo != nil {
		replyTo, err := url.Parse(entry.Metadata.ReplyTo.URL)
		if err != nil {
			return "", err
		}

		user := strings.TrimSuffix(replyTo.Path, "/")
		user = strings.TrimPrefix(user, "/")
		parts := strings.Split(user, "/")

		q.Set("in_reply_to_status_id", parts[0])
		q.Set("auto_populate_reply_metadata", "true")
		// TODO: add attachment_url for retweet with status
	}

	u.RawQuery = q.Encode()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), nil)
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

	id, ok := tid["id_str"]
	if !ok {
		return "", fmt.Errorf("got invalid response: %x", tid)
	}

	return "https://twitter.com/" + t.conf.User + "/status/" + fmt.Sprint(id), nil
}

func (t *Twitter) UserExists(user string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.twitter.com/1.1/users/lookup.json?screen_name="+user, nil)
	if err != nil {
		return false, err
	}

	res, err := t.client.Do(req)
	if err != nil {
		return false, err
	}

	var r interface{}
	err = json.NewDecoder(res.Body).Decode(&r)
	if err != nil {
		return false, err
	}

	if reflect.ValueOf(r).Kind() == reflect.Slice {
		return reflect.ValueOf(r).Len() > 0, nil
	}

	return false, nil
}
