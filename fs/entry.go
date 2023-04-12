package fs

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hacdias/eagle/eagle"
	"github.com/hacdias/eagle/util"
	"github.com/samber/lo"
)

type EntryTransformer func(*eagle.Entry) (*eagle.Entry, error)

func (fs *FS) GetEntry(id string) (*eagle.Entry, error) {
	filename := fs.getEntryFilename(id)
	raw, err := fs.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	e, err := fs.parser.Parse(id, string(raw))
	if err != nil {
		return nil, err
	}

	return e, nil
}

func (fs *FS) GetEntries(includeList bool) (eagle.Entries, error) {
	ee := eagle.Entries{}
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

func (f *FS) RenameEntry(oldID, newID string) (*eagle.Entry, error) {
	f.entriesMu.Lock()
	defer f.entriesMu.Unlock()

	old, err := f.GetEntry(oldID)
	if err != nil {
		return nil, err
	}

	oldDir := filepath.Join(ContentDirectory, oldID)
	newDir := filepath.Join(ContentDirectory, newID)

	exists, err := f.Exists(newDir)
	if err != nil {
		return nil, err
	} else if exists {
		return nil, errors.New("target directory already exists")
	}

	err = f.MkdirAll(filepath.Dir(newDir), 0777)
	if err != nil {
		return nil, err
	}

	err = f.Rename(oldDir, newDir)
	if err != nil {
		return nil, err
	}

	updates := []string{oldDir, newDir}
	if !old.Draft && !old.Unlisted && !old.Deleted() {
		err = f.AppendRedirect(oldID, newID)
		if err != nil {
			return nil, err
		}
		updates = append(updates, RedirectsFile)
	}

	err = f.sync.Persist(updates...)
	if err != nil {
		return nil, err
	}

	new, err := f.GetEntry(newID)
	if err != nil {
		return nil, err
	}

	if f.AfterSaveHook != nil {
		f.AfterSaveHook(eagle.Entries{new}, eagle.Entries{old})
	}

	return new, nil
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
	e.Tags = cleanTaxonomy(e.Tags)
	e.Categories = cleanTaxonomy(e.Categories)

	filename := f.getEntryFilename(e.ID)
	err := f.MkdirAll(filepath.Dir(filename), 0777)
	if err != nil {
		return err
	}

	str, err := e.String()
	if err != nil {
		return err
	}

	err = f.WriteFile(filename, []byte(str))
	if err != nil {
		return fmt.Errorf("could not save entry: %w", err)
	}

	if f.AfterSaveHook != nil {
		f.AfterSaveHook(eagle.Entries{e}, nil)
	}

	return nil
}

func (f *FS) getEntryFilename(id string) string {
	path := filepath.Join(ContentDirectory, id, "index.md")
	if _, err := f.Afero.Stat(path); err == nil {
		return path
	}

	return filepath.Join(ContentDirectory, id, "_index.md")
}

func cleanTaxonomy(els []string) []string {
	for i := range els {
		els[i] = util.Slugify(els[i])
	}

	return lo.Uniq(els)
}
