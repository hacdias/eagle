package eagle

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/dgraph-io/ristretto"
	"github.com/hacdias/eagle/v4/config"
	"github.com/hacdias/eagle/v4/database"
	"github.com/hacdias/eagle/v4/entry"
	"github.com/hacdias/eagle/v4/fs"
	"github.com/hacdias/eagle/v4/log"
	"github.com/hacdias/eagle/v4/notifier"
	"github.com/spf13/afero"
	"github.com/thoas/go-funk"

	"go.uber.org/zap"
)

const (
	AssetsDirectory  string = "assets"
	ContentDirectory string = "content"
)

type Eagle struct {
	notifier.Notifier

	log        *zap.SugaredLogger
	httpClient *http.Client
	FS         *fs.FS
	DB         database.Database
	Parser     *entry.Parser
	Config     *config.Config

	// TODO: concerns only the server. Move there.
	cache     *ristretto.Cache
	redirects map[string]string
	// cron      *cron.Cron

	// TODO: (likely) concerns only specific hooks. Modularize and move them.
	media    *Media
	imgProxy *ImgProxy

	// Mutexes to lock the updates to entries and sidecars.
	// Only for writes and not for reads. Hope this won't
	// become a problem with traffic and simultaneous
	// reads-writes from files. Adding a mutex for all reads
	// would probably make it much slower though.
	entriesMu  sync.Mutex
	sidecarsMu sync.Mutex
}

func NewEagle(conf *config.Config) (*Eagle, error) {
	httpClient := &http.Client{
		Timeout: time.Minute * 2,
	}

	var srcSync fs.FSSync
	if conf.Development {
		srcSync = fs.NewPlaceboSync()
	} else {
		srcSync = fs.NewGitSync(conf.Source.Directory)
	}
	srcFs := fs.NewFS(conf.Source.Directory, srcSync)

	e := &Eagle{
		log:        log.S().Named("eagle"),
		httpClient: httpClient,
		FS:         srcFs,
		Config:     conf,
		Parser:     entry.NewParser(conf.Server.BaseURL),
	}

	if conf.BunnyCDN != nil {
		e.media = &Media{
			BunnyCDN: conf.BunnyCDN,
			httpClient: &http.Client{
				Timeout: time.Minute * 10,
			},
		}
	}

	if conf.ImgProxy != nil {
		e.imgProxy = &ImgProxy{
			endpoint: conf.ImgProxy.Endpoint,
			httpClient: &http.Client{
				Timeout: time.Minute * 10,
			},
			fs: &afero.Afero{
				Fs: afero.NewBasePathFs(afero.NewOsFs(), conf.ImgProxy.Directory),
			},
		}
	}

	if conf.Notifications.Telegram != nil {
		notifications, err := notifier.NewTelegramNotifier(conf.Notifications.Telegram)
		if err != nil {
			return nil, err
		}
		e.Notifier = notifications
	} else {
		e.Notifier = notifier.NewLogNotifier()
	}

	var err error

	e.DB, err = database.NewDatabase(&conf.PostgreSQL)
	if err != nil {
		return nil, err
	}

	err = e.initCache()
	if err != nil {
		return nil, err
	}

	err = e.initRedirects()
	if err != nil {
		return nil, err
	}

	go e.indexAll()
	return e, nil
}

func (e *Eagle) Close() {
	if e.DB != nil {
		e.DB.Close()
	}
}

func (e *Eagle) SyncStorage() {
	changedFiles, err := e.FS.Sync()
	if err != nil {
		e.Notifier.Error(fmt.Errorf("sync storage: %w", err))
		return
	}

	if len(changedFiles) == 0 {
		return
	}

	ids := []string{}

	for _, file := range changedFiles {
		if !strings.HasPrefix(file, ContentDirectory) {
			continue
		}

		id := strings.TrimPrefix(file, ContentDirectory)
		id = filepath.Dir(id)
		ids = append(ids, id)
	}

	// NOTE: we do not reload the templates and assets because
	// doing so is not concurrent-safe.
	ids = funk.UniqString(ids)
	entries := []*entry.Entry{}

	for _, id := range ids {
		entry, err := e.GetEntry(id)
		if os.IsNotExist(err) {
			e.DB.Remove(id)
			continue
		} else if err != nil {
			e.Notifier.Error(fmt.Errorf("cannot open entry to update %s: %w", id, err))
			continue
		}
		entries = append(entries, entry)
		e.RemoveCache(entry)
	}

	err = e.DB.Add(entries...)
	if err != nil {
		e.Notifier.Error(fmt.Errorf("sync failed: %w", err))
	}
}
