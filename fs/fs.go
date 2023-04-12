package fs

import (
	"encoding/json"
	"sync"

	"github.com/hacdias/eagle/eagle"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v2"
)

const (
	ContentDirectory string = "content"
)

type Sync interface {
	Sync() (updated []string, err error)
	Persist(filename ...string) error
}

type NopSync struct{}

func (g *NopSync) Persist(file ...string) error {
	return nil
}

func (g *NopSync) Sync() ([]string, error) {
	return []string{}, nil
}

type FS struct {
	*afero.Afero
	path string

	sync   Sync
	parser *eagle.Parser

	// Mutexes to lock the updates to entries.
	// Only for writes and not for reads. Hope this won't
	// become a problem with traffic and simultaneous
	// reads-writes from files. Adding a mutex for all reads
	// would probably make it much slower though.
	entriesMu sync.Mutex

	// AfterSaveHook is a hook that is executed after
	// saving an entry to the file system.
	AfterSaveHook func(updated, deleted eagle.Entries)
}

func NewFS(path, baseURL string, sync Sync) *FS {
	afero := &afero.Afero{
		Fs: afero.NewBasePathFs(afero.NewOsFs(), path),
	}

	return &FS{
		Afero:  afero,
		path:   path,
		sync:   sync,
		parser: eagle.NewParser(baseURL),
	}
}

func (f *FS) Sync() ([]string, error) {
	return f.sync.Sync()
}

func (f *FS) WriteFile(filename string, data []byte) error {
	err := f.Afero.WriteFile(filename, data, 0644)
	if err != nil {
		return err
	}

	err = f.sync.Persist(filename)
	if err != nil {
		return err
	}

	return nil
}

func (f *FS) RemoveFile(filename string) error {
	if _, err := f.Stat(filename); err == nil {
		err := f.Afero.Remove(filename)
		if err != nil {
			return err
		}

		err = f.sync.Persist(filename)
		if err != nil {
			return err
		}

	}

	return nil
}

func (f *FS) WriteJSON(filename string, data interface{}) error {
	json, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return f.WriteFile(filename, json)
}

func (f *FS) ReadJSON(filename string, v interface{}) error {
	data, err := f.ReadFile(filename)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, v)
}

func (f *FS) ReadYAML(filename string, v interface{}) error {
	data, err := f.ReadFile(filename)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(data, v)
}
