package media

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/samber/lo"
	"go.hacdias.com/eagle/log"
	"go.uber.org/zap"
)

type Storage interface {
	BaseURL() string
	UploadMedia(filename string, data io.Reader) (string, error)
}

type Transformer interface {
	Transform(reader io.Reader, format string, width, quality int) (io.Reader, error)
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
	Width250  Width = 250
	Width500  Width = 500
	Width1000 Width = 1000
	Width2000 Width = 2000

	// MaximumWidth used for the largest resolution
	MaximumWidth Width = 3000
)

var widths = []Width{
	Width250, Width500, Width1000, Width2000,
}

type Media struct {
	log         *zap.SugaredLogger
	httpClient  *http.Client
	storage     Storage
	transformer Transformer
}

func NewMedia(storage Storage, transformer Transformer) *Media {
	m := &Media{
		log:         log.S().Named("media"),
		httpClient:  &http.Client{Timeout: 2 * time.Minute},
		storage:     storage,
		transformer: transformer,
	}

	return m
}

func (m *Media) UploadMedia(filename, ext string, reader io.Reader) (string, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}

	return m.upload(filename, ext, data)
}

func (m *Media) UploadAnonymousMedia(ext string, reader io.Reader) (string, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}

	filename := fmt.Sprintf("%x", sha256.Sum256(data))
	return m.upload(filename, ext, data)
}

func (m *Media) upload(filename, ext string, data []byte) (string, error) {
	if m.storage == nil {
		return "", errors.New("media is not implemented")
	}

	if isImage(ext) {
		str, err := m.uploadImage(filename, data)
		if err != nil {
			m.log.Errorf("failed to upload image", "filename", filename, "ext", ext, "err", err)
		} else {
			return str, nil
		}
	}

	filename = filepath.Join("media", filename+ext)
	return m.storage.UploadMedia(filename, bytes.NewBuffer(data))
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

func (m *Media) uploadImage(filename string, data []byte) (string, error) {
	if len(data) < 100000 {
		return "", errors.New("image is smaller than 100 KB, ignore")
	}

	if m.transformer == nil {
		return "", errors.New("transformer not implemented")
	}

	imgReader, err := m.transformer.Transform(bytes.NewReader(data), "jpeg", int(MaximumWidth), 100)
	if err != nil {
		return "", err
	}

	_, err = m.storage.UploadMedia(filepath.Join("media", filename+".jpeg"), imgReader)
	if err != nil {
		return "", err
	}

	for _, format := range formats {
		for _, width := range widths {
			imgReader, err = m.transformer.Transform(bytes.NewReader(data), string(format), int(width), 80)
			if err != nil {
				return "", err
			}

			_, err = m.storage.UploadMedia(filepath.Join("img", strconv.Itoa(int(width)), filename+"."+string(format)), imgReader)
			if err != nil {
				return "", err
			}
		}
	}

	return "cdn:/" + filename, nil
}

func (m *Media) GetImageURL(urlStr string, format Format, width Width) (string, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}

	if u.Scheme != "cdn" {
		return urlStr, nil
	}

	id := strings.TrimPrefix(u.Path, "/")
	urlStr = fmt.Sprintf("%s/img/%d/%s.%s", m.storage.BaseURL(), width, id, format)

	return urlStr, nil
}
