package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/araddon/dateparse"
	"github.com/hacdias/eagle/config"
	"go.uber.org/zap"
)

type Crawler struct {
	*zap.SugaredLogger

	xray    config.XRay
	twitter config.Twitter
}

func NewCrawler(conf *config.Config) *Crawler {
	return &Crawler{
		SugaredLogger: conf.S().Named("xray"),
		xray:          conf.XRay,
		twitter:       conf.Twitter,
	}
}

func (c *Crawler) Crawl(url string) (*EmbeddedEntry, error) {
	return c.crawl(url)
}

func (c *Crawler) crawl(u string) (*EmbeddedEntry, error) {
	data := url.Values{}
	data.Set("url", u)

	if strings.Contains(u, "twitter.com") {
		data.Set("twitter_api_key", c.twitter.Key)
		data.Set("twitter_api_secret", c.twitter.Secret)
		data.Set("twitter_access_token", c.twitter.Token)
		data.Set("twitter_access_token_secret", c.twitter.TokenSecret)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.xray.Endpoint+"/parse", strings.NewReader(data.Encode()))
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

	entry := &EmbeddedEntry{
		URL: u,
	}

	if xray.Data == nil {
		return entry, nil
	}

	if v, ok := xray.Data["published"].(string); ok {
		t, err := dateparse.ParseStrict(v)
		if err == nil {
			entry.Date = t
		}
	}

	if v, ok := xray.Data["name"].(string); ok {
		entry.Name = v
	}

	if v, ok := xray.Data["content"].(map[string]interface{}); ok {
		if t, ok := v["text"].(string); ok {
			entry.Content = t
		} else if h, ok := v["html"].(string); ok {
			entry.Content = h
		}
	}

	if v, ok := xray.Data["content"].(string); ok {
		entry.Content = v
	}

	if v, ok := xray.Data["summary"].(string); ok && entry.Content == "" {
		entry.Content = v
	}

	if entry.Content != "" {
		entry.Content = cleanContent(entry.Content)
	}

	if a, ok := xray.Data["author"].(map[string]interface{}); ok {
		entry.Author = &EntryAuthor{}

		if v, ok := a["name"].(string); ok {
			entry.Author.Name = v
		}

		if v, ok := a["url"].(string); ok {
			entry.Author.URL = v
		}

		if v, ok := a["photo"].(string); ok {
			entry.Author.Photo = v
		}
	}

	return entry, nil
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
