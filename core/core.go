package core

import (
	"sync"

	"github.com/spf13/afero"
)

type Core struct {
	cfg *Config

	// Source
	sourceFS   *afero.Afero
	sourceSync fsSync

	// Build
	buildMu   sync.Mutex
	buildFS   *afero.Afero // afero around [Config.PublicDirectory]
	buildName string       // the name of the current build (sub-directory in buildFS)
	BuildHook func(string) // called when the build directory has changed

	// TODO: add method to fetch HTML of built entry
}

func NewCore(cfg *Config) (*Core, error) {
	co := &Core{
		cfg: cfg,

		// Source
		sourceFS: &afero.Afero{
			Fs: afero.NewBasePathFs(afero.NewOsFs(), cfg.SourceDirectory),
		},

		// Build
		buildFS: &afero.Afero{
			Fs: afero.NewBasePathFs(afero.NewOsFs(), cfg.PublicDirectory),
		},
	}

	if cfg.Development {
		co.sourceSync = &noopGit{}
	} else {
		co.sourceSync = newGit(cfg.SourceDirectory)
	}

	return co, nil
}
