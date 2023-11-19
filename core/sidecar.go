package core

import (
	"os"
	"path/filepath"
	"sort"

	"go.hacdias.com/eagle/xray"
	"go.hacdias.com/indielib/microformats"
)

const (
	sidecarFilename = "sidecar.json"
)

type Mention struct {
	xray.Post
	Source string `json:"source,omitempty"`

	ID      string `json:"-"`
	EntryID string `json:"-"`
}

func (m *Mention) IsInteraction() bool {
	return m.Type == microformats.TypeLike ||
		m.Type == microformats.TypeRepost ||
		m.Type == microformats.TypeBookmark ||
		m.Type == microformats.TypeRsvp
}

type Sidecar struct {
	Context      *xray.Post `json:"context,omitempty"`
	Replies      []*Mention `json:"replies,omitempty"`
	Interactions []*Mention `json:"interactions,omitempty"`
}

func (s *Sidecar) Empty() bool {
	return s.Context == nil &&
		len(s.Replies) == 0 &&
		len(s.Interactions) == 0
}

func (f *Core) getSidecar(entry *Entry) (*Sidecar, string, error) {
	filename := filepath.Join(ContentDirectory, entry.ID, sidecarFilename)

	var sidecar *Sidecar

	err := f.ReadJSON(filename, &sidecar)
	if os.IsNotExist(err) {
		err = nil
	} else if err != nil {
		return nil, "", err
	}

	if sidecar == nil {
		sidecar = &Sidecar{}
	}

	if sidecar.Replies == nil {
		sidecar.Replies = []*Mention{}
	}

	if sidecar.Interactions == nil {
		sidecar.Interactions = []*Mention{}
	}

	return sidecar, filename, err
}

// func (f *Core) GetSidecar(entry *Entry) (*Sidecar, error) {
// 	sidecar, _, err := f.getSidecar(entry)
// 	return sidecar, err
// }

func (f *Core) UpdateSidecar(entry *Entry, t func(*Sidecar) (*Sidecar, error)) error {
	oldSidecar, filename, err := f.getSidecar(entry)
	if err != nil {
		return err
	}

	newSidecar, err := t(oldSidecar)
	if err != nil {
		return err
	}

	sort.SliceStable(newSidecar.Replies, func(i, j int) bool {
		return newSidecar.Replies[i].Date.After(newSidecar.Replies[j].Date)
	})

	sort.SliceStable(newSidecar.Interactions, func(i, j int) bool {
		return newSidecar.Interactions[i].Date.After(newSidecar.Interactions[j].Date)
	})

	if newSidecar.Empty() {
		err = f.sourceFS.Remove(filename)
		if os.IsNotExist(err) {
			err = nil
		}
		return err
	}

	return f.WriteJSON(filename, newSidecar, "sidecar: update for "+entry.ID)
}
