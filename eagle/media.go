package eagle

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/hacdias/eagle/config"
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
