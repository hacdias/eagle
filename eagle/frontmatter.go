package eagle

import (
	"time"

	"github.com/hacdias/eagle/pkg/maze"
)

type Context struct {
	// TODO: rename 'name' to 'author' at some point.
	Author    string    `yaml:"name,omitempty"`
	URL       string    `yaml:"url,omitempty"`
	Content   string    `yaml:"content,omitempty"`
	Published time.Time `yaml:"published,omitempty"`
}

type Read struct {
	Name      string `yaml:"name,omitempty"`
	Author    string `yaml:"author,omitempty"`
	Publisher string `yaml:"publisher,omitempty"`
	Pages     int    `yaml:"pages,omitempty"`
	UID       string `yaml:"uid,omitempty"`
}

type FrontMatter struct {
	Title              string         `yaml:"title,omitempty"`
	Description        string         `yaml:"description,omitempty"`
	Draft              bool           `yaml:"draft,omitempty"`
	Date               time.Time      `yaml:"date,omitempty"`
	LastMod            time.Time      `yaml:"lastmod,omitempty"`
	ExpiryDate         time.Time      `yaml:"expiryDate,omitempty"`
	Template           string         `yaml:"template,omitempty"`
	NoSendInteractions bool           `yaml:"noSendInteractions,omitempty"`
	CoverImage         string         `yaml:"coverImage,omitempty"`
	NoIndex            bool           `yaml:"noIndex,omitempty"`
	DisablePagination  bool           `yaml:"disablePagination,omitempty"`
	Tags               []string       `yaml:"tags,omitempty"`
	Categories         []string       `yaml:"categories,omitempty"`
	Layout             string         `yaml:"layout,omitempty"`
	RawLocation        string         `yaml:"rawLocation,omitempty"`
	Location           *maze.Location `yaml:"location,omitempty"`
	Context            *Context       `yaml:"context,omitempty"`
	Syndications       []string       `yaml:"syndications,omitempty"`
	Reply              string         `yaml:"reply,omitempty"`
	Bookmark           string         `yaml:"bookmark,omitempty"`
	Read               *Read          `yaml:"read,omitempty"`
	Rating             int            `yaml:"rating,omitempty"`
}
