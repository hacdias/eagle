package eagle

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"path"
	"path/filepath"
	"strings"

	"github.com/hacdias/eagle/v3/config"
)

type Media struct {
	httpClient *http.Client
	*config.BunnyCDN
}

func (m *Media) UploadMedia(filename string, data io.Reader) (string, error) {
	if !strings.HasPrefix(filename, "/") {
		filename = "/" + filename
	}

	req, err := http.NewRequest(http.MethodPut, "https://storage.bunnycdn.com/"+m.Zone+filename, data)
	if err != nil {
		return "", err
	}

	req.Header.Set("AccessKey", m.Key)

	res, err := m.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated {
		return "", errors.New("status code is not ok")
	}

	return m.Base + filename, nil
}

func (e *Eagle) UploadFile(base, ext string, data io.Reader) (string, error) {
	if e.media == nil {
		return "", errors.New("media is not implemented")
	}

	body, err := ioutil.ReadAll(data)
	if err != nil {
		return "", err
	}

	basename := fmt.Sprintf("%x%s", sha256.Sum256(body), ext)
	filename := filepath.Join(base, basename)

	return e.media.UploadMedia(filename, bytes.NewBuffer(body))
}

func (e *Eagle) uploadFromURL(base, url string) (string, error) {
	if e.media == nil {
		return url, errors.New("media is not implemented")
	}

	resp, err := e.httpClient.Get(url)
	if err != nil {
		return url, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return url, fmt.Errorf("unexpected status code: %s", resp.Status)
	}

	return e.UploadFile(base, path.Ext(url), resp.Body)
}

func (e *Eagle) safeUploadFromURL(base, url string) string {
	newURL, err := e.uploadFromURL(base, url)
	if err != nil {
		e.log.Warnf("could not upload file %s: %s", url, err.Error())
		return url
	}
	return newURL
}
