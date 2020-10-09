package services

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hacdias/eagle/config"
)

type Media struct {
	config.BunnyCDN
}

func (m *Media) Upload(filename string, data io.Reader) (string, error) {
	if !strings.HasPrefix(filename, "/") {
		filename = "/" + filename
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*10)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, "https://storage.bunnycdn.com/"+m.Zone+filename, data)
	if err != nil {
		return "", err
	}

	req.Header.Set("AccessKey", m.Key)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	if res.StatusCode != http.StatusCreated {
		return "", errors.New("status code is not ok")
	}

	return m.Base + filename, nil
}
