package eagle

import (
	"fmt"
	"os/exec"
	"strings"
)

type StorageService interface {
	// Persist persists the files into the storage. If no files are passed
	// everything should be persisted.
	Persist(msg string, files ...string) error

	// Sync syncs the storage with whatever sync service we're using. In case
	// of a CVS, it might be pull + push.
	Sync() ([]string, error)
}

type GitStorage struct {
	dir string
}

func (g *GitStorage) Persist(msg string, files ...string) error {
	if len(files) == 0 {
		cmd := exec.Command("git", "add", "-A")
		cmd.Dir = g.dir
		err := cmd.Run()
		if err != nil {
			return err
		}

		cmd = exec.Command("git", "commit", "-m", msg)
		cmd.Dir = g.dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("git error (%w): %s", err, string(out))
		}
		return nil
	}

	args := append([]string{"add"}, files...)
	cmd := exec.Command("git", args...)
	cmd.Dir = g.dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git error (%w): %s", err, string(out))
	}

	args = []string{"commit", "-m", msg, "--"}
	args = append(args, files...)
	cmd = exec.Command("git", args...)
	cmd.Dir = g.dir
	out, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git error (%w): %s", err, string(out))
	}
	return nil
}

func (g *GitStorage) Sync() ([]string, error) {
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

func (g *GitStorage) push() error {
	cmd := exec.Command("git", "push")
	cmd.Dir = g.dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git error (%w): %s", err, string(out))
	}
	return nil
}

func (g *GitStorage) pull() error {
	cmd := exec.Command("git", "pull")
	cmd.Dir = g.dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git error (%w): %s", err, string(out))
	}
	return nil
}

func (g *GitStorage) currentCommit() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--verify", "HEAD")
	cmd.Dir = g.dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}

func (g *GitStorage) changedFiles(since string) ([]string, error) {
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

type PlaceboStorage struct{}

func (p *PlaceboStorage) Persist(msg string, files ...string) error {
	return nil
}

func (p *PlaceboStorage) Sync() ([]string, error) {
	return []string{}, nil
}
