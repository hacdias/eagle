package core

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v3"
)

const moreSeparator = "<!--more-->"

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

func (e *Entry) Summary() string {
	if strings.Contains(e.Content, moreSeparator) {
		firstPart := strings.Split(e.Content, moreSeparator)[0]
		return strings.TrimSpace(makePlainText(firstPart))
	} else if content := e.TextContent(); content != "" {
		return truncateStringWithEllipsis(content, 300)
	} else {
		return content
	}
}

func (e *Entry) String() (string, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)

	err := enc.Encode(&e.FrontMatter)
	if err != nil {
		return "", err
	}

	text := fmt.Sprintf("---\n%s---\n\n%s\n", buf.String(), strings.TrimSpace(e.Content))
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
	Lastmod     time.Time      `yaml:"lastmod,omitempty"`
	ExpiryDate  time.Time      `yaml:"expiryDate,omitempty"`
	NoIndex     bool           `yaml:"noIndex,omitempty"`
	Categories  []string       `yaml:"categories,omitempty"`
	Tags        []string       `yaml:"tags,omitempty"`
	Other       map[string]any `yaml:",inline"`
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
