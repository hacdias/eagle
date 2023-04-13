package helpers

import (
	"errors"
	"fmt"

	"github.com/hacdias/eagle/core"
	"github.com/hacdias/eagle/log"
	"github.com/hacdias/eagle/pkg/xray"
)

type ContextFetcher struct {
	xray *xray.XRay
	fs   *core.FS
}

func NewContextFetcher(c *core.Config, fs *core.FS) (*ContextFetcher, error) {
	xrayConf := &xray.Config{
		Endpoint:  c.XRay.Endpoint,
		UserAgent: fmt.Sprintf("Eagle/0.0 (%s) XRay", c.ID()),
	}

	xray, err := xray.NewXRay(xrayConf, log.S().Named("xray"))
	return &ContextFetcher{
		xray: xray,
		fs:   fs,
	}, err
}

func (c *ContextFetcher) EnsureXRay(e *core.Entry, replace bool) error {
	if e.Context != nil && !replace {
		return nil
	}

	if e.Reply == "" {
		return nil
	}

	parsed, _, err := c.xray.Fetch(e.Reply)
	if err != nil {
		if errors.Is(err, xray.ErrPostNotFound) {
			return nil
		}

		return fmt.Errorf("could not fetch context xray for %s: %w", e.ID, err)
	}

	_, err = c.fs.TransformEntry(e.ID, func(e *core.Entry) (*core.Entry, error) {
		e.Context = &core.Context{
			Author:    parsed.Author.Name,
			Content:   parsed.Content,
			Published: parsed.Published,
			URL:       parsed.URL,
		}

		if parsed.URL != "" && parsed.URL != e.Reply {
			e.Reply = parsed.URL
		}

		return e, nil
	})

	if err != nil {
		return err
	}

	return nil
}
