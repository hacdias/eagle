package entry

import (
	"fmt"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Entry struct {
	FrontMatter
	ID        string
	IsList    bool
	Permalink string
	Content   string
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

type Entries []*Entry

type FrontMatter struct {
	Title       string         `yaml:"title,omitempty"`
	Description string         `yaml:"description,omitempty"`
	URL         string         `yaml:"url,omitempty"` // TODO: remove.
	Draft       bool           `yaml:"draft,omitempty"`
	Date        time.Time      `yaml:"date,omitempty"`
	ExpiryDate  time.Time      `yaml:"expiryDate,omitempty"`
	NoIndex     bool           `yaml:"noIndex,omitempty"`
	Tags        []string       `yaml:"tags,omitempty"` // TODO: remove.
	Other       map[string]any `yaml:",inline"`
}
