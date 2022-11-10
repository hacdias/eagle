package fs

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hacdias/eagle/eagle"
	"github.com/thoas/go-funk"
)

type EntryTransformer func(*eagle.Entry) (*eagle.Entry, error)

func (fs *FS) GetEntry(id string) (*eagle.Entry, error) {
	filepath, err := fs.guessPath(id)
	if err != nil {
		return nil, err
	}

	raw, err := fs.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	e, err := fs.parser.FromRaw(id, string(raw))
	if err != nil {
		return nil, err
	}

	return e, nil
}

func (fs *FS) GetEntries(includeList bool) ([]*eagle.Entry, error) {
	ee := []*eagle.Entry{}
	err := fs.Walk(ContentDirectory, func(p string, info os.FileInfo, err error) error {
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

		e, err := fs.GetEntry(id)
		if err != nil {
			return err
		}

		if e.Listing == nil || includeList {
			ee = append(ee, e)
		}

		return nil
	})

	return ee, err
}

func (f *FS) SaveEntry(entry *eagle.Entry) error {
	f.entriesMu.Lock()
	defer f.entriesMu.Unlock()

	return f.saveEntry(entry)
}

func (f *FS) TransformEntry(id string, transformers ...EntryTransformer) (*eagle.Entry, error) {
	if len(transformers) == 0 {
		return nil, errors.New("at least one entry transformer must be provided")
	}

	f.entriesMu.Lock()
	defer f.entriesMu.Unlock()

	e, err := f.GetEntry(id)
	if err != nil {
		return nil, err
	}

	for _, t := range transformers {
		e, err = t(e)
		if err != nil {
			return nil, err
		}
	}

	err = f.saveEntry(e)
	return e, err
}

func (f *FS) saveEntry(e *eagle.Entry) error {
	e.Sections = funk.UniqString(e.Sections)

	path, err := f.guessPath(e.ID)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		// Default path for new files is content/{slug}/index.md
		path = filepath.Join(ContentDirectory, e.ID, "index.md")
	}

	err = f.MkdirAll(filepath.Dir(path), 0777)
	if err != nil {
		return err
	}

	str, err := e.String()
	if err != nil {
		return err
	}

	err = f.WriteFile(path, []byte(str), "update "+e.ID)
	if err != nil {
		return fmt.Errorf("could not save entry: %w", err)
	}

	if f.AfterSaveHook != nil {
		f.AfterSaveHook(e)
	}

	return nil
}

func (f *FS) guessPath(id string) (string, error) {
	path := filepath.Join(ContentDirectory, id, "index.md")
	_, err := f.Stat(path)
	if err == nil {
		return path, nil
	}

	return "", err
}
