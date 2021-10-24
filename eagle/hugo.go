package eagle

import (
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/hacdias/eagle/config"
	"github.com/spf13/afero"
)

type Hugo struct {
	sync.Mutex

	conf          config.Hugo
	dstFs         *afero.Afero
	publicDirCh   chan string
	currentSubDir string
}

// ShouldBuild should only be called on startup to make sure there's
// a built public directory to serve.
func (h *Hugo) ShouldBuild() (bool, error) {
	h.Lock()
	defer h.Unlock()

	if h.currentSubDir != "" {
		return false, nil
	}

	content, err := h.dstFs.ReadFile("last")
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}

		return true, err
	}

	h.currentSubDir = string(content)
	h.publicDirCh <- filepath.Join(h.conf.Destination, h.currentSubDir)
	return false, nil
}

func (h *Hugo) Build(clean bool) error {
	if h.currentSubDir == "" {
		_, err := h.ShouldBuild()
		if err != nil {
			return err
		}
	}

	h.Lock()
	defer h.Unlock()

	dir := h.currentSubDir
	new := dir == "" || clean

	if new {
		dir = generateHash()
	}

	destination := filepath.Join(h.conf.Destination, dir)
	args := []string{"--minify", "--destination", destination}

	cmd := exec.Command("hugo", args...)
	cmd.Dir = h.conf.Source
	out, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf("hugo run failed: %w: %s", err, out)
	}

	if new {
		// We build to a different sub directory so we can change the directory
		// we are serving seamlessly without users noticing. Check server/satic.go!
		err = h.dstFs.WriteFile("last", []byte(dir), 0644)
		if err != nil {
			return fmt.Errorf("could not write last dir: %w", err)
		}

		h.currentSubDir = dir
		h.publicDirCh <- filepath.Join(h.conf.Destination, h.currentSubDir)
	}

	return nil
}

func generateHash() string {
	h := fnv.New64a()
	// the implementation does not return errors
	_, _ = h.Write([]byte(time.Now().UTC().String()))
	return hex.EncodeToString(h.Sum(nil))
}
