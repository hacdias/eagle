package eagle

import (
	"path/filepath"

	"github.com/spf13/afero"
)

type Storage struct {
	*afero.Afero
	remote  *gitRepo
	prepend string
}

func NewStorage(path string, remote *gitRepo) *Storage {
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
	err = fm.remote.addAndCommit(message, fullPath)
	if err != nil {
		return err
	}

	return nil
}

func (fm *Storage) Sync() ([]string, error) {
	return fm.remote.pullAndPush()
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
