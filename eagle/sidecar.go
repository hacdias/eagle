package eagle

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/hacdias/eagle/v4/entry"
	"github.com/hacdias/eagle/v4/pkg/xray"
)

const (
	SidecarFilename = ".sidecar.json"
)

type Sidecar struct {
	Targets      []string     `json:"targets,omitempty"`
	Context      *xray.Post   `json:"context,omitempty"`
	Replies      []*xray.Post `json:"replies,omitempty"`
	Interactions []*xray.Post `json:"interactions,omitempty"`
}

func (s *Sidecar) MentionsCount() int {
	return len(s.Replies) + len(s.Interactions)
}

func (e *Eagle) getSidecar(entry *entry.Entry) (*Sidecar, string, error) {
	filename := filepath.Join(ContentDirectory, entry.ID, SidecarFilename)

	var sidecar *Sidecar

	err := e.FS.ReadJSON(filename, &sidecar)
	if os.IsNotExist(err) {
		err = nil
	} else if err != nil {
		return nil, "", err
	}

	if sidecar == nil {
		sidecar = &Sidecar{}
	}

	if sidecar.Targets == nil {
		sidecar.Targets = []string{}
	}

	if sidecar.Replies == nil {
		sidecar.Replies = []*xray.Post{}
	}

	if sidecar.Interactions == nil {
		sidecar.Interactions = []*xray.Post{}
	}

	return sidecar, filename, err
}

func (e *Eagle) GetSidecar(entry *entry.Entry) (*Sidecar, error) {
	sidecar, _, err := e.getSidecar(entry)
	return sidecar, err
}

func (e *Eagle) UpdateSidecar(entry *entry.Entry, t func(*Sidecar) (*Sidecar, error)) error {
	e.sidecarsMu.Lock()
	defer e.sidecarsMu.Unlock()

	oldSidecar, filename, err := e.getSidecar(entry)
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

	if e.Cache != nil {
		e.Cache.Delete(entry)
	}
	return e.FS.WriteJSON(filename, newSd, "sidecar: "+entry.ID)
}
