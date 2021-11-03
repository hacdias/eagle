package eagle

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	urlpkg "net/url"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/araddon/dateparse"
	"github.com/karlseguin/typed"
	"github.com/microcosm-cc/bluemonday"
)

// XRayDirectory is the directory where all the xrays will be stored
// when retrieved by .GetXrayAndSave
const XRayDirectory = "xrays"

// TODO: maybe does not need to be exported
func (e *Eagle) GetXRay(urlStr string) (map[string]interface{}, error) {
	url, err := urlpkg.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	filename, err := getXRayFilename(url)
	if err != nil {
		return nil, err
	}

	jf2, err := e.xrayFromDisk(filename)
	if err == nil {
		return jf2, nil
	}

	jf2, err = e.fetchXRay(url)
	if err != nil {
		return nil, err
	}

	err = e.SrcFs.MkdirAll(filepath.Dir(filename), 0777)
	if err != nil {
		return nil, err
	}

	err = e.PersistJSON(filename, jf2, "xray: "+url.String())
	if err != nil {
		return nil, err
	}

	return jf2, nil
}

func (e *Eagle) safeXRayFromDisk(urlStr string) map[string]interface{} {
	url, err := urlpkg.Parse(urlStr)
	if err != nil {
		return nil
	}

	filename, err := getXRayFilename(url)
	if err != nil {
		return nil
	}

	jf2, _ := e.xrayFromDisk(filename)
	return jf2
}

func (e *Eagle) xrayFromDisk(filename string) (map[string]interface{}, error) {
	_, err := e.SrcFs.Stat(filename)
	if err != nil {
		return nil, err
	}

	var jf2 map[string]interface{}
	data, err := e.SrcFs.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, &jf2)
	if err != nil {
		return nil, err
	}

	return jf2, nil
}

func (e *Eagle) fetchXRay(url *urlpkg.URL) (map[string]interface{}, error) {
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

	var xray xrayResponse
	err = json.NewDecoder(res.Body).Decode(&xray)
	if err != nil {
		return nil, err
	}

	if xray.Data == nil {
		return nil, fmt.Errorf("no xray found for %s", url.String())
	}

	jf2 := e.ParseXRayResponse(xray.Data)
	if jf2 == nil {
		return nil, fmt.Errorf("no xray found for %s", url.String())
	}

	return jf2, nil
}

func getXRayFilename(url *urlpkg.URL) (string, error) {
	var err error

	host := url.Host
	if strings.Contains(host, ":") {
		host, _, err = net.SplitHostPort(url.Host)
		if err != nil {
			return "", err
		}
	}

	return filepath.Join(XRayDirectory, host, url.Path, "data.json"), nil
}

// TODO: call this for all things already so they upload pics
func (e *Eagle) ParseXRayResponse(xray map[string]interface{}) map[string]interface{} {
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

type xrayResponse struct {
	Data map[string]interface{} `json:"data"`
	Code int                    `json:"code"`
}

var (
	spaceCollapser = regexp.MustCompile(`\s+`)
	sanitizer      = bluemonday.StrictPolicy()
)

func cleanContent(data string) string {
	data = sanitizer.Sanitize(data)
	data = strings.TrimSpace(data)
	data = spaceCollapser.ReplaceAllString(data, " ") // Collapse whitespaces
	return data
}
