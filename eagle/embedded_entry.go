package eagle

import (
	"context"
	"encoding/json"
	"net/http"
	urlpkg "net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/araddon/dateparse"
)

func (e *Eagle) GetEmbeddedEntry(url string) (*EmbeddedEntry, error) {
	data := urlpkg.Values{}
	data.Set("url", url)

	if strings.Contains(url, "twitter.com") && e.Config.Twitter != nil {
		data.Set("twitter_api_key", e.Config.Twitter.Key)
		data.Set("twitter_api_secret", e.Config.Twitter.Secret)
		data.Set("twitter_access_token", e.Config.Twitter.Token)
		data.Set("twitter_access_token_secret", e.Config.Twitter.TokenSecret)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.Config.XRay.Endpoint+"/parse", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var xray xrayResponse
	err = json.NewDecoder(res.Body).Decode(&xray)
	if err != nil {
		return nil, err
	}

	ee := parseXRayResponse(&xray)
	ee.URL = url
	return ee, nil
}

func parseXRayResponse(xray *xrayResponse) *EmbeddedEntry {
	ee := &EmbeddedEntry{}

	if xray.Data == nil {
		return ee
	}

	if v, ok := xray.Data["published"].(string); ok {
		t, err := dateparse.ParseStrict(v)
		if err == nil {
			ee.Date = t
		}
	}

	if v, ok := xray.Data["name"].(string); ok {
		ee.Name = v
	}

	if v, ok := xray.Data["content"].(map[string]interface{}); ok {
		if t, ok := v["text"].(string); ok {
			ee.Content = t
		} else if h, ok := v["html"].(string); ok {
			ee.Content = h
		}
	}

	if v, ok := xray.Data["content"].(string); ok {
		ee.Content = v
	}

	if v, ok := xray.Data["summary"].(string); ok && ee.Content == "" {
		ee.Content = v
	}

	if ee.Content != "" {
		ee.Content = cleanContent(ee.Content)
	}

	if a, ok := xray.Data["author"].(map[string]interface{}); ok {
		ee.Author = &EntryAuthor{}

		if v, ok := a["name"].(string); ok {
			ee.Author.Name = v
		}

		if v, ok := a["url"].(string); ok {
			ee.Author.URL = v
		}

		if v, ok := a["photo"].(string); ok {
			ee.Author.Photo = v
		}
	}

	return ee
}

type xrayResponse struct {
	Data map[string]interface{} `json:"data"`
	Code int                    `json:"code"`
}

var spaceCollapser = regexp.MustCompile(`\s+`)

func cleanContent(data string) string {
	data = strings.TrimSpace(data)
	data = spaceCollapser.ReplaceAllString(data, " ") // Collapse whitespaces
	return data
}
