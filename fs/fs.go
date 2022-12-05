package fs

import (
	"encoding/json"
	"sync"

	"github.com/hacdias/eagle/eagle"
	"github.com/spf13/afero"
)

const (
	ContentDirectory string = "content"
)

type Sync interface {
	Sync() (updated []string, err error)
	Persist(message string, filename ...string) error
}

type NopSync struct{}

func (g *NopSync) Persist(msg string, file ...string) error {
	return nil
}

func (g *NopSync) Sync() ([]string, error) {
	return []string{}, nil
}

type FS struct {
	*afero.Afero
	sync   Sync
	parser *eagle.Parser

	// Mutexes to lock the updates to entries and sidecars.
	// Only for writes and not for reads. Hope this won't
	// become a problem with traffic and simultaneous
	// reads-writes from files. Adding a mutex for all reads
	// would probably make it much slower though.
	entriesMu  sync.Mutex
	sidecarsMu sync.Mutex

	// AfterSaveHook is a hook that is executed after
	// saving an entry to the file system.
	AfterSaveHook func(updated, deleted []*eagle.Entry)
}

func NewFS(path, baseURL string, sync Sync) *FS {
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
