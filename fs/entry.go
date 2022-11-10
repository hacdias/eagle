package fs

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hacdias/eagle/v4/eagle"
	"github.com/thoas/go-funk"
)

type EntryTransformer func(*eagle.Entry) (*eagle.Entry, error)

func (e *FS) GetEntry(id string) (*eagle.Entry, error) {
	filepath, err := e.guessPath(id)
	if err != nil {
		return nil, err
	}

	raw, err := e.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	entry, err := e.parser.FromRaw(id, string(raw))
	if err != nil {
		return nil, err
	}

	return entry, nil
}

func (e *FS) GetEntries(includeList bool) ([]*eagle.Entry, error) {
	entries := []*eagle.Entry{}
	err := e.Walk(ContentDirectory, func(p string, info os.FileInfo, err error) error {
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

		if entry.Listing == nil || includeList {
			entries = append(entries, entry)
		}

		return nil
	})

	return entries, err
}

func (e *FS) SaveEntry(entry *eagle.Entry) error {
	e.entriesMu.Lock()
	defer e.entriesMu.Unlock()

	return e.saveEntry(entry)
}

func (e *FS) TransformEntry(id string, transformers ...EntryTransformer) (*eagle.Entry, error) {
	if len(transformers) == 0 {
		return nil, errors.New("at least one entry transformer must be provided")
	}

	e.entriesMu.Lock()
	defer e.entriesMu.Unlock()

	ee, err := e.GetEntry(id)
	if err != nil {
		return nil, err
	}

	for _, t := range transformers {
		ee, err = t(ee)
		if err != nil {
			return nil, err
		}
	}

	err = e.saveEntry(ee)
	return ee, err
}

func (e *FS) saveEntry(entry *eagle.Entry) error {
	entry.Sections = funk.UniqString(entry.Sections)

	path, err := e.guessPath(entry.ID)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		// Default path for new files is content/{slug}/index.md
		path = filepath.Join(ContentDirectory, entry.ID, "index.md")
	}

	err = e.MkdirAll(filepath.Dir(path), 0777)
	if err != nil {
		return err
	}

	str, err := entry.String()
	if err != nil {
		return err
	}

	err = e.WriteFile(path, []byte(str), "update "+entry.ID)
	if err != nil {
		return fmt.Errorf("could not save entry: %w", err)
	}

	// wip: thingy
	if e.afterSaveHook != nil {
		e.afterSaveHook(entry)

		// _ = e.DB.Add(entry)
		// if e.Cache != nil {
		// 	e.Cache.Delete(entry)
		// }
	}

	return nil
}

func (e *FS) guessPath(id string) (string, error) {
	path := filepath.Join(ContentDirectory, id, "index.md")
	_, err := e.Stat(path)
	if err == nil {
		return path, nil
	}

	return "", err
}
