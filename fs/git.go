package fs

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

var nothingToCommit = []byte("nothing to commit, working tree clean")

type GitSync struct {
	dir string
	mu  sync.Mutex
}

func NewGitSync(path string) Sync {
	return &GitSync{dir: path}
}

func (g *GitSync) Persist(msg string, file ...string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	args := append([]string{"add"}, file...)
	cmd := exec.Command("git", args...)
	cmd.Dir = g.dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git error (%w): %s", err, string(out))
	}

	args = []string{"commit", "-m", msg, "--"}
	args = append(args, file...)
	cmd = exec.Command("git", args...)
	cmd.Dir = g.dir
	out, err = cmd.CombinedOutput()
	if err != nil && !bytes.Contains(out, nothingToCommit) {
		return fmt.Errorf("git error (%w): %s", err, string(out))
	}
	return nil
}

func (g *GitSync) Sync() ([]string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	oldCommit, err := g.currentCommit()
	if err != nil {
		return nil, err
	}

	err = g.pull()
	if err != nil {
		return nil, err
	}

	err = g.push()
	if err != nil {
		return nil, err
	}

	changedFiles, err := g.changedFiles(oldCommit)
	if err != nil {
		return nil, err
	}

	return changedFiles, nil
}

func (g *GitSync) push() error {
	cmd := exec.Command("git", "push")
	cmd.Dir = g.dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git error (%w): %s", err, string(out))
	}
	return nil
}

func (g *GitSync) pull() error {
	cmd := exec.Command("git", "pull")
	cmd.Dir = g.dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git error (%w): %s", err, string(out))
	}
	return nil
}

func (g *GitSync) currentCommit() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--verify", "HEAD")
	cmd.Dir = g.dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}

func (g *GitSync) changedFiles(since string) ([]string, error) {
	cmd := exec.Command("git", "show", "--name-only", "--format=tformat:", since+"...HEAD")
	cmd.Dir = g.dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	rawFiles := strings.Split(string(out), "\n")
	files := []string{}

	for _, file := range rawFiles {
		file = strings.TrimSpace(file)
		if file != "" {
			files = append(files, file)
		}
	}

	return files, nil
}
