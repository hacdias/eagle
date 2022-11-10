package fs

import (
	"encoding/json"
	"sync"

	"github.com/hacdias/eagle/v4/eagle"
	"github.com/spf13/afero"
)

const (
	AssetsDirectory  string = "assets"
	ContentDirectory string = "content"
)

type FSSync interface {
	Sync() (updated []string, err error)
	Persist(message, filename string) error
}

type FS struct {
	*afero.Afero

	sync   FSSync
	parser *eagle.Parser

	// WIP
	afterSaveHook func(*eagle.Entry)

	// Mutexes to lock the updates to entries and sidecars.
	// Only for writes and not for reads. Hope this won't
	// become a problem with traffic and simultaneous
	// reads-writes from files. Adding a mutex for all reads
	// would probably make it much slower though.
	entriesMu  sync.Mutex
	sidecarsMu sync.Mutex
}

func NewFS(path, baseURL string, sync FSSync) *FS {
	afero := &afero.Afero{
		Fs: afero.NewBasePathFs(afero.NewOsFs(), path),
	}

	return &FS{
		Afero:  afero,
		sync:   sync,
		parser: eagle.NewParser(baseURL),
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
