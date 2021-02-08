package services

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

	"github.com/hacdias/eagle/yaml"
)

type EntryManager struct {
	sync.Mutex
	domain string
	source string
}

func (m *EntryManager) GetEntry(id string) (*Entry, error) {
	id = m.cleanID(id)
	filepath, err := m.guessPath(id)
	if err != nil {
		return nil, err
	}

	bytes, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	splits := strings.SplitN(string(bytes), "\n---", 2)
	if len(splits) != 2 {
		return nil, errors.New("could not parse file: splits !== 2")
	}

	permalink, err := m.makePermalink(id)
	if err != nil {
		return nil, err
	}

	entry := &Entry{
		Path:      filepath,
		ID:        id,
		Permalink: permalink,
		Content:   strings.TrimSpace(splits[1]),
		Metadata:  EntryMetadata{},
	}

	err = yaml.Unmarshal([]byte(splits[0]), &entry.Metadata)
	if err != nil {
		return nil, err
	}

	return entry, nil
}

func (m *EntryManager) SaveEntry(entry *Entry) error {
	entry.ID = m.cleanID(entry.ID)
	if entry.Path == "" {
		path, err := m.guessPath(entry.ID)
		if err != nil {
			if os.IsNotExist(err) {
				// Default path for new files is {slug}.md
				path = filepath.Join(m.source, "content", entry.ID+".md")
			} else {
				return err
			}
		}
		entry.Path = path
	}

	err := os.MkdirAll(filepath.Dir(entry.Path), 0777)
	if err != nil {
		return err
	}

	val, err := yaml.Marshal(&entry.Metadata)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(entry.Path, []byte(fmt.Sprintf("---\n%s---\n\n%s\n", string(val), entry.Content)), 0644)
	if err != nil {
		return fmt.Errorf("could not save entry: %s", err)
	}

	return nil
}

func (m *EntryManager) GetAll() ([]*Entry, error) {
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
