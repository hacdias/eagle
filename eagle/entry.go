package eagle

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
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
	Title       string          `yaml:"title,omitempty"`
	Description string          `yaml:"description,omitempty"`
	Tags        []string        `yaml:"tags,omitempty"`
	Date        time.Time       `yaml:"date,omitempty"`
	Lastmod     time.Time       `yaml:"lastmod,omitempty"`
	ExpiryDate  time.Time       `yaml:"expiryDate,omitempty"`
	Syndication []string        `yaml:"syndication,omitempty"`
	ReplyTo     *EmbeddedEntry  `yaml:"replyTo,omitempty"`
	URL         string          `yaml:"url,omitempty"`
	Aliases     []string        `yaml:"aliases,omitempty"`
	Emoji       string          `yaml:"emoji,omitempty"`
	Layout      string          `yaml:"layout,omitempty"`
	NoIndex     bool            `yaml:"noIndex,omitempty"`
	NoMentions  bool            `yaml:"noMentions,omitempty"`
	Math        bool            `yaml:"math,omitempty"`
	Mermaid     bool            `yaml:"mermaid,omitempty"`
	Pictures    []*EntryPicture `yaml:"pictures,omitempty"`
	Cover       *EntryPicture   `yaml:"cover,omitempty"`
	Mentions    []EntryMention  `yaml:"mentions,omitempty"`
	Draft       bool            `yaml:"draft,omitempty"`
	Reading     *EntryReading   `yaml:"reading,omitempty"`
}

// Bundle transforms the entry into a page bundle.
func (e *Entry) Bundle() error {
	if e.Path == "" {
		return fmt.Errorf("post %s does not contain a path", e.ID)
	}

	if strings.HasSuffix(e.Path, "index.md") {
		// already a page bundle
		return nil
	}

	dir := strings.TrimSuffix(e.Path, filepath.Ext(e.Path))
	file := filepath.Join(dir, "index.md")

	err := os.MkdirAll(dir, 0777)
	if err != nil {
		return err
	}

	err = os.Rename(e.Path, file)
	if err != nil {
		return err
	}

	e.Path = file
	return nil
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

type EntryMention struct {
	Href string `yaml:"href,omitempty"`
	Name string `yaml:"name,omitempty"`
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

type SearchQuery struct {
	Query    string
	Sections []string // if empty, matches all sections
	Draft    bool
}

type SearchIndex interface {
	ResetIndex() error
	Add(entries ...*Entry) error
	Remove(entries ...*Entry) error
	Search(query *SearchQuery, page int) ([]interface{}, error)
}

type EntryManager struct {
	sync.RWMutex

	search SearchIndex
	store  StorageService
	domain string
	source string
}

func (m *EntryManager) GetEntry(id string) (*Entry, error) {
	m.RLock()
	defer m.RUnlock()

	id = m.cleanID(id)
	filepath, err := m.guessPath(id)
	if err != nil {
		return nil, err
	}

	raw, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	entry, err := m.ParseEntry(id, string(raw))
	if err != nil {
		return nil, err
	}

	entry.Path = filepath
	return entry, nil
}

func (m *EntryManager) ParseEntry(id, raw string) (*Entry, error) {
	id = m.cleanID(id)
	splits := strings.SplitN(raw, "\n---", 2)
	if len(splits) != 2 {
		return nil, errors.New("could not parse file: splits !== 2")
	}

	permalink, err := m.makePermalink(id)
	if err != nil {
		return nil, err
	}

	entry := &Entry{
		ID:        id,
		Permalink: permalink,
		Content:   strings.TrimSpace(splits[1]),
		Metadata:  EntryMetadata{},
	}

	entry.RawContent = entry.Content
	err = yaml.Unmarshal([]byte(splits[0]), &entry.Metadata)
	if err != nil {
		return nil, err
	}

	return entry, nil
}

func (m *EntryManager) SaveEntry(entry *Entry) error {
	m.Lock()
	defer m.Unlock()

	entry.ID = m.cleanID(entry.ID)
	if entry.Path == "" {
		path, err := m.guessPath(entry.ID)
		if err != nil {
			if !os.IsNotExist(err) {
				return err

			}
			// Default path for new files is content/{slug}/index.md
			path = filepath.Join(m.source, "content", entry.ID, "index.md")
		}
		entry.Path = path
	}

	err := os.MkdirAll(filepath.Dir(entry.Path), 0777)
	if err != nil {
		return err
	}

	str, err := m.EntryToString(entry)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(entry.Path, []byte(str), 0644)
	if err != nil {
		return fmt.Errorf("could not save entry: %w", err)
	}

	err = m.store.Persist("hugo: update "+entry.ID, entry.Path)
	if err != nil {
		return fmt.Errorf("could not save entry: %w", err)
	}

	if m.search != nil {
		_ = m.search.Add(entry)
	}

	return nil
}

func (m *EntryManager) EntryToString(entry *Entry) (string, error) {
	val, err := yaml.Marshal(&entry.Metadata)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("---\n%s---\n\n%s\n", string(val), entry.Content), nil
}

func (m *EntryManager) DeleteEntry(entry *Entry) error {
	entry.Metadata.ExpiryDate = time.Now()

	if m.search != nil {
		_ = m.search.Remove(entry)
	}

	return m.SaveEntry(entry)
}

func (m *EntryManager) GetAll() ([]*Entry, error) {
	m.RLock()
	defer m.RUnlock()

	entries := []*Entry{}
	content := path.Join(m.source, "content")

	err := filepath.Walk(content, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(p, ".md") {
			return nil
		}

		id := strings.TrimPrefix(p, content)
		id = strings.TrimSuffix(id, ".md")
		id = strings.TrimSuffix(id, "_index")
		id = strings.TrimSuffix(id, "index")

		entry, err := m.GetEntry(id)
		if err != nil {
			return err
		}

		entries = append(entries, entry)
		return nil
	})

	return entries, err
}

func (m *EntryManager) Search(query *SearchQuery, page int) ([]interface{}, error) {
	if m.search == nil {
		return []interface{}{}, nil
	}

	return m.search.Search(query, page)
}

func (m *EntryManager) RebuildIndex() error {
	if m.search == nil {
		return nil
	}

	err := m.search.ResetIndex()
	if err != nil {
		return err
	}

	entries, err := m.GetAll()
	if err != nil {
		return err
	}

	return m.search.Add(entries...)
}

func (m *EntryManager) cleanID(id string) string {
	id = path.Clean(id)
	id = strings.TrimSuffix(id, "/")
	id = strings.TrimPrefix(id, "/")
	return "/" + id
}

func (m *EntryManager) guessPath(id string) (string, error) {
	path := filepath.Join(m.source, "content", id+".md")
	if _, err := os.Stat(path); err == nil {
		return path, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}

	path = filepath.Join(m.source, "content", id, "index.md")
	if _, err := os.Stat(path); err == nil {
		return path, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}

	path = filepath.Join(m.source, "content", id, "_index.md")
	if _, err := os.Stat(path); err == nil {
		return path, nil
	} else {
		return "", err
	}
}

func (m *EntryManager) makePermalink(id string) (string, error) {
	u, err := url.Parse(m.domain)
	if err != nil {
		return "", err
	}
	u.Path = id
	return u.String(), nil
}
