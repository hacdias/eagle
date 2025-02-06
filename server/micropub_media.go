package server

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"

	"github.com/gabriel-vasile/mimetype"
	"github.com/samber/lo"
	"go.hacdias.com/indielib/micropub"
)

const (
	micropubMediaPath = "/micropub/media"
)

func (s *Server) makeMicropubMedia() http.Handler {
	return micropub.NewMediaHandler(func(file multipart.File, header *multipart.FileHeader) (string, error) {
		data, err := io.ReadAll(file)
		if err != nil {
			return "", err
		}

		ext := filepath.Ext(header.Filename)
		if ext == "" {
			// NOTE: I'm not using http.DetectContentType because it depends
			// on OS specific mime type registries. Thus, it was being unreliable
			// on different OSes.
			contentType := header.Header.Get("Content-Type")
			mime := mimetype.Lookup(contentType)
			if mime.Is("application/octet-stream") {
				mime = mimetype.Detect(data)
			}

			if mime == nil {
				return "", errors.New("cannot deduce mimetype")
			}

			ext = mime.Extension()
		}

		filename := fmt.Sprintf("cache://%x%s", sha256.Sum256(data), ext)

		added := s.mediaCache.Set(filename, data)
		if !added {
			return "", errors.New("failed to add item to cache")
		}

		return filename, nil
	}, func(r *http.Request, scope string) bool {
		return lo.Contains(s.getScopes(r), scope)
	})
}
