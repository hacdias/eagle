package core

import (
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/spf13/afero"
	"willnorris.com/go/webmention"
)

type Core struct {
	cfg      *Config
	baseURL  *url.URL
	wmClient *webmention.Client

	// Source
	sourceFS   *afero.Afero
	sourceSync fsSync

	// Build
	buildMu   sync.Mutex
	buildFS   *afero.Afero // afero around [Config.PublicDirectory]
	buildName string       // the name of the current build (sub-directory in buildFS)
	BuildHook func(string) // called when the build directory has changed
}

func NewCore(cfg *Config) (*Core, error) {
	co := &Core{
		cfg: cfg,
		wmClient: webmention.New(&http.Client{
			Timeout: time.Minute,
		}),

		// Source
		sourceFS: &afero.Afero{
			Fs: afero.NewBasePathFs(afero.NewOsFs(), cfg.SourceDirectory),
		},

		// Build
		buildFS: &afero.Afero{
			Fs: afero.NewBasePathFs(afero.NewOsFs(), cfg.PublicDirectory),
		},
	}

	baseURL, err := url.Parse(cfg.Site.BaseURL)
	if err != nil {
		return nil, err
	}
	co.baseURL = baseURL

	if cfg.Development {
		co.sourceSync = &noopGit{}
	} else {
		co.sourceSync = newGit(cfg.SourceDirectory)
	}

	return co, nil
}

// BaseURL returns a clone of the base URL.
func (co *Core) BaseURL() *url.URL {
	return cloneURL(co.baseURL)
}

// SiteConfig returns the site configuration.
func (co *Core) SiteConfig() SiteConfig {
	return co.cfg.Site
}
