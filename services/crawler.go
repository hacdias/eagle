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

	/*{
	  "content": {
	    "text": "Thanks to @hacdias, I just discovered this:curl wttr.in\nBut wttr.in can’t only show you your current weather and the forecast (based on your IP location) in your terminal, it also has a lot of extra options and is open source. Check out the GitHub repository.wttr.in is a console-oriented weather forecast service that supports various information representation methods like terminal-oriented ANSI-sequences for console HTTP clients (curl, httpie, or wget), HTML for web browsers, or PNG for graphical viewers.https://wttr.in/",
	    "html": "<p>Thanks to <a href=\"https://hacdias.com/2020/01/20/4/weather-terminal/\">@hacdias</a>, I just discovered this:</p><pre><code>curl wttr.in\n</code></pre><p>But <a href=\"https://wttr.in/\">wttr.in</a> can’t only show you your current weather and the forecast (based on your IP location) in your terminal, it also has a lot of extra options and is open source. Check out the <a href=\"https://github.com/chubin/wttr.in\">GitHub repository</a>.</p><blockquote><p>wttr.in is a console-oriented weather forecast service that supports various information representation methods like terminal-oriented ANSI-sequences for console HTTP clients (curl, httpie, or wget), HTML for web browsers, or PNG for graphical viewers.</p></blockquote><p><a class=\"u-bookmark-of\" href=\"https://wttr.in/\">https://wttr.in/</a></p>"
	  },
	  "author": {
	    "type": "card",
	    "name": "Jan-Lukas Else",
	    "url": "https://jlelse.dev/",
	    "photo": "https://jlelse.dev/profile-512.jpg"
	  },
	}*/

	/*
	  content: That’s an awesome approach to combine the 90s web with the IndieWeb, Henrique.
	    I really like your new 90s web-inspired website style, well done!
	  author:
	    name: Jan-Lukas Else
	    url: https://jlelse.dev/
	    photo: https://jlelse.dev/profile.png
	*/

	return entry, nil

}

type xrayResponse struct {
	Data map[string]interface{} `json:"data"`
	Code int                    `json:"code"`
}

func clean(data string) string {
	space := regexp.MustCompile(`\s+`)
	data = strings.TrimSpace(data)
	// Collapse whitespaces
	data = space.ReplaceAllString(data, " ")

	// BUG> Remove quotes: https://github.com/gohugoio/hugo/issues/8219
	data = strings.ReplaceAll(data, "\"", "")
	return data
}
