package eagle

import (
	"fmt"
	"os"
	"path/filepath"
)

type EntryData struct {
	Targets     []string      `json:"targets"`
	Webmentions []*Webmention `json:"webmentions"`
}

func (e *Eagle) getEntryDataFilename(entry *Entry) (string, error) {
	if entry.Metadata.DataID == "" {
		// NOTE: this should not be possible as everything goes through .GetEntry
		// which ensures this field is filled.
		return "", fmt.Errorf("entry does not have data id")
	}

	return filepath.Join("data", "content", entry.Metadata.DataID+".json"), nil
}

func (e *Eagle) GetEntryData(entry *Entry) (*EntryData, error) {
	e.entriesDataMu.RLock()
	defer e.entriesDataMu.RUnlock()

	filename, err := e.getEntryDataFilename(entry)
	if err != nil {
		return nil, err
	}

	var entryData *EntryData
	err = e.ReadJSON(filename, &entryData)
	if os.IsNotExist(err) {
		return &EntryData{
			Targets:     []string{},
			Webmentions: []*Webmention{},
		}, nil
	}
	return entryData, err
}

func (e *Eagle) SaveEntryData(entry *Entry, data *EntryData) error {
	e.entriesDataMu.Lock()
	defer e.entriesDataMu.Unlock()

	filename, err := e.getEntryDataFilename(entry)
	if err != nil {
		return err
	}

	return e.PersistJSON(filename, data, "entry data: update "+entry.ID)
}

func (e *Eagle) TransformEntryData(entry *Entry, t func(*EntryData) (*EntryData, error)) error {
	e.entriesDataMu.Lock()
	defer e.entriesDataMu.Unlock()

	filename, err := e.getEntryDataFilename(entry)
	if err != nil {
		return err
	}

	var oldData *EntryData
	err = e.ReadJSON(filename, &oldData)
	if err != nil && !os.IsNotExist(err) {
		return err
	} else if os.IsNotExist(err) {
		oldData = &EntryData{
			Targets:     []string{},
			Webmentions: []*Webmention{},
		}
	}

	newData, err := t(oldData)
	if err != nil {
		return err
	}

	return e.PersistJSON(filename, newData, "entry data: update "+entry.ID)
}
