package core

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	ContentDirectory string = "content"
	DataDirectory    string = "data"

	RedirectsFile = "redirects"
	GoneFile      = "gone"
)

type ModifiedFile struct {
	Filename string
	Content  string
}

func (co *Core) Sync() ([]ModifiedFile, error) {
	modified, err := co.sourceSync.Sync()
	if err == nil {
		now := time.Now()
		co.lastSync.Store(&now)
	}
	return modified, err
}

// LastSync returns the time of the last successful sync since startup, or
// the zero time if there was none.
func (co *Core) LastSync() time.Time {
	if t := co.lastSync.Load(); t != nil {
		return *t
	}
	return time.Time{}
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

func (co *Core) MkdirAll(filename string) error {
	return co.sourceFS.MkdirAll(filename, 0777)
}

func (co *Core) Stat(filename string) (os.FileInfo, error) {
	return co.sourceFS.Stat(filename)
}

func (co *Core) WriteJSON(filename string, data any, message string) error {
	json, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return co.WriteFile(filename, json, message)
}

func (co *Core) ReadJSON(filename string, v any) error {
	data, err := co.sourceFS.ReadFile(filename)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, v)
}

func (co *Core) RemoveAll(path string) error {
	return co.sourceFS.RemoveAll(path)
}

type Redirect struct {
	To     string
	Status int
}

var redirectStatuses = map[string]int{
	"301": http.StatusMovedPermanently,
	"302": http.StatusFound,
	"307": http.StatusTemporaryRedirect,
	"308": http.StatusPermanentRedirect,
}

func (co *Core) GetRedirects(ignoreMalformed bool) (map[string]Redirect, error) {
	redirects := map[string]Redirect{}

	data, err := co.sourceFS.ReadFile(RedirectsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return redirects, nil
		}
		return nil, err
	}

	lines := strings.SplitSeq(string(data), "\n")

	for line := range lines {
		parts := strings.Fields(line)
		if len(parts) == 0 || strings.HasPrefix(parts[0], "#") {
			continue
		}

		status, ok := http.StatusMovedPermanently, len(parts) == 2
		if len(parts) == 3 {
			status, ok = redirectStatuses[parts[2]]
		}

		if ok {
			redirects[parts[0]] = Redirect{To: parts[1], Status: status}
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

	lines := strings.SplitSeq(string(data), "\n")

	for line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		gone[line] = true
	}

	return gone, nil
}
