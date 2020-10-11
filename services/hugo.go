package services

import (
	"encoding/hex"
	"errors"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/hacdias/eagle/config"
	"github.com/hacdias/eagle/yaml"
	"github.com/karlseguin/typed"
)

type HugoEntry struct {
	ID        string
	Permalink string
	Content   string
	Metadata  typed.Typed
}

type Hugo struct {
	config.Hugo
	Domain        string
	DirChanges    chan string
	currentSubDir string
}

func generateHash() string {
	h := fnv.New64a()
	// the implementation does not return errors
	_, _ = h.Write([]byte(time.Now().UTC().String()))
	return hex.EncodeToString(h.Sum(nil))
}

func (h *Hugo) Build(clean bool) error {
	dir := h.currentSubDir
	new := false

	if dir == "" {
		content, err := ioutil.ReadFile(path.Join(h.Destination, "last"))
		if err != nil {
			if !os.IsNotExist(err) {
				return err
			}
		} else {
			new = true
			dir = string(content)
		}
	}
	if dir == "" || clean {
		new = true
		dir = generateHash()
	}

	destination := filepath.Join(h.Destination, dir)
	args := []string{"--minify", "--destination", destination}

	cmd := exec.Command("hugo", args...)
	cmd.Dir = h.Source
	out, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf("hugo run failed: %s: %s", err, out)
	}

	if new {
		// We build to a different sub directory so we can change the directory
		// we are serving seamlessly without users noticing. Check server/satic.go!
		h.currentSubDir = dir
		h.DirChanges <- filepath.Join(h.Destination, h.currentSubDir)
		err = ioutil.WriteFile(path.Join(h.Destination, "last"), []byte(dir), 0644)
		if err != nil {
			return fmt.Errorf("could not write last dir: %s", err)
		}
	}

	return nil
}

func (h *Hugo) makeURL(id string) (string, error) {
	u, err := url.Parse(h.Domain)
	if err != nil {
		return "", err
	}
	u.Path = id
	return u.String(), nil
}

func (h *Hugo) SaveEntry(e *HugoEntry) error {
	if prop, ok := e.Metadata["properties"]; ok {
		e.Metadata["properties"] = mf2ToInternal(prop)
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

	permalink, err := h.makeURL(id)
	if err != nil {
		return nil, err
	}

	entry := &HugoEntry{
		ID:        id,
		Permalink: permalink,
		Metadata:  map[string]interface{}{},
		Content:   strings.TrimSpace(splits[1]),
	}

	var metadata map[string]interface{}

	err = yaml.Unmarshal([]byte(splits[0]), &metadata)
	if err != nil {
		return nil, err
	}

	entry.Metadata = metadata

	if props, ok := entry.Metadata["properties"]; ok {
		entry.Metadata["properties"] = internalToMf2(props)
	} else {
		entry.Metadata["properties"] = map[string][]interface{}{}
	}

	return entry, nil
}

func (h *Hugo) GetAll() ([]*HugoEntry, error) {
	entries := []*HugoEntry{}
	content := path.Join(h.Source, "content")

	err := filepath.Walk(h.Source, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Name() != "index.md" {
			return nil
		}

		id := strings.TrimPrefix(path.Dir(p), content)
		entry, err := h.GetEntry(id)
		if err != nil {
			return err
		}

		entries = append(entries, entry)
		return nil
	})

	return entries, err
}
