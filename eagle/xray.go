package eagle

import (
	"encoding/json"
	"net/http"
	urlpkg "net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/araddon/dateparse"
)

// XRay is an xray of an external post. This is the format used to store
// Webmentions and ReplyTo context.
type XRay struct {
	Type    string    `yaml:"type,omitempty" json:"type,omitempty"`
	URL     string    `yaml:"url,omitempty" json:"url,omitempty"`
	Name    string    `yaml:"name,omitempty" json:"name,omitempty"`
	Content string    `yaml:"content,omitempty" json:"content,omitempty"`
	Date    time.Time `yaml:"date,omitempty" json:"date,omitempty"`
	Author  *Author   `yaml:"author,omitempty" json:"author,omitempty"`
}

type Author struct {
	Name  string `yaml:"name,omitempty" json:"name,omitempty"`
	URL   string `yaml:"url,omitempty" json:"url,omitempty"`
	Photo string `yaml:"photo,omitempty" json:"photo,omitempty"`
}

func (e *Eagle) GetXRay(url string) (*XRay, error) {
	data := urlpkg.Values{}
	data.Set("url", url)

	if strings.Contains(url, "twitter.com") && e.Config.Twitter != nil {
		data.Set("twitter_api_key", e.Config.Twitter.Key)
		data.Set("twitter_api_secret", e.Config.Twitter.Secret)
		data.Set("twitter_access_token", e.Config.Twitter.Token)
		data.Set("twitter_access_token_secret", e.Config.Twitter.TokenSecret)
	}

	req, err := http.NewRequest(http.MethodPost, e.Config.XRayEndpoint+"/parse", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))
	req.Header.Add("User-Agent", e.userAgent("XRay"))

	res, err := e.httpClient.Do(req)
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

func parseXRayResponse(xray *xrayResponse) *XRay {
	ee := &XRay{}

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
		ee.Author = &Author{}

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