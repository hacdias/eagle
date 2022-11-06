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
	"github.com/hacdias/eagle/v4/config"
	"github.com/hacdias/eagle/v4/contenttype"
	"github.com/hacdias/eagle/v4/entry/mf2"
	"github.com/karlseguin/typed"
	"github.com/vartanbeno/go-reddit/v2/reddit"
	"go.uber.org/zap"
)

var (
	ErrXRayNotFound = errors.New("xray not found")
)

type Author struct {
	Name  string `json:"name,omitempty"`
	Photo string `json:"photo,omitempty"`
	URL   string `json:"url,omitempty"`
}

type Post struct {
	Author    Author    `json:"author,omitempty"`
	Published time.Time `json:"published,omitempty"`
	Content   string    `json:"content,omitempty"`
	URL       string    `json:"url,omitempty"`
	Type      mf2.Type  `json:"type,omitempty"`
	Private   bool      `json:"private,omitempty"`
}

type XRay struct {
	Reddit     *reddit.Client
	Twitter    *config.Twitter
	HttpClient *http.Client
	Log        *zap.SugaredLogger
	Endpoint   string
	UserAgent  string
}

func (x *XRay) FetchXRay(urlStr string) (*Post, interface{}, error) {
	url, err := urlpkg.Parse(urlStr)
	if err != nil {
		return nil, nil, err
	}

	if strings.Contains(url.Host, "reddit.com") && x.Reddit != nil {
		parsed, raw, err := x.fetchAndParseRedditURL(urlStr)
		if err == nil {
			return parsed, raw, nil
		} else {
			x.Log.Warnf("could not download info from reddit %s: %s", urlStr, err.Error())
		}
	}

	data := urlpkg.Values{}
	data.Set("url", url.String())

	if strings.Contains(url.Host, "twitter.com") && x.Twitter != nil {
		data.Set("twitter_api_key", x.Twitter.Key)
		data.Set("twitter_api_secret", x.Twitter.Secret)
		data.Set("twitter_access_token", x.Twitter.Token)
		data.Set("twitter_access_token_secret", x.Twitter.TokenSecret)
	}

	req, err := http.NewRequest(http.MethodPost, x.Endpoint+"/parse", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, nil, err
	}

	req.Header.Add("Content-Type", contenttype.WWWForm)
	req.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))
	req.Header.Add("User-Agent", x.UserAgent)

	res, err := x.HttpClient.Do(req)
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
		return nil, nil, fmt.Errorf("%s: %w", url.String(), ErrXRayNotFound)
	}

	parsed := x.ParseXRay(xray.Data)
	if parsed.URL == "" {
		parsed.URL = urlStr
	}

	return parsed, xray.Data, nil
}

func (x *XRay) ParseXRay(data map[string]interface{}) *Post {
	raw := typed.New(data)
	parsed := &Post{
		URL: raw.StringOr("wm-source", raw.String("url")),
	}

	if date := raw.StringOr("published", raw.String("wm-received")); date != "" {
		t, err := dateparse.ParseStrict(date)
		if err == nil {
			parsed.Published = t
		}
	}

	var hasContent bool

	if content, ok := raw.StringIf("content"); ok {
		parsed.Content = cleanContent(content)
		hasContent = true
	}

	if !hasContent {
		if contentMap, ok := raw.MapIf("content"); ok && !hasContent {
			typedContentMap := typed.New(contentMap)
			if text, ok := typedContentMap.StringIf("text"); ok {
				parsed.Content = cleanContent(text)
				hasContent = true
			} else if html, ok := typedContentMap.StringIf("html"); ok {
				parsed.Content = cleanContent(html)
				hasContent = true
			}
		}
	}

	if !hasContent {
		if name, ok := raw.StringIf("name"); ok {
			parsed.Content = cleanContent(name)
		}
	}

	if photos, ok := raw.StringsIf("photo"); ok {
		parsed.Content += strings.Join(photos, " ")
		parsed.Content = strings.TrimSpace(parsed.Content)
	}

	if author, ok := raw.MapIf("author"); ok {
		typedAuthor := typed.New(author)
		parsed.Author.Name = typedAuthor.String("name")
		parsed.Author.Photo = typedAuthor.String("photo")
		parsed.Author.URL = typedAuthor.String("url")
	}

	if wmProperty, ok := raw.StringIf("wm-property"); ok {
		parsed.Type = mf2.PropertyToType(wmProperty)
	}

	if wmPrivate, ok := raw.BoolIf("wm-private"); ok {
		parsed.Private = wmPrivate
	} else {
		parsed.Private = raw.String("wm-private") == "true"
	}

	return parsed
}
