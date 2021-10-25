package eagle

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
)

type RemoteStorage interface {
	// Persist persists the files into the storage. If no files are passed
	// everything should be persisted.
	Persist(msg string, file string) error

	// Sync syncs the storage with whatever sync service we're using. In case
	// of a CVS, it might be pull + push.
	Sync() ([]string, error)
}

type GitStorage struct {
	dir string
}

func (g *GitStorage) Persist(msg string, file string) error {
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

type Storage struct {
	*afero.Afero
	remote  RemoteStorage
	prepend string
}

func NewStorage(path string, remote RemoteStorage) *Storage {
	return &Storage{
		Afero: &afero.Afero{
			Fs: afero.NewBasePathFs(afero.NewOsFs(), path),
		},
		remote:  remote,
		prepend: "",
	}
}

func (fm *Storage) Persist(path string, data []byte, message string) error {
	err := fm.WriteFile(path, data, 0644)
	if err != nil {
		return err
	}

	fullPath := filepath.Join(fm.prepend, path)
	err = fm.remote.Persist(message, fullPath)
	if err != nil {
		return err
	}

	return nil
}

func (fm *Storage) Sync() ([]string, error) {
	return fm.remote.Sync()
}

func (fm *Storage) Sub(path string) *Storage {
	fs := afero.NewBasePathFs(fm.Afero.Fs, path)
	prepend := filepath.Join(fm.prepend, path)

	return &Storage{
		Afero:   &afero.Afero{Fs: fs},
		remote:  fm.remote,
		prepend: prepend,
	}
}
