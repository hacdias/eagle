package eagle

import (
	"os"
	"path/filepath"
)

type EntryData struct {
	Targets     []string `json:"targets"`
	Webmentions []string `json:"webmentions"`
}

func (e *Eagle) getEntryDataFilename(entry *Entry) string {
	return filepath.Join(ContentDirectory, entry.ID, "_interactions.json")
}

func (e *Eagle) GetEntryData(entry *Entry) (*EntryData, error) {
	e.entriesDataMu.RLock()
	defer e.entriesDataMu.RUnlock()

	filename := e.getEntryDataFilename(entry)

	var entryData *EntryData
	err := e.ReadJSON(filename, &entryData)
	if os.IsNotExist(err) {
		return &EntryData{
			Targets:     []string{},
			Webmentions: []string{},
		}, nil
	}
	return entryData, err
}

func (e *Eagle) safeGetEntryData(entry *Entry) *EntryData {
	ed, _ := e.GetEntryData(entry)
	if ed == nil {
		ed = &EntryData{}
	}
	if ed.Targets == nil {
		ed.Targets = []string{}
	}
	if ed.Webmentions == nil {
		ed.Webmentions = []string{}
	}
	return ed
}

func (e *Eagle) SaveEntryData(entry *Entry, data *EntryData) error {
	e.entriesDataMu.Lock()
	defer e.entriesDataMu.Unlock()

	filename := e.getEntryDataFilename(entry)

	return e.PersistJSON(filename, data, "entry data: update "+entry.ID)
}

func (e *Eagle) TransformEntryData(entry *Entry, t func(*EntryData) (*EntryData, error)) error {
	e.entriesDataMu.Lock()
	defer e.entriesDataMu.Unlock()

	filename := e.getEntryDataFilename(entry)

	var oldData *EntryData
	err := e.ReadJSON(filename, &oldData)
	if err != nil && !os.IsNotExist(err) {
		return err
	} else if os.IsNotExist(err) {
		oldData = &EntryData{
			Targets:     []string{},
			Webmentions: []string{},
		}
	}

	newData, err := t(oldData)
	if err != nil {
		return err
	}

	return e.PersistJSON(filename, newData, "entry data: update "+entry.ID)
}
