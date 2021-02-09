package services

/*
import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/hacdias/eagle/middleware/micropub"
)

var handles = regexp.MustCompile(`(?m)@([^\s]*)`)

func (h *Hugo) findMentions(entry *HugoEntry) {
	mentions := []interface{}{}

	newContent := handles.ReplaceAllStringFunc(entry.Content, func(s string) string {
		s = s[1:] // Strip out "@" which is encoded in 1 byte
		href := ""

		if idx := strings.Index(s, "@"); idx != -1 {
			split := strings.SplitN(s, "@", 2)
			user := split[0]
			domain := split[1]

			href = h.isActivityPub(user, domain)
			if href == "" {
				return s
			}
		} else {
			if !h.isTwitterUser(s) {
				return s
			}

			href = "https://twitter.com/" + s
		}

		mentions = append(mentions, map[string]string{
			"name": "@" + s,
			"href": href,
		})

		return "<a href='" + href + "' rel='noopener noreferrer' target='_blank'>@" + s + "</a>"
	})

	if len(mentions) > 0 {
		entry.Metadata["mentions"] = mentions
		entry.Content = newContent
	}
}

// isActivityPub checks if a certain user exists in a certain
// domain and returns its profile link. If an error occurrs, or
// the user does not exist, returns an empty string
func (h *Hugo) isActivityPub(user, domain string) string {
	acct := user + "@" + domain
	url := "https://" + domain + "/.well-known/webfinger?resource=acct:" + acct

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		h.Warnf("isActivityPub: could not create request: %s", err)
		return ""
	}

	req.Header.Add("Accept", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		h.Infof("isActivityPub: could not do request: %s", err)
		return ""
	}

	if res.StatusCode >= 400 {
		h.Infof("isActivityPub: unexpected status code: %d", res.StatusCode)
		return ""
	}

	var r map[string]interface{}
	err = json.NewDecoder(res.Body).Decode(&r)
	if err != nil {
		h.Infof("isActivityPub: invalid request body: %s", err)
		return ""
	}

	links, ok := r["links"].([]interface{})
	if !ok {
		h.Infof("isActivityPub: invalid links: %s", r["links"])
		return ""
	}

	home := ""

	for _, l := range links {
		link, ok := l.(map[string]interface{})
		if !ok {
			continue
		}

		if link["rel"] == "http://webfinger.net/rel/profile-page" {
			home = link["href"].(string)
			break
		}

		if link["rel"] == "self" {
			home = link["href"].(string)
		}
	}

	return home
}

// isTwitterUser checks if a user exists on twitter. If it doesn't,
// or if there's an error, returns false. True otherwise.
func (h *Hugo) isTwitterUser(user string) bool {
	if h.Twitter == nil {
		return false
	}

	exists, err := h.Twitter.UserExists(user)
	if err != nil {
		h.Warnf("isTwitterUser: error on twitter API: %s", err)
	}

	return exists
}

*/
