package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

	// TODO: make this configurable.
	e.IsList = strings.HasPrefix(id, "/categories/") || strings.HasPrefix(id, "/tags/")
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

		// Ignore special entry.
		// TODO: ideally this wouldn't be needed in the future.
		if id == "/_eagle/" {
			return nil
		}

		e, err := fs.GetEntry(id)
		if err != nil {
			return err
		}

		if v, ok := e.Other["_build"]; ok {
			if m, ok := v.(map[string]any); ok {
				if m["render"] == "never" {
					return nil
				}
			}
		}

		if !e.IsList || includeList {
			ee = append(ee, e)
		}

		return nil
	})

	return ee, err
}

func (f *FS) SaveEntry(e *Entry) error {
	filename := f.guessFilename(e.ID)
	err := f.afero.MkdirAll(filepath.Dir(filename), 0777)
	if err != nil {
		return err
	}

	str, err := e.String()
	if err != nil {
		return err
	}

	err = f.WriteFile(filename, []byte(str), "entry: update "+e.ID)
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
