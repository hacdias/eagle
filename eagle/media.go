package eagle

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hacdias/eagle/v4/config"
	"github.com/thoas/go-funk"
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

func (e *Eagle) UploadAnonymousMedia(ext string, reader io.Reader) (string, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}

	return e.uploadAnonymous("media", ext, data, false)
}

func (e *Eagle) UploadMedia(filename, ext string, reader io.Reader) (string, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}

	return e.upload("media", filename, ext, data, false)
}

func (e *Eagle) UploadFromURL(base, url string, skipImageCheck bool) (string, error) {
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

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return e.uploadAnonymous(base, path.Ext(url), data, skipImageCheck)
}

func (e *Eagle) SafeUploadFromURL(base, url string, skipImageCheck bool) string {
	newURL, err := e.UploadFromURL(base, url, skipImageCheck)
	if err != nil {
		e.log.Warnf("could not upload file %s: %s", url, err.Error())
		return url
	}
	return newURL
}

func (e *Eagle) uploadAnonymous(base, ext string, data []byte, skipImageCheck bool) (string, error) {
	filename := fmt.Sprintf("%x", sha256.Sum256(data))
	return e.upload(base, filename, ext, data, skipImageCheck)
}

func (e *Eagle) upload(base, filename, ext string, data []byte, skipImageCheck bool) (string, error) {
	if e.media == nil {
		return "", errors.New("media is not implemented")
	}

	if !skipImageCheck && isImage(ext) && base == "media" {
		str, err := e.uploadImage(filename, data)
		if err != nil {
			e.log.Warnf("could not upload image: %s", err.Error())
		} else {
			return str, nil
		}
	}

	filename = filepath.Join(base, filename+ext)
	return e.media.UploadMedia(filename, bytes.NewBuffer(data))
}

var imageExtensions []string = []string{
	".jpeg",
	".jpg",
	".webp",
	".png",
}

func isImage(ext string) bool {
	return funk.ContainsString(imageExtensions, strings.ToLower(ext))
}

func (e *Eagle) uploadImage(filename string, data []byte) (string, error) {
	if len(data) < 100000 {
		return "", errors.New("image is smaller than 100 KB, ignore")
	}

	if e.imgProxy == nil {
		return "", errors.New("imgproxy is not implemented")
	}

	imgReader, err := e.imgProxy.Transform(bytes.NewReader(data), "jpeg", 3000, 100)
	if err != nil {
		return "", err
	}

	_, err = e.media.UploadMedia(filepath.Join("media", filename+".jpeg"), imgReader)
	if err != nil {
		return "", err
	}

	for _, format := range []string{"webp", "jpeg"} {
		for _, width := range []int{250, 500, 1000, 2000} {
			imgReader, err = e.imgProxy.Transform(bytes.NewReader(data), format, width, 80)
			if err != nil {
				return "", err
			}

			_, err = e.media.UploadMedia(filepath.Join("img", strconv.Itoa(width), filename+"."+format), imgReader)
			if err != nil {
				return "", err
			}
		}
	}

	return "cdn:/" + filename, nil
}
