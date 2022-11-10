package bunny

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hacdias/eagle/v4/eagle"
)

type Bunny struct {
	httpClient *http.Client
	conf       *eagle.BunnyCDN
}

func NewBunny(conf *eagle.BunnyCDN) *Bunny {
	return &Bunny{
		conf: conf,
		httpClient: &http.Client{
			Timeout: time.Minute * 10,
		},
	}
}

func (m *Bunny) BaseURL() string {
	return m.conf.Base
}

func (m *Bunny) UploadMedia(filename string, data io.Reader) (string, error) {
	if !strings.HasPrefix(filename, "/") {
		filename = "/" + filename
	}

	req, err := http.NewRequest(http.MethodPut, "https://storage.bunnycdn.com/"+m.conf.Zone+filename, data)
	if err != nil {
		return "", err
	}

	req.Header.Set("AccessKey", m.conf.Key)

	res, err := m.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated {
		return "", errors.New("status code is not ok")
	}

	return m.conf.Base + filename, nil
}
