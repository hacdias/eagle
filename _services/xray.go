package services

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/hacdias/eagle/config"
	"go.uber.org/zap"
)

type XRay struct {
	*zap.SugaredLogger
	config.XRay
	Domain      string
	StoragePath string
	Twitter     config.Twitter
}

type XRayRequest struct {
	URL  string
	Body string
}

type xrayResponse struct {
	Data map[string]interface{} `json:"data"`
	Code int                    `json:"code"`
}

func (x *XRay) Request(opts *XRayRequest) (*xrayResponse, error) {
	data := url.Values{}
	data.Set("twitter_api_key", x.Twitter.Key)
	data.Set("twitter_api_secret", x.Twitter.Secret)
	data.Set("twitter_access_token", x.Twitter.Token)
	data.Set("twitter_access_token_secret", x.Twitter.TokenSecret)

	if opts.URL != "" {
		data.Set("url", opts.URL)
	}

	if opts.Body != "" {
		data.Set("body", opts.Body)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, x.Endpoint+"/parse", strings.NewReader(data.Encode()))
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

	if xray.Data == nil {
		return &xray, nil
	}

	// TODO: why did I add this?
	// if _, ok := xray.Data["published"]; ok {
	// 	xray.Data["published"] = new Date(res.body.data.published).toISOString() (ISO 8601),
	// }

	return &xray, nil
}
func (x *XRay) RequestAndSave(url string) error {
	if strings.HasPrefix(url, "/") {
		url = x.Domain + url
	}

	file := path.Join(x.StoragePath, fmt.Sprintf("%x.json", sha256.Sum256([]byte(url))))

	if _, err := os.Stat(file); err == nil {
		x.Infof("%s already x-rayed: %s", url, file)
		return nil
	}

	data, err := x.Request(&XRayRequest{URL: url})
	if err != nil {
		return err
	}

	if data.Code != 200 {
		return errors.New("page cannot be x-rayed")
	}

	if _, ok := data.Data["url"]; !ok {
		data.Data["url"] = url
	}

	js, err := json.MarshalIndent(data.Data, "", "  ")
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(file, js, 0644)
	if err != nil {
		return err
	}

	x.Infof("%s x-rayed successfully", url)
	return nil
}
