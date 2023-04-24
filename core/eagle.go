package core

import (
	"fmt"
	"strings"
	"time"

	"github.com/hacdias/maze"
	yaml "gopkg.in/yaml.v2"
)

type Entry struct {
	FrontMatter
	Path      string // The original path of the file. Might be empty.
	ID        string
	Permalink string
	Content   string
}

func (e *Entry) IsList() bool {
	return strings.Contains(e.Path, "_index.md")
}

func (e *Entry) Deleted() bool {
	if e.FrontMatter.ExpiryDate.IsZero() {
		return false
	}

	return e.FrontMatter.ExpiryDate.Before(time.Now())
}

func (e *Entry) String() (string, error) {
	val, err := yaml.Marshal(&e.FrontMatter)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("---\n%s---\n\n%s\n", string(val), strings.TrimSpace(e.Content)), nil
}

func (e *Entry) TextContent() string {
	return makePlainText(e.Content)
}

type Entries []*Entry

type FrontMatter struct {
	Title              string         `yaml:"title,omitempty"`
	Description        string         `yaml:"description,omitempty"`
	Draft              bool           `yaml:"draft,omitempty"`
	Date               time.Time      `yaml:"date,omitempty"`
	LastMod            time.Time      `yaml:"lastmod,omitempty"`
	ExpiryDate         time.Time      `yaml:"expiryDate,omitempty"`
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

type Context struct {
	Author    string    `yaml:"name,omitempty"` // TODO: rename 'name' to 'author' at some point.
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

type EntryHook interface {
	EntryHook(old, new *Entry) error
}

type Notifier interface {
	Info(msg string)
	Error(err error)
}

type WebFinger struct {
	Subject string          `json:"subject"`
	Aliases []string        `json:"aliases,omitempty"`
	Links   []WebFingerLink `json:"links,omitempty"`
}

type WebFingerLink struct {
	Href     string `json:"href"`
	Rel      string `json:"rel,omitempty"`
	Type     string `json:"type,omitempty"`
	Template string `json:"template,omitempty"`
}

type GuestbookEntry struct {
	Name    string    `json:"name,omitempty"`
	Website string    `json:"website,omitempty"`
	Content string    `json:"content,omitempty"`
	Date    time.Time `json:"date,omitempty"`
}

type GuestbookEntries []GuestbookEntry
