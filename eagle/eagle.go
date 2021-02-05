package eagle

import (
	"bytes"
	"errors"
	"io/ioutil"
	"path/filepath"

	"github.com/hacdias/eagle/config"
	"github.com/hacdias/eagle/yaml"
)

type Eagle struct {
	*config.Config
}

func NewEagle(c *config.Config) (*Eagle, error) {
	return &Eagle{c}, nil
}

func (e *Eagle) GetEntry(id string) (*Entry, error) {
	file := filepath.Join(e.Source, "content", id+".md")

	raw, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	splits := bytes.SplitN(raw, []byte("\n---"), 2)
	if len(splits) != 2 {
		return nil, errors.New("could not parse file: splits !== 2")
	}

	entry := &Entry{
		ID:       id,
		Metadata: EntryMetadata{},
		Content:  bytes.TrimSpace(splits[1]),
	}

	err = yaml.Unmarshal(splits[0], &entry.Metadata)
	if err != nil {
		return nil, err
	}

	return entry, nil
}
