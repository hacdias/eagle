package services

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"

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
	isList := false
	raw, err := e.afs.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			file = id + "/_index.md"
			raw, err = e.afs.ReadFile(file)
			isList = true
		}

		if err != nil {
			return nil, err
		}
	}

	splits := bytes.SplitN(raw, []byte("\n---"), 2)
	if len(splits) != 2 {
		return nil, errors.New("could not parse file: splits !== 2")
	}

	content := bytes.TrimSpace(splits[1])

	entry := &eagle.Entry{
		ID:         id,
		Metadata:   eagle.EntryMetadata{},
		RawContent: content,
		Content:    content, // TODO
		Permalink:  "http://hacdias.com" + id,
		Section:    strings.Split(strings.TrimLeft(id, "/"), "/")[0], // TODO: check
		IsList:     isList,
	}

	err = yaml.Unmarshal(splits[0], &entry.Metadata)
	if err != nil {
		return nil, err
	}

	return entry, nil
}
