package eagle

// import (
// 	"encoding/json"
// 	"fmt"
// 	"net/http"
// 	"regexp"
// 	"strings"
// )

// // AutoLinkMentions mentions replaces all Twitter and ActivityPub @mentions in a
// // post by proper links.
// // FIXME: do not check for mentions inside code blocks, only paragraphs.
// func (e *Eagle) AutoLinkMentions(entry *Entry) {
// 	entry.Content = handles.ReplaceAllStringFunc(entry.Content, func(s string) string {
// 		ss := s[1:] // Strip out "@" which is encoded in 1 byte
// 		href := ""

// 		if idx := strings.Index(ss, "@"); idx != -1 {
// 			split := strings.SplitN(ss, "@", 2)
// 			user := split[0]
// 			domain := split[1]

// 			href, err := e.isActivityPub(user, domain)
// 			if err != nil || href == "" {
// 				return s
// 			}
// 		} else if e.Twitter != nil {
// 			exists, _ := e.Twitter.UserExists(ss)
// 			if !exists {
// 				return s
// 			}

// 			href = "https://twitter.com/" + s
// 		} else {
// 			return s
// 		}

// 		return "<a href='" + href + "' rel='noopener noreferrer' target='_blank'>@" + ss + "</a>"
// 	})
// }

// // isActivityPub checks if a certain user exists in a certain
// // domain and returns its profile link. If an error occurrs, or
// // the user does not exist, returns an empty string
// func (e *Eagle) isActivityPub(user, domain string) (string, error) {
// 	acct := user + "@" + domain
// 	url := "https://" + domain + "/.well-known/webfinger?resource=acct:" + acct

// 	req, err := http.NewRequest(http.MethodGet, url, nil)
// 	if err != nil {
// 		return "", fmt.Errorf("error while creating request: %w", err)
// 	}

// 	req.Header.Add("Accept", "application/json")
// 	req.Header.Add("User-Agent", e.userAgent("ActivityPub"))

// 	res, err := e.httpClient.Do(req)
// 	if err != nil {
// 		return "", fmt.Errorf("webfinger request failed: %w", err)
// 	}
// 	defer res.Body.Close()

// 	if res.StatusCode >= 400 {
// 		return "", fmt.Errorf("unexpected status code for webfinger: %w", err)
// 	}

// 	var r map[string]interface{}
// 	err = json.NewDecoder(res.Body).Decode(&r)
// 	if err != nil {
// 		return "", fmt.Errorf("invalid request body received: %w", err)
// 	}

// 	links, ok := r["links"].([]interface{})
// 	if !ok {
// 		return "", fmt.Errorf("invalid links received: %v", r["links"])
// 	}

// 	home := ""

// 	for _, l := range links {
// 		link, ok := l.(map[string]interface{})
// 		if !ok {
// 			continue
// 		}

// 		if link["rel"] == "http://webfinger.net/rel/profile-page" {
// 			home = link["href"].(string)
// 			break
// 		}

// 		if link["rel"] == "self" {
// 			home = link["href"].(string)
// 		}
// 	}

// 	return home, nil
// }

// var handles = regexp.MustCompile(`(?m)@([^\s]*)`)
