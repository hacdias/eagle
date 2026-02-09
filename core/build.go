package core

import (
	"encoding/hex"
	"fmt"
	"hash/fnv"
	urlpkg "net/url"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// ShouldBuild returns true if the website has to be built. This should only
// return true after initialization.
func (co *Core) ShouldBuild() (bool, error) {
	co.buildMu.Lock()
	defer co.buildMu.Unlock()

	if co.buildName != "" {
		return false, nil
	}

	content, err := co.buildFS.ReadFile("last")
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}

		return true, err
	}

	co.buildName = string(content)
	if co.BuildHook != nil {
		co.BuildHook(filepath.Join(co.cfg.PublicDirectory, co.buildName))
	}
	return false, nil
}

func (co *Core) Build(cleanBuildDirectory bool) error {
	co.buildMu.Lock()
	defer co.buildMu.Unlock()

	dir := co.buildName
	new := dir == "" || cleanBuildDirectory

	if new {
		h := fnv.New64a()
		_, err := h.Write([]byte(time.Now().UTC().String()))
		if err != nil {
			return fmt.Errorf("failed to generate hash: %w", err)
		}
		dir = hex.EncodeToString(h.Sum(nil))
	}

	destination := filepath.Join(co.cfg.PublicDirectory, dir)
	args := []string{
		"--minify",
		"--destination", destination,
		"--baseURL", co.cfg.Site.BaseURL,
		"--environment", "eagle",
	}

	cmd := exec.Command("hugo", args...)
	cmd.Dir = co.cfg.SourceDirectory
	out, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf("hugo run failed: %w: %s", err, out)
	}

	if new {
		err = co.buildFS.WriteFile("last", []byte(dir), 0644)
		if err != nil {
			return fmt.Errorf("could not write last dir: %w", err)
		}

		co.buildName = dir
		if co.BuildHook != nil {
			co.BuildHook(destination)
		}
	}

	return nil
}

// IsLinkValid checks if the given link exists in the built version of the website.
func (co *Core) IsLinkValid(permalink string) (bool, error) {
	url, err := urlpkg.Parse(permalink)
	if err != nil {
		return false, err
	}

	_, err = co.buildFS.Stat(filepath.Join(co.buildName, url.Path))
	if err == nil {
		return true, nil
	}

	_, err = co.buildFS.Stat(filepath.Join(co.buildName, url.Path, "index.html"))
	if err == nil {
		return true, err
	}

	return false, nil
}
