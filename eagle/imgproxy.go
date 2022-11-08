package eagle

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/afero"
)

type ImgProxy struct {
	httpClient *http.Client
	fs         *afero.Afero
	endpoint   string
}

func (i *ImgProxy) Transform(reader io.Reader, format string, width, quality int) (io.Reader, error) {
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

	urlStr := fmt.Sprintf("%s/rs:auto:%d/q:%d/plain/%s@%s", i.endpoint, width, quality, filename, format)

	res, err := i.httpClient.Get(urlStr)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, errors.New("status code is not ok")
	}

	data, err = io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(data), nil
}
