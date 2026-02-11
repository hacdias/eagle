package media

import (
	"io"

	"github.com/spf13/afero"
	"go.hacdias.com/eagle/core"
)

var (
	_ Storage = &FileSystem{}
)

type FileSystem struct {
	base string
	fs   *afero.Afero
}

func NewFileSystem(conf *core.FileSystem) *FileSystem {
	return &FileSystem{
		base: conf.Base,
		fs: &afero.Afero{
			Fs: afero.NewBasePathFs(afero.NewOsFs(), conf.Directory),
		},
	}
}

func (fs *FileSystem) BaseURL() string {
	return fs.base
}

func (fs *FileSystem) UploadMedia(filename string, data io.Reader) (string, error) {
	return fs.base + "/" + filename, fs.fs.WriteReader(filename, data)
}
