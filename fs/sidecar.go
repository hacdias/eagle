package fs

import (
	"os"
	"path/filepath"
	"sort"

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

	if sidecar.Targets == nil {
		sidecar.Targets = []string{}
	}

	if sidecar.Replies == nil {
		sidecar.Replies = []*eagle.Mention{}
	}

	if sidecar.Interactions == nil {
		sidecar.Interactions = []*eagle.Mention{}
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

	sort.SliceStable(newSd.Replies, func(i, j int) bool {
		return newSd.Replies[i].Published.Before(newSd.Replies[j].Published)
	})

	sort.SliceStable(newSd.Interactions, func(i, j int) bool {
		return newSd.Interactions[i].Published.Before(newSd.Interactions[j].Published)
	})

	if f.AfterSaveHook != nil {
		f.AfterSaveHook(entry)
	}

	return f.WriteJSON(filename, newSd, "sidecar: "+entry.ID)
}
