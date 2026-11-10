package media

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gabriel-vasile/mimetype"
	"github.com/samber/lo"
	"go.hacdias.com/eagle/core"
)

type Storage interface {
	BaseURL() string
	UploadMedia(filename string, data io.Reader) (string, error)
}

type Format string

const (
	FormatWebP Format = "webp"
	FormatJPEG Format = "jpeg"
)

var formats = []Format{
	FormatWebP, FormatJPEG,
}

type Width int

const (
	// Widths used for transforms
	Width600  Width = 600
	Width900  Width = 900
	Width1800 Width = 1800
	Width3000 Width = 3000
)

var widths = []Width{
	Width600, Width900, Width1800, Width3000,
}

type Media struct {
	httpClient *http.Client

	storage Storage
}

func NewMedia(conf *core.Media) *Media {
	var storage Storage

	if conf.Storage.FileSystem != nil {
		storage = NewFileSystem(conf.Storage.FileSystem)
	} else if conf.Storage.Bunny != nil {
		storage = NewBunny(conf.Storage.Bunny)
	}

	m := &Media{
		httpClient: &http.Client{Timeout: 2 * time.Minute},
		storage:    storage,
	}

	return m
}

func (m *Media) UploadMedia(filename, ext string, reader io.Reader) (string, *core.Photo, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", nil, err
	}

	return m.upload(filename, ext, data)
}

func (m *Media) upload(filename, ext string, data []byte) (string, *core.Photo, error) {
	if m.storage == nil {
		return "", nil, errors.New("media is not implemented")
	}

	if isImage(ext) {
		// Images are never uploaded as they are, as that would publish their metadata.
		p, err := m.uploadImage(filename, data)
		return "", p, err
	}

	// Consistency
	if ext == ".jpg" {
		ext = ".jpeg"
	}

	s, err := m.storage.UploadMedia(filename+ext, bytes.NewBuffer(data))
	return s, nil, err
}

var imageExtensions []string = []string{
	".jpeg",
	".jpg",
	".webp",
	".png",
}

func isImage(ext string) bool {
	return lo.Contains(imageExtensions, strings.ToLower(ext))
}

func (m *Media) uploadImage(filename string, data []byte) (*core.Photo, error) {
	// The photo details are read first, as transforming removes all metadata.
	format, photo, err := probe(data)
	if err != nil {
		return nil, fmt.Errorf("failed to read image: %w", err)
	}

	// Small images keep their format, and are referenced by their full URL
	// instead of "image:".
	if len(data) < 100000 {
		data, err = transform(data, format, 0, 92, 0)
		if err != nil {
			return nil, err
		}

		photo.URL, err = m.storage.UploadMedia(filename+"."+format, bytes.NewReader(data))
		if err != nil {
			return nil, err
		}

		return photo, nil
	}

	// The original is stored as JPEG, keeping its size and colour profile. It
	// can be larger than an upload that was already heavily compressed.
	original, err := transform(data, "jpeg", 0, 92, 0)
	if err != nil {
		return nil, err
	}

	_, err = m.storage.UploadMedia(filename+".jpeg", bytes.NewReader(original))
	if err != nil {
		return nil, err
	}

	// All widths are uploaded, as they are expected to exist for "image:" URLs.
	// Images are not enlarged: images narrower than the width are only
	// re-encoded, keeping their size, and never larger than the original.
	for _, format := range formats {
		for _, width := range widths {
			variant, err := transform(data, string(format), int(width), 80, len(original))
			if err != nil {
				return nil, err
			}

			_, err = m.storage.UploadMedia(filepath.Join("image", strconv.Itoa(int(width)), filename+"."+string(format)), bytes.NewReader(variant))
			if err != nil {
				return nil, err
			}
		}
	}

	photo.URL = "image:" + filename
	return photo, nil
}

func (m *Media) GetImageURL(urlStr string, format Format, width Width) (string, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}

	if u.Scheme != "image" {
		return urlStr, nil
	}

	urlStr = fmt.Sprintf("%s/image/%d/%s.%s", m.storage.BaseURL(), width, u.Opaque, format)
	return urlStr, nil
}

func (m *Media) GetImage(url string) ([]byte, string, error) {
	photoUrl, err := m.GetImageURL(url, FormatJPEG, Width1800)
	if err != nil {
		return nil, "", err
	}

	res, err := m.httpClient.Get(photoUrl)
	if err != nil {
		return nil, "", err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, res.Body)
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("failed to fetch image %s: status %d", photoUrl, res.StatusCode)
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, "", err
	}

	if len(data) > 1000000 {
		data, err = transform(data, "jpeg", 1800, 80, 1000000)
		if err != nil {
			return nil, "", err
		}
	}

	mime := mimetype.Detect(data)
	if mime == nil {
		return nil, "", fmt.Errorf("cannot detect mimetype of %s", url)
	}

	return data, mime.String(), nil
}
