package eagle

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hacdias/eagle/v4/entry"
	"github.com/thoas/go-funk"
)

type EntryTransformer func(*entry.Entry) (*entry.Entry, error)

func (e *Eagle) GetEntry(id string) (*entry.Entry, error) {
	filepath, err := e.guessPath(id)
	if err != nil {
		return nil, err
	}

	raw, err := e.FS.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	entry, err := e.Parser.FromRaw(id, string(raw))
	if err != nil {
		return nil, err
	}

	return entry, nil
}

func (e *Eagle) GetEntries(includeList bool) ([]*entry.Entry, error) {
	entries := []*entry.Entry{}
	err := e.FS.Walk(ContentDirectory, func(p string, info os.FileInfo, err error) error {
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

func (e *Eagle) SaveEntry(entry *entry.Entry) error {
	e.entriesMu.Lock()
	defer e.entriesMu.Unlock()

	return e.saveEntry(entry)
}

func (e *Eagle) TransformEntry(id string, transformers ...EntryTransformer) (*entry.Entry, error) {
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

func EntryTemplates(ee *entry.Entry) []string {
	tpls := []string{}
	if ee.Template != "" {
		tpls = append(tpls, ee.Template)
	}
	tpls = append(tpls, TemplateSingle)
	return tpls
}

func (e *Eagle) saveEntry(entry *entry.Entry) error {
	entry.Sections = funk.UniqString(entry.Sections)

	path, err := e.guessPath(entry.ID)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		// Default path for new files is content/{slug}/index.md
		path = filepath.Join(ContentDirectory, entry.ID, "index.md")
	}

	err = e.FS.MkdirAll(filepath.Dir(path), 0777)
	if err != nil {
		return err
	}

	str, err := entry.String()
	if err != nil {
		return err
	}

	err = e.FS.WriteFile(path, []byte(str), "update "+entry.ID)
	if err != nil {
		return fmt.Errorf("could not save entry: %w", err)
	}

	_ = e.DB.Add(entry)
	return nil
}

func (e *Eagle) guessPath(id string) (string, error) {
	path := filepath.Join(ContentDirectory, id, "index.md")
	_, err := e.FS.Stat(path)
	if err == nil {
		return path, nil
	}

	return "", err
}
