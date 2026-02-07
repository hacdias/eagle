package imgproxy

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/spf13/afero"
	"go.hacdias.com/eagle/core"
)

type ImgProxy struct {
	httpClient *http.Client
	fs         *afero.Afero
	endpoint   string
}

func NewImgProxy(conf *core.ImgProxy) *ImgProxy {
	return &ImgProxy{
		endpoint: conf.Endpoint,
		httpClient: &http.Client{
			Timeout: time.Minute * 10,
		},
		fs: &afero.Afero{
			Fs: afero.NewBasePathFs(afero.NewOsFs(), conf.Directory),
		},
	}
}

func (i *ImgProxy) Transform(reader io.Reader, format string, width, quality, maxBytes int) (io.Reader, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	filename := fmt.Sprintf("%x", sha256.Sum256(data))
	err = i.fs.WriteFile(filename, data, 0666)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = i.fs.Remove(filename)
	}()

	urlStr := fmt.Sprintf("%s/rs:auto:%d/q:%d/mb:%d/plain/%s@%s", i.endpoint, width, quality, maxBytes, filename, format)

	res, err := i.httpClient.Get(urlStr)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusOK {
		return nil, errors.New("status code is not ok")
	}

	data, err = io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(data), nil
}
