package hooks

import (
	"net/url"

	"github.com/hacdias/eagle/eagle"
	"github.com/hacdias/eagle/fs"
)

type DescriptionGenerator struct {
	fs *fs.FS
}

func NewDescriptionGenerator(fs *fs.FS) *DescriptionGenerator {
	return &DescriptionGenerator{
		fs: fs,
	}
}

func (d *DescriptionGenerator) EntryHook(old, new *eagle.Entry) error {
	if old == nil {
		return d.GenerateDescription(new, false)
	}

	return nil
}

func (d *DescriptionGenerator) GenerateDescription(e *eagle.Entry, replaceDescription bool) error {
	if e.Description != "" && !replaceDescription {
		return nil
	}

	if e.Bookmark != "" {
		e.Description = "Bookmarked a post on " + domain(e.Bookmark)
	} else if e.Reply != "" {
		e.Description = "Replied to a post on " + domain(e.Reply)
	}

	return nil
}

func domain(text string) string {
	u, err := url.Parse(text)
	if err != nil {
		return text
	}

	return u.Host
}
