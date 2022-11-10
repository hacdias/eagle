package media

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
	"time"

	"github.com/hacdias/eagle/v4/eagle"
	"github.com/hacdias/eagle/v4/log"
	"github.com/thoas/go-funk"
	"go.uber.org/zap"
)

// wip: make interface for bunny (media storage) and imgproxy (media transformer)? make sure it's initialized evertywhere
type Media struct {
	httpClient *http.Client
	log        *zap.SugaredLogger
	bunnyCDN   *BunnyCDN
	imgProxy   *ImgProxy
}

func (m *Media) BaseURL() string {
	if m.bunnyCDN == nil {
		return ""
	}
	return m.bunnyCDN.Base
}

func NewMedia(conf *eagle.Config) *Media {
	m := &Media{
		httpClient: &http.Client{Timeout: 2 * time.Minute},
		log:        log.S().Named("media"),
	}

	if conf.BunnyCDN != nil {
		m.bunnyCDN = NewBunnyCDN(conf.BunnyCDN)
	}

	if conf.ImgProxy != nil {
		m.imgProxy = NewImgProxy(conf.ImgProxy)
	}

	return m
}

func (m *Media) UploadAnonymousMedia(ext string, reader io.Reader) (string, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}

	return m.uploadAnonymous("media", ext, data, false)
}

func (m *Media) UploadMedia(filename, ext string, reader io.Reader) (string, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}

	return m.upload("media", filename, ext, data, false)
}

func (m *Media) UploadFromURL(base, url string, skipImageCheck bool) (string, error) {
	if m.bunnyCDN == nil {
		return url, errors.New("media is not implemented")
	}

	resp, err := m.httpClient.Get(url)
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

	return m.uploadAnonymous(base, path.Ext(url), data, skipImageCheck)
}

func (m *Media) SafeUploadFromURL(base, url string, skipImageCheck bool) string {
	newURL, err := m.UploadFromURL(base, url, skipImageCheck)
	if err != nil {
		m.log.Warnf("could not upload file %s: %s", url, err.Error())
		return url
	}
	return newURL
}

func (e *Media) uploadAnonymous(base, ext string, data []byte, skipImageCheck bool) (string, error) {
	filename := fmt.Sprintf("%x", sha256.Sum256(data))
	return e.upload(base, filename, ext, data, skipImageCheck)
}

func (m *Media) upload(base, filename, ext string, data []byte, skipImageCheck bool) (string, error) {
	if m.bunnyCDN == nil {
		return "", errors.New("media is not implemented")
	}

	if !skipImageCheck && isImage(ext) && base == "media" {
		str, err := m.uploadImage(filename, data)
		if err != nil {
			m.log.Warnf("could not upload image: %s", err.Error())
		} else {
			return str, nil
		}
	}

	filename = filepath.Join(base, filename+ext)
	return m.bunnyCDN.UploadMedia(filename, bytes.NewBuffer(data))
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

func (m *Media) uploadImage(filename string, data []byte) (string, error) {
	if len(data) < 100000 {
		return "", errors.New("image is smaller than 100 KB, ignore")
	}

	if m.imgProxy == nil {
		return "", errors.New("imgproxy is not implemented")
	}

	imgReader, err := m.imgProxy.Transform(bytes.NewReader(data), "jpeg", 3000, 100)
	if err != nil {
		return "", err
	}

	_, err = m.bunnyCDN.UploadMedia(filepath.Join("media", filename+".jpeg"), imgReader)
	if err != nil {
		return "", err
	}

	for _, format := range []string{"webp", "jpeg"} {
		for _, width := range []int{250, 500, 1000, 2000} {
			imgReader, err = m.imgProxy.Transform(bytes.NewReader(data), format, width, 80)
			if err != nil {
				return "", err
			}

			_, err = m.bunnyCDN.UploadMedia(filepath.Join("img", strconv.Itoa(width), filename+"."+format), imgReader)
			if err != nil {
				return "", err
			}
		}
	}

	return "cdn:/" + filename, nil
}
