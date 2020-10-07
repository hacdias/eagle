package services

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"path"

	"gopkg.in/yaml.v2"
)

type Hugo struct {
	Source      string
	Destination string
	dataDir     string
	contentDir  string
}

func (h *Hugo) Build(clean bool) error {
	args := []string{"--minify", "--destination", h.Destination}

	if clean {
		args = append(args, "--gc", "--cleanDestinationDir")
	}

	cmd := exec.Command("hugo", args...)
	return cmd.Run()
}

func (h *Hugo) NewEntry() error {
	return nil
}

type HugoEntry struct {
	Post     string
	Metadata map[string]interface{}
	Content  string
}

func (h *Hugo) SaveEntry(e *HugoEntry) error {
	/*
		  if (meta.properties && !keepOriginal) {
				      meta.properties = converter.mf2ToInternal(meta.properties)
				    }

	*/

	filePath := path.Join(h.contentDir, e.Post)
	index := path.Join(filePath, "index.md")

	val, err := yaml.Marshal(&e.Metadata)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(index, []byte(fmt.Sprintf("---\n%s\n\n---%s", string(val), e.Content)), 0666)
	if err != nil {
		return err
	}

	return nil
}

func (h *Hugo) GetAll() error {
	return nil
}

func (h *Hugo) GetEntry() error {
	return nil
}

func (h *Hugo) GetEntryHTML() error {
	return nil
}
