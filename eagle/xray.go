package eagle

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	urlpkg "net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/araddon/dateparse"
	"github.com/hacdias/eagle/v4/contenttype"
	"github.com/karlseguin/typed"
)

type xray struct {
	Data map[string]interface{} `json:"data"`
	Code int                    `json:"code"`
}

func (e *Eagle) getXRay(urlStr string) (map[string]interface{}, error) {
	url, err := urlpkg.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	if strings.Contains(url.Host, "reddit.com") && e.reddit != nil {
		data, err := e.getRedditXRay(urlStr)
		if err == nil {
			return data, nil
		} else {
			e.log.Warnf("could not download info from reddit %s: %s", urlStr, err.Error())
		}
	}

	data := urlpkg.Values{}
	data.Set("url", url.String())

	if strings.Contains(url.Host, "twitter.com") && e.Config.Twitter != nil {
		data.Set("twitter_api_key", e.Config.Twitter.Key)
		data.Set("twitter_api_secret", e.Config.Twitter.Secret)
		data.Set("twitter_access_token", e.Config.Twitter.Token)
		data.Set("twitter_access_token_secret", e.Config.Twitter.TokenSecret)
	}

	req, err := http.NewRequest(http.MethodPost, e.Config.XRayEndpoint+"/parse", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", contenttype.WWWForm)
	req.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))
	req.Header.Add("User-Agent", e.userAgent("XRay"))

	res, err := e.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var xray xray
	err = json.NewDecoder(res.Body).Decode(&xray)
	if err != nil {
		return nil, err
	}

	if xray.Data == nil {
		return nil, fmt.Errorf("no xray found for %s", url.String())
	}

	jf2 := e.parseXRay(xray.Data)
	if jf2 == nil {
		return nil, fmt.Errorf("no xray found for %s", url.String())
	}

	return jf2, nil
}

func (e *Eagle) getRedditXRay(urlStr string) (map[string]interface{}, error) {
	url, err := urlpkg.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	path := strings.TrimSuffix(url.Path, "/")
	path = strings.TrimPrefix(path, "/")
	parts := strings.Split(path, "/")

	if len(parts) == 5 {
		return e.getRedditPostXRay("t3_" + parts[3])
	} else if len(parts) == 6 {
		return e.getRedditCommentXRay("t1_" + parts[5])
	}

	return nil, errors.New("unsupported reddit url")
}

func (e *Eagle) getRedditCommentXRay(id string) (map[string]interface{}, error) {
	_, comments, _, _, err := e.reddit.Listings.Get(context.Background(), id)
	if err != nil {
		return nil, err
	}

	if len(comments) != 1 {
		return nil, errors.New("comment not found")
	}

	content := html.UnescapeString(comments[0].Body)
	if content == "[deleted]" {
		return nil, errors.New("comment was deleted")
	}

	data := map[string]interface{}{
		"content":   cleanContent(content),
		"published": comments[0].Created.Time.Format(time.RFC3339),
		"url":       "https://www.reddit.com" + comments[0].Permalink,
		"type":      "entry",
	}

	if comments[0].Author != "[deleted]" {
		data["author"] = map[string]interface{}{
			"name": comments[0].Author,
			"url":  "https://www.reddit.com/u/" + comments[0].Author,
			"type": "card",
		}
	}

	return data, nil
}

func (e *Eagle) getRedditPostXRay(id string) (map[string]interface{}, error) {
	posts, _, _, _, err := e.reddit.Listings.Get(context.Background(), id)
	if err != nil {
		return nil, err
	}

	if len(posts) != 1 {
		return nil, errors.New("post not found")
	}

	content := html.UnescapeString(posts[0].Body)
	if content == "[deleted]" || content == "" {
		content = posts[0].Title
	}

	if content == "[deleted]" {
		return nil, errors.New("post was deleted")
	}

	data := map[string]interface{}{
		"content":   cleanContent(content),
		"published": posts[0].Created.Time.Format(time.RFC3339),
		"url":       posts[0].URL,
		"type":      "entry",
	}

	if posts[0].Author != "[deleted]" {
		data["author"] = map[string]interface{}{
			"name": posts[0].Author,
			"url":  "https://www.reddit.com/u/" + posts[0].Author,
			"type": "card",
		}
	}

	return data, nil
}

func (e *Eagle) parseXRay(xray map[string]interface{}) map[string]interface{} {
	data := typed.New(xray)

	hasDate := false

	if date, ok := data.StringIf("published"); ok {
		t, err := dateparse.ParseStrict(date)
		if err == nil {
			data["published"] = t.Format(time.RFC3339)
		}
		hasDate = true
	}

	if date, ok := data.StringIf("wm-received"); ok {
		t, err := dateparse.ParseStrict(date)
		if err == nil {
			data["wm-received"] = t.Format(time.RFC3339)
		}
		if !hasDate {
			data["published"] = data["wm-received"]
		}
	}

	var hasContent bool

	if content, ok := data.StringIf("content"); ok {
		data["content"] = cleanContent(content)
		hasContent = true
	}

	if cmap, ok := data.MapIf("content"); ok && !hasContent {
		content := typed.New(cmap)
		if text, ok := content.StringIf("text"); ok {
			data["content"] = cleanContent(text)
		} else if html, ok := content.StringIf("html"); ok {
			data["content"] = cleanContent(html)
		}
	}

	if cauthor, ok := data.MapIf("author"); ok {
		author := typed.New(cauthor)

		if photo, ok := author.StringIf("photo"); ok {
			author["photo"] = e.safeUploadFromURL("wm", photo, true)
		}

		data["author"] = author
	}

	if _, ok := data.StringIf("url"); !ok {
		if source, ok := data.StringIf("wm-source"); ok {
			data["url"] = source
		}
	}

	if wmProperty, ok := data.StringIf("wm-property"); ok {
		if v, ok := webmentionTypes[wmProperty]; ok {
			data["post-type"] = v
		} else {
			data["post-type"] = "mention"
		}
	}

	return data
}

var (
	spaceCollapser = regexp.MustCompile(`\s+`)
)

func cleanContent(data string) string {
	data = strings.TrimSpace(data)
	data = spaceCollapser.ReplaceAllString(data, " ") // Collapse whitespaces
	return data
}
