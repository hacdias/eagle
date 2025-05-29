package xray

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	urlpkg "net/url"
	"strconv"
	"strings"
	"time"

	"github.com/araddon/dateparse"
	"github.com/karlseguin/typed"
	"go.hacdias.com/indielib/microformats"
	"go.uber.org/zap"
)

var (
	ErrPostNotFound = errors.New("post xray not found")
)

type Config struct {
	GitHubToken string
	Endpoint    string
	UserAgent   string
}

type XRay struct {
	c          *Config
	log        *zap.SugaredLogger
	httpClient *http.Client
}

func NewXRay(c *Config, log *zap.SugaredLogger) (*XRay, error) {
	x := &XRay{
		c:   c,
		log: log,
		httpClient: &http.Client{
			Timeout: 2 * time.Minute,
		},
	}

	return x, nil
}

type xrayResponse struct {
	Data map[string]any `json:"data"`
	Code int            `json:"code"`
}

func (x *XRay) Fetch(urlStr string) (*Post, any, error) {
	url, err := urlpkg.Parse(urlStr)
	if err != nil {
		return nil, nil, err
	}

	data := urlpkg.Values{}
	data.Set("url", url.String())

	if strings.Contains(url.Host, "github.com") && x.c.GitHubToken != "" {
		data.Set("github_access_token", x.c.GitHubToken)
	}

	req, err := http.NewRequest(http.MethodPost, x.c.Endpoint+"/parse", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, nil, err
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))
	req.Header.Add("User-Agent", x.c.UserAgent)

	res, err := x.httpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer res.Body.Close()

	var xray xrayResponse
	err = json.NewDecoder(res.Body).Decode(&xray)
	if err != nil {
		return nil, nil, err
	}

	if xray.Data == nil ||
		typed.New(xray.Data).String("type") == "unknown" {
		return nil, nil, fmt.Errorf("%s: %w", url.String(), ErrPostNotFound)
	}

	parsed := x.Parse(xray.Data)
	if parsed.URL == "" {
		parsed.URL = urlStr
	}

	return parsed, xray.Data, nil
}

func (x *XRay) Parse(data map[string]any) *Post {
	return Parse(data)
}

func Parse(data map[string]any) *Post {
	raw := typed.New(data)

	if raw.String("type") == "feed" {
		items := raw.Maps("items")
		if len(items) >= 1 {
			for _, item := range items {
				if typed.New(item).String("type") == "entry" {
					return Parse(item)
				}
			}

			return Parse(items[0])
		}
	}

	parsed := &Post{
		URL: raw.StringOr("wm-source", raw.String("url")),
	}

	if date := raw.StringOr("published", raw.String("wm-received")); date != "" {
		t, err := dateparse.ParseStrict(date)
		if err == nil {
			parsed.Date = t
		}
	}

	if name, ok := raw.StringIf("name"); ok {
		parsed.Name = name
	}

	var hasContent bool

	if content, ok := raw.StringIf("content"); ok {
		parsed.Content = SanitizeContent(content)
		hasContent = true
	}

	if !hasContent {
		if contentMap, ok := raw.MapIf("content"); ok && !hasContent {
			typedContentMap := typed.New(contentMap)
			if text, ok := typedContentMap.StringIf("text"); ok {
				parsed.Content = SanitizeContent(text)
				hasContent = true
			} else if html, ok := typedContentMap.StringIf("html"); ok {
				parsed.Content = SanitizeContent(html)
				hasContent = true
			}
		}
	}

	if !hasContent {
		if name, ok := raw.StringIf("name"); ok {
			parsed.Content = SanitizeContent(name)
		}
	}

	if photos, ok := raw.StringsIf("photo"); ok {
		parsed.Content += " " + strings.Join(photos, " ")
		parsed.Content = strings.TrimSpace(parsed.Content)
	}

	if author, ok := raw.MapIf("author"); ok {
		typedAuthor := typed.New(author)
		parsed.Author = typedAuthor.String("name")
		parsed.AuthorPhoto = typedAuthor.String("photo")
		parsed.AuthorURL = typedAuthor.String("url")
	}

	if wmProperty, ok := raw.StringIf("wm-property"); ok {
		parsed.Type = microformats.PropertyToType(wmProperty)
	}

	if wmPrivate, ok := raw.BoolIf("wm-private"); ok {
		parsed.Private = wmPrivate
	} else {
		parsed.Private = raw.String("wm-private") == "true"
	}

	return parsed
}
