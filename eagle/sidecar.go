package eagle

import (
	"os"
	"path/filepath"

	"github.com/hacdias/eagle/v2/entry"
)

type Sidecar struct {
	Targets     []string                 `json:"targets"`
	Context     map[string]interface{}   `json:"context"`
	Webmentions []map[string]interface{} `json:"webmentions"`
}

func (e *Eagle) getSidecar(entry *entry.Entry) (*Sidecar, string, error) {
	filename := filepath.Join(ContentDirectory, entry.ID, "_sidecar.json")

	var sidecar *Sidecar

	err := e.ReadJSON(filename, &sidecar)
	if os.IsNotExist(err) {
		err = nil
	}

	if sidecar == nil {
		sidecar = &Sidecar{}
	}

	if sidecar.Targets == nil {
		sidecar.Targets = []string{}
	}

	if sidecar.Webmentions == nil {
		sidecar.Webmentions = []map[string]interface{}{}
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

	newSidecar, err := t(oldSidecar)
	if err != nil {
		return err
	}

	return e.PersistJSON(filename, newSidecar, "sidecar: "+entry.ID)
}
