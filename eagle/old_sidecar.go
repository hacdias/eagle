package eagle

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/hacdias/eagle/v4/entry"
)

type OldSidecar struct {
	Targets     []string                 `json:"targets"`
	Context     map[string]interface{}   `json:"context"`
	Webmentions []map[string]interface{} `json:"webmentions"`
}

func (e *Eagle) getOldSidecar(entry *entry.Entry) (*OldSidecar, string, error) {
	filename := filepath.Join(ContentDirectory, entry.ID, "_sidecar.json")

	var sidecar *OldSidecar

	err := e.fs.ReadJSON(filename, &sidecar)
	if os.IsNotExist(err) {
		err = nil
	}

	if sidecar == nil {
		sidecar = &OldSidecar{}
	}

	if sidecar.Targets == nil {
		sidecar.Targets = []string{}
	}

	if sidecar.Webmentions == nil {
		sidecar.Webmentions = []map[string]interface{}{}
	}

	sort.Slice(sidecar.Webmentions, func(i, j int) bool {
		a, ok := sidecar.Webmentions[i]["published"].(string)
		if !ok {
			return false
		}

		b, ok := sidecar.Webmentions[j]["published"].(string)
		if !ok {
			return false
		}

		return a > b
	})

	return sidecar, filename, err
}

func (e *Eagle) GetOldSidecar(entry *entry.Entry) (*OldSidecar, error) {
	sidecar, _, err := e.getOldSidecar(entry)
	return sidecar, err
}

func (e *Eagle) UpdateOldSidecar(entry *entry.Entry, t func(*OldSidecar) (*OldSidecar, error)) error {
	e.sidecarsMu.Lock()
	defer e.sidecarsMu.Unlock()

	oldSidecar, filename, err := e.getOldSidecar(entry)
	if err != nil {
		return err
	}

	newSidecar, err := t(oldSidecar)
	if err != nil {
		return err
	}

	return e.fs.WriteJSON(filename, newSidecar, "sidecar: "+entry.ID)
}
