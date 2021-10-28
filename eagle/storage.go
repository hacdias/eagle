package eagle

import (
	"github.com/spf13/afero"
)

type Storage struct {
	*afero.Afero
	remote *gitRepo
}

func NewStorage(path string) *Storage {
	return &Storage{
		Afero: &afero.Afero{
			Fs: afero.NewBasePathFs(afero.NewOsFs(), path),
		},
		remote: &gitRepo{
			dir: path,
		},
	}
}

func (fm *Storage) Persist(filename string, data []byte, message string) error {
	err := fm.WriteFile(filename, data, 0644)
	if err != nil {
		return err
	}

	err = fm.remote.addAndCommit(message, filename)
	if err != nil {
		return err
	}

	return nil
}

func (fm *Storage) Sync() ([]string, error) {
	return fm.remote.pullAndPush()
}
