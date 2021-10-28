package eagle

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/hacdias/eagle/yaml"
)

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

	entry.Path = filepath
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

func (e *Eagle) SaveEntry(entry *Entry) error {
	e.entriesMu.Lock()
	defer e.entriesMu.Unlock()

	entry.ID = e.cleanID(entry.ID)
	if entry.Path == "" {
		path, err := e.guessPath(entry.ID)
		if err != nil {
			if !os.IsNotExist(err) {
				return err

			}
			// Default path for new files is content/{slug}/index.md
			path = filepath.Join("content", entry.ID, "index.md")
		}
		entry.Path = path
	}

	err := e.srcFs.MkdirAll(filepath.Dir(entry.Path), 0777)
	if err != nil {
		return err
	}

	str, err := entry.String()
	if err != nil {
		return err
	}

	err = e.Persist(entry.Path, []byte(str), "hugo: update "+entry.ID)
	if err != nil {
		return fmt.Errorf("could not save entry: %w", err)
	}

	if e.search != nil {
		_ = e.search.Add(entry)
	}

	return nil
}

func (e *Eagle) DeleteEntry(entry *Entry) error {
	entry.Metadata.ExpiryDate = time.Now()

	if e.search != nil {
		// We update the search index so it knows the post is expired.
		// Only remove posts that actually do not exist in disk.
		_ = e.search.Add(entry)
	}

	return e.SaveEntry(entry)
}

func (e *Eagle) GetAllEntries() ([]*Entry, error) {
	e.entriesMu.RLock()
	defer e.entriesMu.RUnlock()

	entries := []*Entry{}
	err := e.srcFs.Walk("content/", func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(p, ".md") {
			return nil
		}

		id := strings.TrimPrefix(p, "content/")
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

func (e *Eagle) MakeEntryBundle(entry *Entry) error {
	if entry.Path == "" {
		return fmt.Errorf("entry %s does not contain a path", entry.ID)
	}

	if strings.HasSuffix(entry.Path, "index.md") {
		// already a page bundle
		return nil
	}

	dir := strings.TrimSuffix(entry.Path, filepath.Ext(entry.Path))
	file := filepath.Join(dir, "index.md")

	err := e.srcFs.MkdirAll(dir, 0777)
	if err != nil {
		return err
	}

	err = e.srcFs.Rename(entry.Path, file)
	if err != nil {
		return err
	}

	entry.Path = file
	return nil
}

func (e *Eagle) cleanID(id string) string {
	id = path.Clean(id)
	id = strings.TrimSuffix(id, "/")
	id = strings.TrimPrefix(id, "/")
	return "/" + id
}

func (e *Eagle) guessPath(id string) (string, error) {
	path := filepath.Join("content", id+".md")
	if _, err := e.srcFs.Stat(path); err == nil {
		return path, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}

	path = filepath.Join("content", id, "index.md")
	if _, err := e.srcFs.Stat(path); err == nil {
		return path, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}

	path = filepath.Join("content", id, "_index.md")
	if _, err := e.srcFs.Stat(path); err == nil {
		return path, nil
	} else {
		return "", err
	}
}

func (e *Eagle) makePermalink(id string) (string, error) {
	u, err := url.Parse(e.Config.BaseURL)
	if err != nil {
		return "", err
	}
	u.Path = id
	return u.String(), nil
}
