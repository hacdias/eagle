package core

import (
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/spf13/afero"
)

type Hugo struct {
	mu sync.Mutex

	// srcDir and pubDir are the source and public directory, respectively,
	// for the Hugo builds. Note that pubDir is just the parent directory
	// of the directory where the build will actually be placed.
	srcDir string
	pubDir string

	baseURL string

	// fs is an Afero filesystem wrapper for pubDir.
	fs *afero.Afero

	// current is the ID of the current build. This corresponds with
	// a subdirectory of pubDir.
	current string

	// BuildHook is a hook that is called after building if the public directory
	// has changed.
	BuildHook func(string)
}

func NewHugo(srcDir, pubDir, baseURL string) *Hugo {
	return &Hugo{
		srcDir:  srcDir,
		pubDir:  pubDir,
		baseURL: baseURL,
		fs: &afero.Afero{
			Fs: afero.NewBasePathFs(afero.NewOsFs(), pubDir),
		},
	}
}

// ShouldBuild indicates if the website should be built. This should only
// return true after initialization.
func (h *Hugo) ShouldBuild() (bool, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.current != "" {
		return false, nil
	}

	content, err := h.fs.ReadFile("last")
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}

		return true, err
	}

	h.current = string(content)
	if h.BuildHook != nil {
		h.BuildHook(filepath.Join(h.pubDir, h.current))
	}
	return false, nil
}

func (h *Hugo) Build(clean bool) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	dir := h.current
	new := dir == "" || clean

	if new {
		h := fnv.New64a()
		_, err := h.Write([]byte(time.Now().UTC().String()))
		if err != nil {
			return fmt.Errorf("failed to generate hash: %w", err)
		}
		dir = hex.EncodeToString(h.Sum(nil))
	}

	destination := filepath.Join(h.pubDir, dir)
	args := []string{"--minify", "--destination", destination, "--baseURL", h.baseURL, "--environment", "eagle"}

	cmd := exec.Command("hugo", args...)
	cmd.Dir = h.srcDir
	out, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf("hugo run failed: %w: %s", err, out)
	}

	if new {
		err = h.fs.WriteFile("last", []byte(dir), 0644)
		if err != nil {
			return fmt.Errorf("could not write last dir: %w", err)
		}

		h.current = dir
		if h.BuildHook != nil {
			h.BuildHook(destination)
		}
	}

	return nil
}
