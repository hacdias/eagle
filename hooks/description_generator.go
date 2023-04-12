package hooks

import (
	"github.com/hacdias/eagle/eagle"
	"github.com/hacdias/eagle/fs"
	"github.com/hacdias/eagle/pkg/mf2"
	"github.com/hacdias/eagle/util"
)

var typeToDescription = map[mf2.Type]string{
	mf2.TypeReply:    "Replied to ",
	mf2.TypeLike:     "Liked ",
	mf2.TypeRepost:   "Reposted ",
	mf2.TypeBookmark: "Bookmarked ",
	mf2.TypeAte:      "Just ate: ",
	mf2.TypeDrank:    "Just drank: ",
}

type DescriptionGenerator struct {
	fs *fs.FS
}

func NewDescriptionGenerator(fs *fs.FS) *DescriptionGenerator {
	return &DescriptionGenerator{
		fs: fs,
	}
}

func (d *DescriptionGenerator) EntryHook(old, new *eagle.Entry) error {
	if old == nil && new.Listing == nil {
		return d.GenerateDescription(new, false)
	}

	return nil
}

func (d *DescriptionGenerator) GenerateDescription(e *eagle.Entry, replaceDescription bool) error {
	if e.Description != "" && !replaceDescription {
		return nil
	}

	if e.Bookmark != "" {
		e.Description = "Bookmarked a post on " + util.Domain(e.Bookmark)
	} else if e.Reply != "" {
		e.Description = "Replied to a post on " + util.Domain(e.Reply)
	}

	return nil
}
