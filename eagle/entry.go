package eagle

import (
	"fmt"
	"strings"
	"time"

	"github.com/hacdias/eagle/yaml"
)

type Entry struct {
	Path       string // The original path of the file. Might be empty.
	ID         string
	Permalink  string
	Content    string
	RawContent string
	Metadata   EntryMetadata
}

type EntryMetadata struct {
	Title       string                `yaml:"title,omitempty"`
	Description string                `yaml:"description,omitempty"`
	Tags        []string              `yaml:"tags,omitempty"`
	Date        time.Time             `yaml:"date,omitempty"`
	Lastmod     time.Time             `yaml:"lastmod,omitempty"`
	ExpiryDate  time.Time             `yaml:"expiryDate,omitempty"`
	Syndication []string              `yaml:"syndication,omitempty"`
	ReplyTo     *EmbeddedEntry        `yaml:"replyTo,omitempty"`
	URL         string                `yaml:"url,omitempty"`
	Aliases     []string              `yaml:"aliases,omitempty"`
	Emoji       string                `yaml:"emoji,omitempty"`
	Layout      string                `yaml:"layout,omitempty"`
	NoIndex     bool                  `yaml:"noIndex,omitempty"`
	NoMentions  bool                  `yaml:"noMentions,omitempty"`
	Math        bool                  `yaml:"math,omitempty"`
	Mermaid     bool                  `yaml:"mermaid,omitempty"`
	Pictures    []*EntryPicture       `yaml:"pictures,omitempty"`
	Cover       *EntryPicture         `yaml:"cover,omitempty"`
	Draft       bool                  `yaml:"draft,omitempty"`
	Reading     *EntryReading         `yaml:"reading,omitempty"`
	Growth      string                `yaml:"growth,omitempty"`
	Menu        map[string]*EntryMenu `yaml:"menu,omitempty"`
}

func (e *Entry) Section() string {
	cleanID := strings.TrimPrefix(e.ID, "/")
	cleanID = strings.TrimSuffix(cleanID, "/")

	section := ""
	if strings.Count(cleanID, "/") >= 1 {
		section = strings.Split(cleanID, "/")[0]
	}
	return section
}

func (e *Entry) Deleted() bool {
	if e.Metadata.ExpiryDate.IsZero() {
		return false
	}

	return e.Metadata.ExpiryDate.Before(time.Now())
}

func (e *Entry) Date() string {
	return e.Metadata.Date.Format(time.RFC3339)
}

func (e *Entry) String() (string, error) {
	val, err := yaml.Marshal(&e.Metadata)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("---\n%s---\n\n%s\n", string(val), e.Content), nil
}

type EmbeddedEntry struct {
	WmID    int          `yaml:"wm-id,omitempty"`
	Type    string       `yaml:"type,omitempty"`
	URL     string       `yaml:"url,omitempty"`
	Name    string       `yaml:"name,omitempty"`
	Content string       `yaml:"content,omitempty"`
	Date    time.Time    `yaml:"date,omitempty"`
	Author  *EntryAuthor `yaml:"author,omitempty"`
}

type EntryPicture struct {
	Title string `yaml:"title,omitempty"`
	Slug  string `yaml:"slug,omitempty"`
	Hide  bool   `yaml:"hide,omitempty"`
}

type EntryAuthor struct {
	Name  string `yaml:"name,omitempty" json:"name"`
	URL   string `yaml:"url,omitempty" json:"url"`
	Photo string `yaml:"photo,omitempty" json:"photo"`
}

type EntryReading struct {
	Name   string    `yaml:"name,omitempty"`
	Author string    `yaml:"author,omitempty"`
	ISBN   string    `yaml:"isbn,omitempty"`
	Date   time.Time `yaml:"date,omitempty"`
	Tags   []string  `yaml:"tags,omitempty"`
}

type EntryMenu struct {
	Weight int    `yaml:"weight,omitempty"`
	Name   string `yaml:"name,omitempty"`
	Pre    string `yaml:"pre,omitempty"`
}
