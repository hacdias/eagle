package media

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hacdias/eagle/v4/eagle"
)

type BunnyCDN struct {
	httpClient *http.Client
	*eagle.BunnyCDN
}

func NewBunnyCDN(conf *eagle.BunnyCDN) *BunnyCDN {
	return &BunnyCDN{
		BunnyCDN: conf,
		httpClient: &http.Client{
			Timeout: time.Minute * 10,
		},
	}
}

func (m *BunnyCDN) UploadMedia(filename string, data io.Reader) (string, error) {
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
