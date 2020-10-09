package services

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/hacdias/eagle/config"
	"github.com/hacdias/eagle/yaml"
	"github.com/karlseguin/typed"
)

type HugoEntry struct {
	ID       string
	Metadata typed.Typed
	Content  string
}

type Hugo struct {
	config.Hugo
}

func (h *Hugo) Build(clean bool) error {
	args := []string{"--minify", "--destination", h.Destination}

	if clean {
		args = append(args, "--gc", "--cleanDestinationDir")
	}

	cmd := exec.Command("hugo", args...)
	cmd.Dir = h.Source
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("hugo run failed: %s: %s", err, out)
	}
	return nil
}

func (h *Hugo) SaveEntry(e *HugoEntry) error {
	if prop, ok := e.Metadata.MapIf("properties"); ok {
		e.Metadata["properties"] = h.mf2ToInternal(prop)
	}

	filePath := filepath.Join(h.Source, "content", e.ID)
	err := os.MkdirAll(filePath, 0777)
	if err != nil {
		return err
	}

	index := filepath.Join(filePath, "index.md")

	val, err := yaml.Marshal(&e.Metadata)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(index, []byte(fmt.Sprintf("---\n%s---\n\n%s", string(val), e.Content)), 0644)
	if err != nil {
		return fmt.Errorf("could not save entry: %s", err)
	}

	return nil
}

func (h *Hugo) GetEntry(id string) (*HugoEntry, error) {
	index := path.Join(h.Source, "content", id, "index.md")
	bytes, err := ioutil.ReadFile(index)
	if err != nil {
		return nil, err
	}

	splits := strings.SplitN(string(bytes), "\n---", 2)
	if len(splits) != 2 {
		return nil, errors.New("could not parse file: splits !== 2")
	}

	entry := &HugoEntry{
		ID:       id,
		Metadata: map[string]interface{}{},
		Content:  strings.TrimSpace(splits[1]),
	}

	var metadata map[string]interface{}

	err = yaml.Unmarshal([]byte(splits[0]), &metadata)
	if err != nil {
		return nil, err
	}

	entry.Metadata = metadata

	if props, ok := entry.Metadata["properties"]; ok {
		entry.Metadata["properties"] = h.internalToMf2(props)
	} else {
		entry.Metadata["properties"] = map[string][]interface{}{}
	}

	return entry, nil
}

func (h *Hugo) GetEntryHTML(id string) ([]byte, error) {
	index := path.Join(h.Destination, id, "index.html")
	return ioutil.ReadFile(index)
}
