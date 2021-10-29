package eagle

import (
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// ShouldBuild indicates if the website should be built. This should only
// return true after initialization.
func (e *Eagle) ShouldBuild() (bool, error) {
	e.buildMu.Lock()
	defer e.buildMu.Unlock()

	if e.currentPublicDir != "" {
		return false, nil
	}

	content, err := e.dstFs.ReadFile("last")
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}

		return true, err
	}

	e.currentPublicDir = string(content)
	e.PublicDirCh <- filepath.Join(e.Config.PublicDirectory, e.currentPublicDir)
	return false, nil
}

func (e *Eagle) Build(clean bool) error {
	// TODO: maybe this is not actually needed and can be removed.
	if e.currentPublicDir == "" {
		_, err := e.ShouldBuild()
		if err != nil {
			return err
		}
	}

	e.buildMu.Lock()
	defer e.buildMu.Unlock()

	dir := e.currentPublicDir
	new := dir == "" || clean

	if new {
		dir = generateHash()
	}

	destination := filepath.Join(e.Config.PublicDirectory, dir)
	args := []string{"--minify", "--destination", destination}

	cmd := exec.Command("hugo", args...)
	cmd.Dir = e.Config.SourceDirectory
	out, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf("hugo run failed: %w: %s", err, out)
	}

	if new {
		// We build to a different sub directory so we can change the directory
		// we are serving seamlessly without users noticing. Check server/satic.go!
		err = e.dstFs.WriteFile("last", []byte(dir), 0644)
		if err != nil {
			return fmt.Errorf("could not write last dir: %w", err)
		}

		e.currentPublicDir = dir
		e.PublicDirCh <- destination
	}

	return nil
}

func (e *Eagle) getEntryHTML(entry *Entry) ([]byte, error) {
	filename := entry.ID
	if !strings.HasSuffix(filename, ".html") {
		filename = path.Join(filename, "index.html")
	}

	e.buildMu.Lock()
	defer e.buildMu.Unlock()

	filename = filepath.Join(e.currentPublicDir, filename)
	return e.dstFs.ReadFile(filename)
}

func generateHash() string {
	h := fnv.New64a()
	// the implementation does not return errors
	_, _ = h.Write([]byte(time.Now().UTC().String()))
	return hex.EncodeToString(h.Sum(nil))
}
