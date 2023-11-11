package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/samber/lo"
	"github.com/spf13/afero"
)

const (
	ContentDirectory string = "content"
	DataDirectory    string = "data"

	RedirectsFile = "redirects"
	GoneFile      = "gone"
)

type FS struct {
	afero *afero.Afero
	path  string

	sync   Sync
	parser *Parser

	// Mutexes to lock the updates to entries.
	// Only for writes and not for reads. Hope this won't
	// become a problem with traffic and simultaneous
	// reads-writes from files. Adding a mutex for all reads
	// would probably make it much slower though.
	entriesMu sync.Mutex
}

func NewFS(path, baseURL string, sync Sync) *FS {
	afero := &afero.Afero{
		Fs: afero.NewBasePathFs(afero.NewOsFs(), path),
	}

	return &FS{
		afero:  afero,
		path:   path,
		sync:   sync,
		parser: NewParser(baseURL),
	}
}

func (f *FS) Sync() ([]string, error) {
	return f.sync.Sync()
}

func (f *FS) WriteFile(filename string, data []byte, message string) error {
	err := f.afero.WriteFile(filename, data, 0644)
	if err != nil {
		return err
	}

	return f.sync.Persist(message, filename)
}

func (f *FS) WriteFiles(filesAndData map[string][]byte, message string) error {
	var filenames []string

	for filename, data := range filesAndData {
		err := f.afero.WriteFile(filename, data, 0644)
		if err != nil {
			return err
		}
		filenames = append(filenames, filename)
	}

	return f.sync.Persist(message, filenames...)
}

func (f *FS) ReadFile(filename string) ([]byte, error) {
	return f.afero.ReadFile(filename)
}

func (f *FS) WriteJSON(filename string, data interface{}, message string) error {
	json, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return f.WriteFile(filename, json, message)
}

func (f *FS) ReadJSON(filename string, v interface{}) error {
	data, err := f.afero.ReadFile(filename)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, v)
}

func (f *FS) RemoveAll(path string) error {
	return f.afero.RemoveAll(path)
}

type EntryTransformer func(*Entry) (*Entry, error)

func (fs *FS) GetEntry(id string) (*Entry, error) {
	filename := fs.guessFilename(id)
	raw, err := fs.afero.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	e, err := fs.parser.Parse(id, string(raw))
	if err != nil {
		return nil, err
	}

	e.IsList = strings.Contains(filename, "_index.md")
	return e, nil
}

func (fs *FS) GetEntries(includeList bool) (Entries, error) {
	ee := Entries{}
	err := fs.afero.Walk(ContentDirectory, func(p string, info os.FileInfo, err error) error {
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

		if !e.IsList || includeList {
			ee = append(ee, e)
		}

		return nil
	})

	return ee, err
}

func (f *FS) SaveEntry(entry *Entry) error {
	f.entriesMu.Lock()
	defer f.entriesMu.Unlock()

	return f.saveEntry(entry, "")
}

func (f *FS) RenameEntry(oldID, newID string) (*Entry, error) {
	f.entriesMu.Lock()
	defer f.entriesMu.Unlock()

	old, err := f.GetEntry(oldID)
	if err != nil {
		return nil, err
	}

	oldDir := filepath.Join(ContentDirectory, oldID)
	newDir := filepath.Join(ContentDirectory, newID)

	exists, err := f.afero.Exists(newDir)
	if err != nil {
		return nil, err
	} else if exists {
		return nil, errors.New("target directory already exists")
	}

	err = f.afero.MkdirAll(filepath.Dir(newDir), 0777)
	if err != nil {
		return nil, err
	}

	err = f.afero.Rename(oldDir, newDir)
	if err != nil {
		return nil, err
	}

	updates := []string{oldDir, newDir}
	if !old.Draft && !old.NoIndex && !old.Deleted() {
		err = f.AppendRedirect(oldID, newID)
		if err != nil {
			return nil, err
		}
		updates = append(updates, RedirectsFile)
	}

	err = f.sync.Persist(fmt.Sprintf("entry: rename from %s to %s", oldID, newID), updates...)
	if err != nil {
		return nil, err
	}

	new, err := f.GetEntry(newID)
	if err != nil {
		return nil, err
	}

	return new, nil
}

func (f *FS) TransformEntry(id, message string, transformers ...EntryTransformer) (*Entry, error) {
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

	err = f.saveEntry(e, message)
	return e, err
}

func (f *FS) saveEntry(e *Entry, message string) error {
	e.Tags = cleanTaxonomy(e.Tags)

	filename := f.guessFilename(e.ID)
	err := f.afero.MkdirAll(filepath.Dir(filename), 0777)
	if err != nil {
		return err
	}

	str, err := e.String()
	if err != nil {
		return err
	}

	if message == "" {
		message = "entry: update " + e.ID
	}

	err = f.WriteFile(filename, []byte(str), message)
	if err != nil {
		return fmt.Errorf("could not save entry: %w", err)
	}

	return nil
}

func (f *FS) guessFilename(id string) string {
	path := filepath.Join(ContentDirectory, id, "_index.md")
	if _, err := f.afero.Stat(path); err == nil {
		return path
	}

	return filepath.Join(ContentDirectory, id, "index.md")
}

func cleanTaxonomy(els []string) []string {
	for i := range els {
		els[i] = slugify(els[i])
	}

	return lo.Uniq(els)
}

func (fs *FS) AppendRedirect(old, new string) error {
	f, err := fs.afero.OpenFile(RedirectsFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(fmt.Sprintf("%s %s\n", old, new))
	return err
}

func (fs *FS) LoadRedirects(ignoreMalformed bool) (map[string]string, error) {
	redirects := map[string]string{}

	data, err := fs.afero.ReadFile(RedirectsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return redirects, nil
		}
		return nil, err
	}

	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		parts := strings.Split(line, " ")
		if len(parts) == 2 {
			redirects[parts[0]] = parts[1]
		} else if !ignoreMalformed {
			return nil, fmt.Errorf("found invalid redirect entry: %s", line)
		}
	}

	return redirects, nil
}

func (fs *FS) LoadGone() (map[string]bool, error) {
	gone := map[string]bool{}

	data, err := fs.afero.ReadFile(GoneFile)
	if err != nil {
		if os.IsNotExist(err) {
			return gone, nil
		}
		return nil, err
	}

	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		gone[line] = true
	}

	return gone, nil
}
