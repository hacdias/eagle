package fs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	"go.hacdias.com/eagle/entry"
)

const (
	contentDirectory string = "content"
)

type FS struct {
	ContentFS *afero.Afero

	afero  *afero.Afero
	parser *entry.Parser
}

func NewFS(path, baseURL string) *FS {
	return &FS{
		afero: &afero.Afero{
			Fs: afero.NewBasePathFs(afero.NewOsFs(), path),
		},

		ContentFS: &afero.Afero{
			Fs: afero.NewBasePathFs(afero.NewOsFs(), filepath.Join(path, contentDirectory)),
		},
		parser: entry.NewParser(baseURL),
	}
}

func (f *FS) Stat(name string) (os.FileInfo, error) {
	return f.afero.Stat(name)
}

func (f *FS) WriteFile(filename string, data []byte, message string) error {
	// TODO: use message for Git.
	return f.afero.WriteFile(filename, data, 0644)
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

	// TODO: use message for Git.
	_ = filenames
	return nil
}

func (f *FS) WriteJSON(filename string, data interface{}, message string) error {
	json, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return f.WriteFile(filename, json, message)
}

func (f *FS) ReadFile(filename string) ([]byte, error) {
	return f.afero.ReadFile(filename)
}

func (f *FS) ReadJSON(filename string, v interface{}) error {
	data, err := f.afero.ReadFile(filename)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, v)
}

// func (f *FS) RemoveAll(path string) error {
// 	return f.afero.RemoveAll(path)
// }

func (fs *FS) GetEntry(id string) (*entry.Entry, error) {
	filename := filepath.Join(contentDirectory, id, "index.md")
	raw, err := fs.afero.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	e, err := fs.parser.Parse(id, string(raw))
	if err != nil {
		return nil, err
	}

	return e, nil
}

func (fs *FS) GetEntries(includeList bool) ([]*entry.Entry, error) {
	ee := []*entry.Entry{}
	err := fs.afero.Walk(contentDirectory, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(p, ".md") {
			return nil
		}

		id := strings.TrimPrefix(p, contentDirectory)
		id = strings.TrimSuffix(id, ".md")
		id = strings.TrimSuffix(id, "index")

		e, err := fs.GetEntry(id)
		if err != nil {
			return err
		}

		// TODO: filter
		// if !e.IsList || includeList {
		ee = append(ee, e)
		// }

		return nil
	})

	return ee, err
}

func (f *FS) SaveEntry(e *entry.Entry, message string) error {
	// TODO: sanitize taxonomies.

	filename := filepath.Join(contentDirectory, e.ID, "index.md")
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
