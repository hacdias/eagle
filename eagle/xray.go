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

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
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

// TODO: merge parsing with https://github.com/hacdias/eagle/blob/main/eagle/webmentions.go#L245
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
			hasContent = true
			data["content"] = cleanContent(text)
		} else if html, ok := content.StringIf("html"); ok {
			hasContent = true
			data["content"] = cleanContent(html)
		}
	}

	if cauthor, ok := data.MapIf("author"); ok {
		author := typed.New(cauthor)

		if photo, ok := author.StringIf("photo"); ok {
			author["photo"] = e.uploadXRayAuthorPhoto(photo)
		}

		data["author"] = author
	}

	if _, ok := data.StringIf("url"); !ok {
		if source, ok := data.StringIf("wm-source"); ok {
			data["url"] = source
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
