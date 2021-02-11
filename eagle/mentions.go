package eagle

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// PopulateMentions replaces all Twitter and ActivityPub @mentions in a post
// by proper links, as well as populate the "mentions" frontmatter field.
func (e *Eagle) PopulateMentions(entry *Entry) {
	mentions := []EntryMention{}

	newContent := handles.ReplaceAllStringFunc(entry.Content, func(s string) string {
		ss := s[1:] // Strip out "@" which is encoded in 1 byte
		href := ""

		if idx := strings.Index(ss, "@"); idx != -1 {
			split := strings.SplitN(ss, "@", 2)
			user := split[0]
			domain := split[1]

			href, err := isActivityPub(user, domain)
			if err != nil || href == "" {
				return s
			}
		} else if e.Twitter != nil {
			exists, _ := e.Twitter.UserExists(ss)
			if !exists {
				return s
			}

			href = "https://twitter.com/" + s
		} else {
			return s
		}

		mentions = append(mentions, EntryMention{
			Name: "@" + ss,
			Href: href,
		})

		return "<a href='" + href + "' rel='noopener noreferrer' target='_blank'>@" + ss + "</a>"
	})

	if len(mentions) > 0 {
		entry.Metadata.Mentions = mentions
		entry.Content = newContent
	}
}

// isActivityPub checks if a certain user exists in a certain
// domain and returns its profile link. If an error occurrs, or
// the user does not exist, returns an empty string
func isActivityPub(user, domain string) (string, error) {
	acct := user + "@" + domain
	url := "https://" + domain + "/.well-known/webfinger?resource=acct:" + acct

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("error while creating request: %w", err)
	}

	req.Header.Add("Accept", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("webfinger request failed: %w", err)
	}

	if res.StatusCode >= 400 {
		return "", fmt.Errorf("unexpected status code for webfinger: %w", err)
	}

	var r map[string]interface{}
	err = json.NewDecoder(res.Body).Decode(&r)
	if err != nil {
		return "", fmt.Errorf("invalid request body received: %w", err)
	}

	links, ok := r["links"].([]interface{})
	if !ok {
		return "", fmt.Errorf("invalid links received: %v", r["links"])
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

	return home, nil
}

var handles = regexp.MustCompile(`(?m)@([^\s]*)`)
