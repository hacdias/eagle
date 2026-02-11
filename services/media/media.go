package media

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
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
	"go.hacdias.com/eagle/log"
	"go.uber.org/zap"
)

type Storage interface {
	BaseURL() string
	UploadMedia(filename string, data io.Reader) (string, error)
}

type Transformer interface {
	Transform(reader io.Reader, format string, width, quality, maxBytes int) (io.Reader, error)
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
)

var widths = []Width{
	Width600, Width900, Width1800,
}

type Media struct {
	log        *zap.SugaredLogger
	httpClient *http.Client

	storage     Storage
	Transformer Transformer
}

func NewMedia(conf *core.Media) *Media {
	var (
		storage     Storage
		transformer Transformer
	)

	if conf.Storage.FileSystem != nil {
		storage = NewFileSystem(conf.Storage.FileSystem)
	} else if conf.Storage.Bunny != nil {
		storage = NewBunny(conf.Storage.Bunny)
	}

	if conf.Transformer.ImgProxy != nil {
		transformer = NewImgProxy(conf.Transformer.ImgProxy)
	}

	m := &Media{
		log:         log.S().Named("media"),
		httpClient:  &http.Client{Timeout: 2 * time.Minute},
		storage:     storage,
		Transformer: transformer,
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
		p, err := m.uploadImage(filename, data)
		if err != nil {
			m.log.Errorf("failed to upload image", "filename", filename, "ext", ext, "err", err)
		} else {
			return "", p, nil
		}
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
	if m.Transformer == nil {
		return nil, errors.New("transformer not implemented")
	}

	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image config: %w", err)
	}

	if len(data) < 100000 {
		var reader io.Reader
		if filepath.Ext(filename) == ".jpeg" || filepath.Ext(filename) == ".jpg" {
			reader = bytes.NewReader(data)
		} else {
			reader, err = m.Transformer.Transform(bytes.NewReader(data), "jpeg", config.Width, 100, 0)
			if err != nil {
				return nil, err
			}
		}

		_, err = m.storage.UploadMedia(filename+".jpeg", reader)
		if err != nil {
			return nil, err
		}
	} else {
		_, err := m.storage.UploadMedia(filename+".jpeg", bytes.NewReader(data))
		if err != nil {
			return nil, err
		}

		for _, format := range formats {
			for _, width := range widths {
				reader, err := m.Transformer.Transform(bytes.NewReader(data), string(format), int(width), 80, 0)
				if err != nil {
					return nil, err
				}

				_, err = m.storage.UploadMedia(filepath.Join("image", strconv.Itoa(int(width)), filename+"."+string(format)), reader)
				if err != nil {
					return nil, err
				}
			}
		}

	}

	return &core.Photo{
		URL:    "image:" + filename,
		Width:  config.Width,
		Height: config.Height,
	}, nil
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

	res, err := http.Get(photoUrl)
	if err != nil {
		return nil, "", err
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, "", err
	}

	err = res.Body.Close()
	if err != nil {
		return nil, "", err
	}

	if len(data) > 1000000 {
		reader, err := m.Transformer.Transform(bytes.NewReader(data), "jpeg", 1800, 80, 1000000)
		if err != nil {
			return nil, "", err
		}

		data, err = io.ReadAll(reader)
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
