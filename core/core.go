package core

import (
	"context"
	"net/http"
	"net/url"
	"path/filepath"
	"sync"
	"time"

	"github.com/spf13/afero"
	"willnorris.com/go/webmention"
)

type Core struct {
	cfg      *Config
	baseURL  *url.URL
	db       *Database
	queue    *Queue
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
	db, err := newDatabase(filepath.Join(cfg.DataDirectory, "eagle.db"))
	if err != nil {
		return nil, err
	}

	co := &Core{
		cfg:   cfg,
		db:    db,
		queue: newQueue(db),
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

// DB returns the database.
func (co *Core) DB() *Database {
	return co.db
}

// Queue returns the queue processor.
func (co *Core) Queue() *Queue {
	return co.queue
}

// Close closes the database.
func (co *Core) Close() error {
	return co.db.Close()
}

// Enqueue adds an item to the processing queue.
func (co *Core) Enqueue(ctx context.Context, typ string, payload any) error {
	return co.queue.Enqueue(ctx, typ, payload)
}
