package eagle

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/hacdias/eagle/yaml"
)

type EntryManager struct {
	sync.RWMutex

	search  SearchIndex
	store   *Storage
	baseURL string
}

func (m *EntryManager) GetEntry(id string) (*Entry, error) {
	m.RLock()
	defer m.RUnlock()

	id = m.cleanID(id)
	filepath, err := m.guessPath(id)
	if err != nil {
		return nil, err
	}

	raw, err := m.store.ReadFile(filepath)
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
			// Default path for new files is {slug}/index.md
			path = filepath.Join(entry.ID, "index.md")
		}
		entry.Path = path
	}

	err := m.store.MkdirAll(filepath.Dir(entry.Path), 0777)
	if err != nil {
		return err
	}

	str, err := entry.String()
	if err != nil {
		return err
	}

	err = m.store.Persist(entry.Path, []byte(str), "hugo: update "+entry.ID)
	if err != nil {
		return fmt.Errorf("could not save entry: %w", err)
	}

	if m.search != nil {
		_ = m.search.Add(entry)
	}

	return nil
}

func (m *EntryManager) DeleteEntry(entry *Entry) error {
	entry.Metadata.ExpiryDate = time.Now()

	if m.search != nil {
		// We update the search index so it knows the post is expired.
		// Only remove posts that actually do not exist in disk.
		_ = m.search.Add(entry)
	}

	return m.SaveEntry(entry)
}

func (m *EntryManager) GetAll() ([]*Entry, error) {
	m.RLock()
	defer m.RUnlock()

	entries := []*Entry{}
	err := m.store.Walk(".", func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(p, ".md") {
			return nil
		}

		id := strings.TrimSuffix(p, ".md")
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

func (m *EntryManager) Search(query *SearchQuery, page int) ([]*SearchEntry, error) {
	if m.search == nil {
		return []*SearchEntry{}, nil
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

func (m *EntryManager) MakeBundle(entry *Entry) error {
	if entry.Path == "" {
		return fmt.Errorf("entry %s does not contain a path", entry.ID)
	}

	if strings.HasSuffix(entry.Path, "index.md") {
		// already a page bundle
		return nil
	}

	dir := strings.TrimSuffix(entry.Path, filepath.Ext(entry.Path))
	file := filepath.Join(dir, "index.md")

	err := m.store.MkdirAll(dir, 0777)
	if err != nil {
		return err
	}

	err = m.store.Rename(entry.Path, file)
	if err != nil {
		return err
	}

	entry.Path = file
	return nil
}

func (m *EntryManager) cleanID(id string) string {
	id = path.Clean(id)
	id = strings.TrimSuffix(id, "/")
	id = strings.TrimPrefix(id, "/")
	return "/" + id
}

func (m *EntryManager) guessPath(id string) (string, error) {
	path := id + ".md"
	if _, err := m.store.Stat(path); err == nil {
		return path, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}

	path = filepath.Join(id, "index.md")
	if _, err := m.store.Stat(path); err == nil {
		return path, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}

	path = filepath.Join(id, "_index.md")
	if _, err := m.store.Stat(path); err == nil {
		return path, nil
	} else {
		return "", err
	}
}

func (m *EntryManager) makePermalink(id string) (string, error) {
	u, err := url.Parse(m.baseURL)
	if err != nil {
		return "", err
	}
	u.Path = id
	return u.String(), nil
}
