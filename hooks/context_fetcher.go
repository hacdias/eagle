package hooks

import (
	"errors"
	"fmt"

	"github.com/hacdias/eagle/eagle"
	"github.com/hacdias/eagle/fs"
	"github.com/hacdias/eagle/log"
	"github.com/hacdias/eagle/pkg/mf2"
	"github.com/hacdias/eagle/pkg/xray"
)

type ContextFetcher struct {
	xray *xray.XRay
	fs   *fs.FS
}

func NewContextFetcher(c *eagle.Config, fs *fs.FS) (*ContextFetcher, error) {
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

func (c *ContextFetcher) EntryHook(_, e *eagle.Entry) error {
	return c.EnsureXRay(e, false)
}

func (c *ContextFetcher) EnsureXRay(e *eagle.Entry, replace bool) error {
	mm := e.Helper()
	typ := mm.PostType()

	switch typ {
	case mf2.TypeLike,
		mf2.TypeRepost,
		mf2.TypeReply,
		mf2.TypeRsvp:
		// Keep going
	default:
		return nil
	}

	property := mm.TypeProperty()
	if typ == mf2.TypeRsvp {
		property = "in-reply-to"
	}

	urlStr := mm.String(property)
	if urlStr == "" {
		return fmt.Errorf("expected context url to be non-empty for %s", e.ID)
	}

	if e.Context != nil && !replace {
		return nil
	}

	parsed, _, err := c.xray.Fetch(urlStr)
	if err != nil {
		if errors.Is(err, xray.ErrPostNotFound) {
			return nil
		}

		return fmt.Errorf("could not fetch context xray for %s: %w", e.ID, err)
	}

	_, err = c.fs.TransformEntry(e.ID, func(e *eagle.Entry) (*eagle.Entry, error) {
		e.Context = &eagle.Context{
			Author:    parsed.Author.Name,
			Content:   parsed.Content,
			Published: parsed.Published,
			URL:       parsed.URL,
		}

		if parsed.URL != "" && parsed.URL != urlStr {
			e.Properties[property] = parsed.URL
		}

		return e, nil
	})

	if err != nil {
		return err
	}

	return nil
}
