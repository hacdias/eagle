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
	"github.com/hacdias/eagle/v4/pkg/contenttype"
	"github.com/hacdias/eagle/v4/pkg/mf2"
	"github.com/karlseguin/typed"
	"github.com/vartanbeno/go-reddit/v2/reddit"
	"go.uber.org/zap"
)

var (
	ErrPostNotFound = errors.New("post xray not found")
)

type Twitter struct {
	Key         string
	Secret      string
	Token       string
	TokenSecret string
}

type Reddit = reddit.Credentials

type Config struct {
	Reddit      *Reddit
	Twitter     *Twitter
	GitHubToken string
	Endpoint    string
	UserAgent   string
}

type XRay struct {
	c            *Config
	log          *zap.SugaredLogger
	redditClient *reddit.Client
	httpClient   *http.Client
}

func NewXRay(c *Config, log *zap.SugaredLogger) (*XRay, error) {
	x := &XRay{
		c:   c,
		log: log,
		httpClient: &http.Client{
			Timeout: 2 * time.Minute,
		},
	}

	if c.Reddit != nil {
		client, err := reddit.NewClient(*c.Reddit)
		if err != nil {
			return nil, err
		}
		x.redditClient = client
	}

	return x, nil
}

type xrayResponse struct {
	Data map[string]interface{} `json:"data"`
	Code int                    `json:"code"`
}

func (x *XRay) Fetch(urlStr string) (*Post, interface{}, error) {
	url, err := urlpkg.Parse(urlStr)
	if err != nil {
		return nil, nil, err
	}

	if strings.Contains(url.Host, "reddit.com") && x.redditClient != nil {
		parsed, raw, err := x.fetchAndParseRedditURL(urlStr)
		if err == nil {
			return parsed, raw, nil
		} else {
			x.log.Warnf("could not download info from reddit %s: %s", urlStr, err.Error())
		}
	}

	data := urlpkg.Values{}
	data.Set("url", url.String())

	if strings.Contains(url.Host, "twitter.com") && x.c.Twitter != nil {
		data.Set("twitter_api_key", x.c.Twitter.Key)
		data.Set("twitter_api_secret", x.c.Twitter.Secret)
		data.Set("twitter_access_token", x.c.Twitter.Token)
		data.Set("twitter_access_token_secret", x.c.Twitter.TokenSecret)
	}

	if strings.Contains(url.Host, "github.com") && x.c.GitHubToken != "" {
		data.Set("github_access_token", x.c.GitHubToken)
	}

	req, err := http.NewRequest(http.MethodPost, x.c.Endpoint+"/parse", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, nil, err
	}

	req.Header.Add("Content-Type", contenttype.WWWForm)
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

func (x *XRay) Parse(data map[string]interface{}) *Post {
	return Parse(data)
}

func Parse(data map[string]interface{}) *Post {
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

	if name, ok := raw.StringIf("name"); ok {
		parsed.Name = name
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

	if coins, ok := raw.IntIf("swarm-coins"); ok {
		parsed.SwarmCoins = coins
	}

	return parsed
}
