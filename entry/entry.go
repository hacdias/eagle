package entry

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Entry struct {
	FrontMatter
	ID        string
	Permalink string
	Content   string
}

func (e *Entry) String() (string, error) {
	fr := bytes.Buffer{}
	enc := yaml.NewEncoder(&fr)
	enc.SetIndent(2)

	err := enc.Encode(&e.FrontMatter)
	if err != nil {
		return "", err
	}

	text := fmt.Sprintf("---\n%s---\n\n%s\n", fr.String(), strings.TrimSpace(e.Content))
	text = strings.TrimSpace(text) + "\n"
	return normalizeNewlines(text), nil
}

type Entries []*Entry

type FrontMatter struct {
	Title       string         `yaml:"title,omitempty"`
	Description string         `yaml:"description,omitempty"`
	URL         string         `yaml:"url,omitempty"` // TODO: remove.
	Draft       bool           `yaml:"draft,omitempty"`
	Date        time.Time      `yaml:"date,omitempty"`
	Lastmod     time.Time      `yaml:"lastmod,omitempty"`
	ExpiryDate  time.Time      `yaml:"expiryDate,omitempty"`
	NoIndex     bool           `yaml:"noIndex,omitempty"`
	Tags        []string       `yaml:"tags,omitempty"` // TODO: remove.
	Other       map[string]any `yaml:",inline"`
}
