package core

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

const (
	ContentDirectory string = "content"
	DataDirectory    string = "data"

	RedirectsFile = "redirects"
	GoneFile      = "gone"
)

func (co *Core) Sync() ([]string, error) {
	return co.sourceSync.Sync()
}

func (co *Core) WriteFile(filename string, data []byte, message string) error {
	err := co.sourceFS.WriteFile(filename, data, 0644)
	if err != nil {
		return err
	}

	return co.sourceSync.Persist(message, filename)
}

func (co *Core) WriteFiles(filesAndData map[string][]byte, message string) error {
	var filenames []string

	for filename, data := range filesAndData {
		err := co.sourceFS.WriteFile(filename, data, 0644)
		if err != nil {
			return err
		}
		filenames = append(filenames, filename)
	}

	return co.sourceSync.Persist(message, filenames...)
}

func (co *Core) ReadFile(filename string) ([]byte, error) {
	return co.sourceFS.ReadFile(filename)
}

func (co *Core) ReadDir(filename string) ([]os.FileInfo, error) {
	return co.sourceFS.ReadDir(filename)
}

func (co *Core) Stat(filename string) (os.FileInfo, error) {
	return co.sourceFS.Stat(filename)
}

func (co *Core) WriteJSON(filename string, data interface{}, message string) error {
	json, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return co.WriteFile(filename, json, message)
}

func (co *Core) ReadJSON(filename string, v interface{}) error {
	data, err := co.sourceFS.ReadFile(filename)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, v)
}

func (co *Core) RemoveAll(path string) error {
	return co.sourceFS.RemoveAll(path)
}

func (co *Core) GetRedirects(ignoreMalformed bool) (map[string]string, error) {
	redirects := map[string]string{}

	data, err := co.sourceFS.ReadFile(RedirectsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return redirects, nil
		}
		return nil, err
	}

	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		parts := strings.Split(line, " ")
		if len(parts) == 2 {
			redirects[parts[0]] = parts[1]
		} else if !ignoreMalformed {
			return nil, fmt.Errorf("found invalid redirect entry: %s", line)
		}
	}

	return redirects, nil
}

func (co *Core) GetGone() (map[string]bool, error) {
	gone := map[string]bool{}

	data, err := co.sourceFS.ReadFile(GoneFile)
	if err != nil {
		if os.IsNotExist(err) {
			return gone, nil
		}
		return nil, err
	}

	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		gone[line] = true
	}

	return gone, nil
}
