package core

import (
	"fmt"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v3"
)

type Entry struct {
	FrontMatter
	ID        string
	IsList    bool
	Permalink string
	Content   string
}

func (e *Entry) Deleted() bool {
	if e.FrontMatter.ExpiryDate.IsZero() {
		return false
	}

	return e.FrontMatter.ExpiryDate.Before(time.Now())
}

func (e *Entry) String() (string, error) {
	fr, err := yaml.Marshal(&e.FrontMatter)
	if err != nil {
		return "", err
	}

	text := fmt.Sprintf("---\n%s---\n\n%s\n", string(fr), strings.TrimSpace(e.Content))
	text = strings.TrimSpace(text) + "\n"
	return normalizeNewlines(text), nil
}

func (e *Entry) TextContent() string {
	return makePlainText(e.Content)
}

type Entries []*Entry

type FrontMatter struct {
	Title       string         `yaml:"title,omitempty"`
	Description string         `yaml:"description,omitempty"`
	URL         string         `yaml:"url,omitempty"`
	Draft       bool           `yaml:"draft,omitempty"`
	Date        time.Time      `yaml:"date,omitempty"`
	ExpiryDate  time.Time      `yaml:"expiryDate,omitempty"`
	NoIndex     bool           `yaml:"noIndex,omitempty"`
	Tags        []string       `yaml:"tags,omitempty"`
	Other       map[string]any `yaml:",inline"`
}

type EntryHook interface {
	EntryHook(old, new *Entry) error
}

type Notifier interface {
	Info(msg string)
	Error(err error)
}

type GuestbookEntry struct {
	ID      string    `json:"-"`
	Name    string    `json:"name,omitempty"`
	Website string    `json:"website,omitempty"`
	Content string    `json:"content,omitempty"`
	Date    time.Time `json:"date,omitempty"`
}

type GuestbookEntries []GuestbookEntry
