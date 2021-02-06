package services

import (
	"bytes"
	"errors"
	"path/filepath"

	"github.com/hacdias/eagle"
	"github.com/hacdias/eagle/config"
	"github.com/hacdias/eagle/yaml"
	"github.com/spf13/afero"
)

type EntryManager struct {
	*config.Config
	fs  afero.Fs
	afs *afero.Afero
}

func NewEntryManager(c *config.Config) (*EntryManager, error) {
	dir := filepath.Join(c.Source, "content")
	fs := afero.NewBasePathFs(afero.NewOsFs(), dir)

	return &EntryManager{
		Config: c,
		fs:     fs,
		afs:    &afero.Afero{Fs: fs},
	}, nil
}

func (e *EntryManager) GetEntry(id string) (*eagle.Entry, error) {
	file := id + ".md"
	raw, err := e.afs.ReadFile(file)
	if err != nil {
		return nil, err
	}

	splits := bytes.SplitN(raw, []byte("\n---"), 2)
	if len(splits) != 2 {
		return nil, errors.New("could not parse file: splits !== 2")
	}

	entry := &eagle.Entry{
		ID:       id,
		Metadata: eagle.EntryMetadata{},
		Content:  bytes.TrimSpace(splits[1]),
	}

	err = yaml.Unmarshal(splits[0], &entry.Metadata)
	if err != nil {
		return nil, err
	}

	return entry, nil
}
