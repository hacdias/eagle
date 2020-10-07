package services

import (
	"os/exec"
	"sync"
)

type Git struct {
	*sync.Mutex
	Directory string
}

func (g *Git) Commit(msg string) error {
	g.Lock()
	cmd := exec.Command("git", "add", "-A")
	cmd.Dir = g.Directory
	err := cmd.Run()

	if err != nil {
		return err
	}

	cmd = exec.Command("git", "commit", "-m", msg)
	cmd.Dir = g.Directory
	err = cmd.Run()
	g.Unlock()
	return err
}

func (g *Git) CommitFile(msg string, files ...string) error {
	g.Lock()
	args := []string{"commit", "-m", msg, "--"}
	args = append(args, files...)
	cmd := exec.Command("git", args...)
	cmd.Dir = g.Directory
	err := cmd.Run()
	g.Unlock()
	return err
}

func (g *Git) Push() error {
	g.Lock()
	cmd := exec.Command("git", "push")
	cmd.Dir = g.Directory
	err := cmd.Run()
	g.Unlock()
	return err
}

func (g *Git) Pull() error {
	g.Lock()
	cmd := exec.Command("git", "pull")
	cmd.Dir = g.Directory
	err := cmd.Run()
	g.Unlock()
	return err
}
