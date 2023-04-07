package fs

import (
	"os"
	"path/filepath"

	"github.com/hacdias/eagle/eagle"
)

const (
	SidecarFilename = ".sidecar.json"
)

func (f *FS) getSidecar(entry *eagle.Entry) (*eagle.Sidecar, string, error) {
	filename := filepath.Join(ContentDirectory, entry.ID, SidecarFilename)

	var sidecar *eagle.Sidecar

	err := f.ReadJSON(filename, &sidecar)
	if os.IsNotExist(err) {
		err = nil
	} else if err != nil {
		return nil, "", err
	}

	if sidecar == nil {
		sidecar = &eagle.Sidecar{}
	}

	return sidecar, filename, err
}

func (f *FS) GetSidecar(entry *eagle.Entry) (*eagle.Sidecar, error) {
	sidecar, _, err := f.getSidecar(entry)
	return sidecar, err
}

func (f *FS) UpdateSidecar(entry *eagle.Entry, t func(*eagle.Sidecar) (*eagle.Sidecar, error)) error {
	f.sidecarsMu.Lock()
	defer f.sidecarsMu.Unlock()

	oldSidecar, filename, err := f.getSidecar(entry)
	if err != nil {
		return err
	}

	newSd, err := t(oldSidecar)
	if err != nil {
		return err
	}

	if f.AfterSaveHook != nil {
		f.AfterSaveHook(eagle.Entries{entry}, nil)
	}

	if newSd.Empty() {
		return f.RemoveFile(filename)
	}

	return f.WriteJSON(filename, newSd)
}
