package eagle

import (
	"errors"
	"fmt"
	urlpkg "net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/hacdias/eagle/v2/pkg/yaml"
	"github.com/karlseguin/typed"
)

type Entry struct {
	Frontmatter
	ID        string
	Permalink string
	Content   string
}

type Frontmatter struct {
	Title          string    `yaml:"title,omitempty"`
	Description    string    `yaml:"description,omitempty"`
	Draft          bool      `yaml:"draft,omitempty"`
	Deleted        bool      `yaml:"deleted,omitempty"`
	Private        bool      `yaml:"private,omitempty"`
	NoInteractions bool      `yaml:"noInteractions,omitempty"`
	Emoji          string    `yaml:"emoji,omitempty"`
	Published      time.Time `yaml:"published,omitempty"`
	Updated        time.Time `yaml:"updated,omitempty"`
	Section        string    `yaml:"section,omitempty"`

	// JF2 encoded properties.
	Properties map[string]interface{} `yaml:"properties,omitempty"`
}

func (e *Entry) Tags() []string {
	m := typed.New(e.Properties)

	if v, ok := m.StringsIf("category"); ok {
		return v
	}

	if v, ok := m.StringIf("category"); ok {
		return []string{v}
	}

	return []string{}
}

func (e *Entry) String() (string, error) {
	val, err := yaml.Marshal(&e.Frontmatter)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("---\n%s---\n\n%s\n", string(val), e.Content), nil
}

// type Picture struct {
// 	Title string `yaml:"title,omitempty"`
// 	Slug  string `yaml:"slug,omitempty"`
// 	Hide  bool   `yaml:"hide,omitempty"`
// }

func (e *Eagle) GetEntry(id string) (*Entry, error) {
	e.entriesMu.RLock()
	defer e.entriesMu.RUnlock()

	id = e.cleanID(id)
	filepath, err := e.guessPath(id)
	if err != nil {
		return nil, err
	}

	raw, err := e.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	entry, err := e.ParseEntry(id, string(raw))
	if err != nil {
		return nil, err
	}

	return entry, nil
}

func (e *Eagle) ParseEntry(id, raw string) (*Entry, error) {
	id = e.cleanID(id)
	splits := strings.SplitN(raw, "\n---", 2)
	if len(splits) != 2 {
		return nil, errors.New("could not parse file: splits !== 2")
	}

	permalink, err := e.makePermalink(id)
	if err != nil {
		return nil, err
	}

	entry := &Entry{
		ID:          id,
		Permalink:   permalink,
		Content:     strings.TrimSpace(splits[1]),
		Frontmatter: Frontmatter{},
	}

	err = yaml.Unmarshal([]byte(splits[0]), &entry.Frontmatter)
	if err != nil {
		return nil, err
	}

	return entry, nil
}

func (e *Eagle) SaveEntry(entry *Entry) error {
	e.entriesMu.Lock()
	defer e.entriesMu.Unlock()

	entry.ID = e.cleanID(entry.ID)
	path, err := e.guessPath(entry.ID)
	if err != nil {
		if !os.IsNotExist(err) {
			return err

		}
		// Default path for new files is content/{slug}/index.md
		path = filepath.Join(ContentDirectory, entry.ID, "index.md")
	}

	err = e.SrcFs.MkdirAll(filepath.Dir(path), 0777)
	if err != nil {
		return err
	}

	str, err := entry.String()
	if err != nil {
		return err
	}

	err = e.Persist(path, []byte(str), "update "+entry.ID)
	if err != nil {
		return fmt.Errorf("could not save entry: %w", err)
	}

	_ = e.IndexAdd(entry)

	return nil
}

func (e *Eagle) TransformEntry(id string, t func(*Entry) (*Entry, error)) (*Entry, error) {
	oldEntry, err := e.GetEntry(id)
	if err != nil {
		return nil, err
	}

	// TODO: make this open the file for writing and avoid using locks.

	newEntry, err := t(oldEntry)
	if err != nil {
		return nil, err
	}

	err = e.SaveEntry(newEntry)
	return newEntry, err
}

func (e *Eagle) GetAllEntries() ([]*Entry, error) {
	entries := []*Entry{}
	err := e.SrcFs.Walk(ContentDirectory, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(p, ".md") {
			return nil
		}

		id := strings.TrimPrefix(p, ContentDirectory)
		id = strings.TrimSuffix(id, ".md")
		id = strings.TrimSuffix(id, "_index")
		id = strings.TrimSuffix(id, "index")

		entry, err := e.GetEntry(id)
		if err != nil {
			return err
		}

		entries = append(entries, entry)
		return nil
	})

	return entries, err
}

// TODO: put microformats conversions here too, cleanup function names.

func (e *Eagle) cleanID(id string) string {
	id = path.Clean(id)
	id = strings.TrimSuffix(id, "/")
	id = strings.TrimPrefix(id, "/")
	return "/" + id
}

func (e *Eagle) guessPath(id string) (string, error) {
	path := filepath.Join(ContentDirectory, id, "index.md")
	if _, err := e.SrcFs.Stat(path); err == nil {
		return path, nil
	} else {
		return "", err
	}

	// path = filepath.Join(ContentDirectory, id, "_index.md")
	// if _, err := e.SrcFs.Stat(path); err == nil {
	// 	return path, nil
	// } else {
	// 	return "", err
	// }
}

func (e *Eagle) makePermalink(id string) (string, error) {
	url, err := urlpkg.Parse(e.Config.BaseURL) // Shouldn't this error be non-existent since we verify the BaseURL when parsing the conf?
	if err != nil {
		return "", err
	}
	url.Path = id
	return url.String(), nil
}
