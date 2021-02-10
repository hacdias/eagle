package eagle

import (
	"os/exec"
)

type StorageService interface {
	// Persist persists the files into the storage. If no files are passed
	// everything should be persisted.
	Persist(msg string, files ...string) error

	// Sync syncs the storage with whatever sync service we're using. In case
	// of a CVS, it might be pull + push.
	Sync() error
}

type GitStorage struct {
	Directory string
}

func (g *GitStorage) Persist(msg string, files ...string) error {
	if len(files) == 0 {
		cmd := exec.Command("git", "add", "-A")
		cmd.Dir = g.Directory
		err := cmd.Run()
		if err != nil {
			return err
		}

		cmd = exec.Command("git", "commit", "-m", msg)
		cmd.Dir = g.Directory
		err = cmd.Run()
		return err
	}

	args := []string{"commit", "-m", msg, "--"}
	args = append(args, files...)
	cmd := exec.Command("git", args...)
	cmd.Dir = g.Directory
	err := cmd.Run()
	return err
}

func (g *GitStorage) Sync() error {
	cmd := exec.Command("git", "pull")
	cmd.Dir = g.Directory
	err := cmd.Run()
	if err != nil {
		return err
	}

	cmd = exec.Command("git", "push")
	cmd.Dir = g.Directory
	return cmd.Run()
}

type PlaceboStorage struct{}

func (p *PlaceboStorage) Persist(msg string, files ...string) error {
	return nil
}

func (p *PlaceboStorage) Sync() error {
	return nil
}
