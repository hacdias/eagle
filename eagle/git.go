package eagle

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

var nothingToCommit = []byte("nothing to commit, working tree clean")

type gitRepo struct {
	dir string
}

func (g *gitRepo) addAndCommit(msg string, file string) error {
	args := append([]string{"add"}, file)
	cmd := exec.Command("git", args...)
	cmd.Dir = g.dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git error (%w): %s", err, string(out))
	}

	args = []string{"commit", "-m", msg, "--"}
	args = append(args, file)
	cmd = exec.Command("git", args...)
	cmd.Dir = g.dir
	out, err = cmd.CombinedOutput()
	if err != nil && !bytes.Contains(out, nothingToCommit) {
		return fmt.Errorf("git error (%w): %s", err, string(out))
	}
	return nil
}

func (g *gitRepo) pullAndPush() ([]string, error) {
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

func (g *gitRepo) push() error {
	cmd := exec.Command("git", "push")
	cmd.Dir = g.dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git error (%w): %s", err, string(out))
	}
	return nil
}

func (g *gitRepo) pull() error {
	cmd := exec.Command("git", "pull")
	cmd.Dir = g.dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git error (%w): %s", err, string(out))
	}
	return nil
}

func (g *gitRepo) currentCommit() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--verify", "HEAD")
	cmd.Dir = g.dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}

func (g *gitRepo) changedFiles(since string) ([]string, error) {
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
