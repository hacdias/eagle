package fs

import (
	"encoding/json"

	"github.com/spf13/afero"
)

type FSSync interface {
	Sync() (updated []string, err error)
	Persist(message, filename string) error
}

type FS struct {
	*afero.Afero
	sync FSSync
}

func NewFS(path string, sync FSSync) *FS {
	afero := &afero.Afero{
		Fs: afero.NewBasePathFs(afero.NewOsFs(), path),
	}

	return &FS{
		Afero: afero,
		sync:  sync,
	}
}

func (f *FS) Sync() ([]string, error) {
	return f.sync.Sync()
}

func (f *FS) WriteFile(filename string, data []byte, message string) error {
	err := f.Afero.WriteFile(filename, data, 0644)
	if err != nil {
		return err
	}

	err = f.sync.Persist(message, filename)
	if err != nil {
		return err
	}

	return nil
}

func (f *FS) WriteJSON(filename string, data interface{}, msg string) error {
	json, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return f.WriteFile(filename, json, msg)
}

func (f *FS) ReadJSON(filename string, v interface{}) error {
	data, err := f.ReadFile(filename)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, v)
}
