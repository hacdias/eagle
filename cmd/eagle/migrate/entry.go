package migrate

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

type Entry struct {
	Path     string
	ID       string
	Content  string
	Metadata Metadata
}

type Webmention struct {
	XRay    `yaml:",inline"`
	WmID    int  `yaml:"wm-id,omitempty" json:"wm-id,omitempty"`
	Private bool `json:"private,omitempty"`
}
type EntryData struct {
	Targets     []string      `json:"targets"`
	Webmentions []*Webmention `json:"webmentions"`
}

// XRay is an xray of an external post. This is the format used to store
// Webmentions and ReplyTo context.
type XRay struct {
	Type    string    `yaml:"type,omitempty" json:"type,omitempty"`
	URL     string    `yaml:"url,omitempty" json:"url,omitempty"`
	Name    string    `yaml:"name,omitempty" json:"name,omitempty"`
	Content string    `yaml:"content,omitempty" json:"content,omitempty"`
	Date    time.Time `yaml:"date,omitempty" json:"date,omitempty"`
	Author  *Author   `yaml:"author,omitempty" json:"author,omitempty"`
}

type Author struct {
	Name  string `yaml:"name,omitempty" json:"name,omitempty"`
	URL   string `yaml:"url,omitempty" json:"url,omitempty"`
	Photo string `yaml:"photo,omitempty" json:"photo,omitempty"`
}

type Metadata struct {
	DataID      string        `yaml:"dataId,omitempty"`
	Title       string        `yaml:"title,omitempty"`
	Description string        `yaml:"description,omitempty"`
	Tags        []string      `yaml:"tags,omitempty"`
	Date        time.Time     `yaml:"date,omitempty"`
	Lastmod     time.Time     `yaml:"lastmod,omitempty"`
	ExpiryDate  time.Time     `yaml:"expiryDate,omitempty"`
	Syndication []string      `yaml:"syndication,omitempty"`
	ReplyTo     *XRay         `yaml:"replyTo,omitempty"`
	URL         string        `yaml:"url,omitempty"`
	Aliases     []string      `yaml:"aliases,omitempty"`
	Emoji       string        `yaml:"emoji,omitempty"`
	Layout      string        `yaml:"layout,omitempty"`
	NoMentions  bool          `yaml:"noMentions,omitempty"`
	Cover       *Picture      `yaml:"cover,omitempty"`
	Draft       bool          `yaml:"draft,omitempty"`
	Growth      string        `yaml:"growth,omitempty"`
	Photo       []interface{} `yaml:"photo,omitempty"`
	PhotoClass  string        `yaml:"photoClass,omitempty"`
}

type Picture struct {
	Slug string
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

func (e *Entry) Slug() string {
	cleanID := strings.TrimPrefix(e.ID, "/")
	cleanID = strings.TrimSuffix(cleanID, "/")
	a := strings.Split(cleanID, "/")
	return a[len(a)-1]
}

func getAllEntries() ([]*Entry, error) {
	entries := []*Entry{}
	err := filepath.Walk(oldContentPath, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(p, ".md") {
			return nil
		}

		id := strings.TrimPrefix(p, oldContentPath)
		id = strings.TrimSuffix(id, ".md")
		id = strings.TrimSuffix(id, "_index")
		id = strings.TrimSuffix(id, "index")
		id = strings.TrimPrefix(id, "/")
		id = strings.TrimSuffix(id, "/")
		id = "/" + id

		entry, err := getEntry(id)
		if err != nil {
			return fmt.Errorf("error getting %s: %w", id, err)
		}

		entries = append(entries, entry)
		return nil
	})

	return entries, err
}

func getEntry(id string) (*Entry, error) {
	filepath, err := guessPath(id)
	if err != nil {
		return nil, err
	}

	raw, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	entry, err := parseEntry(id, string(raw))
	if err != nil {
		return nil, err
	}

	entry.Path = filepath
	return entry, nil
}

func parseEntry(id, raw string) (*Entry, error) {
	splits := strings.SplitN(raw, "\n---", 2)
	if len(splits) != 2 {
		return nil, errors.New("could not parse file: splits !== 2")
	}

	entry := &Entry{
		ID:       id,
		Content:  strings.TrimSpace(splits[1]),
		Metadata: Metadata{},
	}

	err := yaml.Unmarshal([]byte(splits[0]), &entry.Metadata)
	if err != nil {
		return nil, err
	}

	return entry, nil
}

func guessPath(id string) (string, error) {
	path := filepath.Join(oldContentPath, id, "index.md")
	if _, err := os.Stat(path); err == nil {
		return path, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}

	path = filepath.Join(oldContentPath, id, "_index.md")
	if _, err := os.Stat(path); err == nil {
		return path, nil
	} else {
		return "", err
	}
}

func copy(src, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer source.Close()

	err = os.MkdirAll(filepath.Dir(dst), 0777)
	if err != nil {
		return 0, err
	}

	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}
