package core

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

type Sync interface {
	Sync() (updated []string, err error)
	Persist(message string, filename ...string) error
}

type NopSync struct{}

func (g *NopSync) Persist(message string, file ...string) error {
	return nil
}

func (g *NopSync) Sync() ([]string, error) {
	return []string{}, nil
}

var nothingToCommit = []byte("nothing to commit, working tree clean")
var noChangedAdded = []byte("no changes added to commit")

type GitSync struct {
	dir      string
	mu       sync.Mutex
	messages []string
}

func NewGitSync(path string) Sync {
	return &GitSync{dir: path}
}

func (g *GitSync) Persist(message string, filenames ...string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.messages = append(g.messages, message)
	return g.add(filenames...)
}

func (g *GitSync) add(filenames ...string) error {
	args := append([]string{"add"}, filenames...)
	cmd := exec.Command("git", args...)
	cmd.Dir = g.dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git error (%w): %s", err, string(out))
	}
	return nil
}

func (g *GitSync) commit(message string) error {
	args := []string{"commit", "-m", message, "--"}
	cmd := exec.Command("git", args...)
	cmd.Dir = g.dir
	out, err := cmd.CombinedOutput()
	if err != nil && !bytes.Contains(out, nothingToCommit) && !bytes.Contains(out, noChangedAdded) {
		return fmt.Errorf("git error (%w): %s", err, string(out))
	}
	return nil
}

func (g *GitSync) hasStaged() bool {
	cmd := exec.Command("git", "diff-index", "--quiet", "--cached", "HEAD", "--")
	cmd.Dir = g.dir
	err := cmd.Run()
	// NOTE: not totally correct. Other errors might have happened here.
	return err != nil
}

func (g *GitSync) Sync() ([]string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.hasStaged() {
		var message string
		if len(g.messages) == 1 {
			message = g.messages[0]
		} else {
			message = strings.Join(g.messages, "\n")
			message = "eagle: add staged changes \n\n" + message
		}

		err := g.commit(message)
		if err != nil {
			return nil, err
		}
		g.messages = nil
	}

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
