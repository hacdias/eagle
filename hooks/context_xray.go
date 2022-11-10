package hooks

import (
	"fmt"

	"github.com/hacdias/eagle/v4/eagle"
	"github.com/hacdias/eagle/v4/fs"
	"github.com/hacdias/eagle/v4/log"
	"github.com/hacdias/eagle/v4/media"
	"github.com/hacdias/eagle/v4/pkg/mf2"
	"github.com/hacdias/eagle/v4/pkg/xray"
	"github.com/vartanbeno/go-reddit/v2/reddit"
)

type ContextXRay struct {
	xray  *xray.XRay
	fs    *fs.FS
	media *media.Media
}

func NewContentXRay(c *eagle.Config, fs *fs.FS, media *media.Media) (*ContextXRay, error) {
	xrayConf := &xray.Config{
		Endpoint:  c.XRay.Endpoint,
		UserAgent: fmt.Sprintf("Eagle/0.0 (%s) XRay", c.ID()),
	}

	if c.XRay.Twitter && c.Twitter != nil {
		xrayConf.Twitter = &xray.Twitter{
			Key:         c.Twitter.Key,
			Secret:      c.Twitter.Secret,
			Token:       c.Twitter.Token,
			TokenSecret: c.Twitter.TokenSecret,
		}
	}

	if c.XRay.Reddit && c.Reddit != nil {
		xrayConf.Reddit = &reddit.Credentials{
			ID:       c.Reddit.App,
			Secret:   c.Reddit.Secret,
			Username: c.Reddit.User,
			Password: c.Reddit.Password,
		}
	}

	xray, err := xray.NewXRay(xrayConf, log.S().Named("xray"))
	return &ContextXRay{
		xray:  xray,
		fs:    fs,
		media: media,
	}, err
}

func (c *ContextXRay) EntryHook(e *eagle.Entry, isNew bool) error {
	return c.EnsureXRay(e, false)
}

func (c *ContextXRay) EnsureXRay(e *eagle.Entry, replace bool) error {
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

	sidecar, err := c.fs.GetSidecar(e)
	if err != nil {
		return fmt.Errorf("could not fetch sidecar for %s: %w", e.ID, err)
	}

	if sidecar.Context != nil && !replace {
		return nil
	}

	parsed, _, err := c.xray.Fetch(urlStr)
	if err != nil {
		return fmt.Errorf("could not fetch context xray for %s: %w", e.ID, err)
	}

	if parsed.Author.Photo != "" && c.media != nil {
		parsed.Author.Photo = c.media.SafeUploadFromURL("wm", parsed.Author.Photo, true)
	}

	return c.fs.UpdateSidecar(e, func(data *eagle.Sidecar) (*eagle.Sidecar, error) {
		data.Context = parsed
		return data, nil
	})
}
