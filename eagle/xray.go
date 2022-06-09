package eagle

import (
	"encoding/json"
	"fmt"
	"net/http"
	urlpkg "net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/araddon/dateparse"
	"github.com/hacdias/eagle/v3/contenttype"
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
		data, err := e.reddit.GetXRay(urlStr)
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
			author["photo"] = e.safeUploadFromURL("wm", photo)
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
