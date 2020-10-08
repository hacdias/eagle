package services

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os/exec"
	"path"
	"strings"
	"sync"

	"github.com/hacdias/eagle/config"
	"github.com/karlseguin/typed"
	"gopkg.in/yaml.v2"
)

type HugoEntry struct {
	ID       string
	Metadata typed.Typed
	Content  string
}

type Hugo struct {
	*sync.Mutex
	config.Hugo
}

func (h *Hugo) Build(clean bool) error {
	h.Lock()
	defer h.Unlock()

	args := []string{"--minify", "--destination", h.Destination}

	if clean {
		args = append(args, "--gc", "--cleanDestinationDir")
	}

	cmd := exec.Command("hugo", args...)
	return cmd.Run()
}

func (h *Hugo) SaveEntry(e *HugoEntry) error {
	if prop, ok := e.Metadata.MapIf("properties"); ok {
		e.Metadata["properties"] = h.mf2ToInternal(prop)
	}

	filePath := path.Join(h.Source, "content", e.ID)
	index := path.Join(filePath, "index.md")

	val, err := yaml.Marshal(&e.Metadata)
	if err != nil {
		return err
	}

	h.Lock()
	defer h.Unlock()

	err = ioutil.WriteFile(index, []byte(fmt.Sprintf("---\n%s\n\n---%s", string(val), e.Content)), 0644)
	if err != nil {
		return err
	}

	return nil
}

func (h *Hugo) GetAll() error {
	h.Lock()
	defer h.Unlock()

	// TODO:

	/*

	   const getAllFiles = function (dirPath, arrayOfFiles) {
	     const files = fs.readdirSync(dirPath)

	     arrayOfFiles = arrayOfFiles || []

	     files.forEach(function (file) {
	       if (fs.statSync(dirPath + '/' + file).isDirectory()) {
	         arrayOfFiles = getAllFiles(dirPath + '/' + file, arrayOfFiles)
	       } else {
	         arrayOfFiles.push(join(dirPath, '/', file))
	       }
	     })

	     return arrayOfFiles
	   }
	*/

	return nil
}

func (h *Hugo) GetEntry(id string) (*HugoEntry, error) {
	h.Lock()
	defer h.Unlock()

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

	err = yaml.Unmarshal([]byte(splits[0]), &entry.Metadata)
	if err != nil {
		return nil, err
	}

	if props, ok := entry.Metadata["properties"]; ok {
		entry.Metadata["properties"] = h.internalToMf2(props)
	} else {
		entry.Metadata["properties"] = map[string]interface{}{}
	}

	return entry, nil
}

func (h *Hugo) GetEntryHTML(id string) ([]byte, error) {
	h.Lock()
	defer h.Unlock()

	index := path.Join(h.Destination, id, "index.html")
	return ioutil.ReadFile(index)
}
