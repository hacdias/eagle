package core

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"github.com/samber/lo"
)

type fsSync interface {
	Sync() (modified []ModifiedFile, err error)
	Persist(message string, filename ...string) error
}

type noopGit struct{}

func (g *noopGit) Persist(message string, file ...string) error {
	return nil
}

func (g *noopGit) Sync() ([]ModifiedFile, error) {
	return []ModifiedFile{}, nil
}

var nothingToCommit = []byte("nothing to commit, working tree clean")
var noChangedAdded = []byte("no changes added to commit")

type git struct {
	dir      string
	mu       sync.Mutex
	messages []string
}

func newGit(path string) fsSync {
	return &git{dir: path}
}

func (g *git) Persist(message string, filenames ...string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.messages = append(g.messages, message)
	return g.add(filenames...)
}

func (g *git) add(filenames ...string) error {
	filenames = lo.Map(filenames, func(v string, _ int) string {
		return strings.TrimPrefix(v, "/")
	})
	args := append([]string{"add"}, filenames...)
	cmd := exec.Command("git", args...)
	cmd.Dir = g.dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git error (%w): %s", err, string(out))
	}
	return nil
}

func (g *git) commit(message string) error {
	args := []string{"commit", "-m", message, "--"}
	cmd := exec.Command("git", args...)
	cmd.Dir = g.dir
	out, err := cmd.CombinedOutput()
	if err != nil && !bytes.Contains(out, nothingToCommit) && !bytes.Contains(out, noChangedAdded) {
		return fmt.Errorf("git error (%w): %s", err, string(out))
	}
	return nil
}

func (g *git) hasStaged() bool {
	cmd := exec.Command("git", "diff-index", "--quiet", "--cached", "HEAD", "--")
	cmd.Dir = g.dir
	err := cmd.Run()
	// NOTE: not totally correct. Other errors might have happened here.
	return err != nil
}

func (g *git) Sync() ([]ModifiedFile, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.hasStaged() {
		g.messages = lo.Uniq(g.messages)
		var message string

		if len(g.messages) == 1 {
			message = g.messages[0]
		} else {
			message = strings.Join(g.messages, "\n")
			message = "eagle: add staged changes \n\n" + message
		}

		err := g.commit(message)
		if err != nil {
			return nil, fmt.Errorf("failed to commit staged: %w", err)
		}
		g.messages = nil
	}

	oldCommit, err := g.currentCommit()
	if err != nil {
		return nil, fmt.Errorf("failed to get current commit: %w", err)
	}

	err = g.pull()
	if err != nil {
		return nil, fmt.Errorf("failed to pull: %w", err)
	}

	err = g.push()
	if err != nil {
		return nil, fmt.Errorf("failed to push: %w", err)
	}

	changedFiles, err := g.changedFiles(oldCommit)
	if err != nil {
		return nil, fmt.Errorf("failed to get changed files: %w", err)
	}

	return changedFiles, nil
}

func (g *git) push() error {
	cmd := exec.Command("git", "push")
	cmd.Dir = g.dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git error (%w): %s", err, string(out))
	}
	return nil
}

func (g *git) pull() error {
	cmd := exec.Command("git", "pull")
	cmd.Dir = g.dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git error (%w): %s", err, string(out))
	}
	return nil
}

func (g *git) currentCommit() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--verify", "HEAD")
	cmd.Dir = g.dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git error (%w): %s", err, string(out))
	}

	return strings.TrimSpace(string(out)), nil
}

func (g *git) fileContent(filename, commit string) (string, error) {
	cmd := exec.Command("git", "show", fmt.Sprintf("%s:%s", commit, filename))
	cmd.Dir = g.dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		if bytes.Contains(out, []byte("exists on disk, but not in")) {
			return "", nil
		}

		return "", fmt.Errorf("git error (%w): %s", err, string(out))
	}

	return strings.TrimSpace(string(out)), nil
}

func (g *git) changedFiles(since string) ([]ModifiedFile, error) {
	cmd := exec.Command("git", "show", "--name-only", "--format=tformat:", since+"...HEAD")
	cmd.Dir = g.dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git error (%w): %s", err, string(out))
	}

	rawFiles := strings.Split(string(out), "\n")
	files := []ModifiedFile{}

	for _, file := range rawFiles {
		file = strings.TrimSpace(file)
		if file != "" {
			content, err := g.fileContent(file, since)
			if err != nil {
				return nil, err
			}

			files = append(files, ModifiedFile{
				Filename: file,
				Content:  content,
			})
		}
	}

	return files, nil
}
