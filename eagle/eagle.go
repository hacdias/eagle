package eagle

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/dgraph-io/ristretto"
	"github.com/hacdias/eagle/v3/config"
	"github.com/hacdias/eagle/v3/database"
	"github.com/hacdias/eagle/v3/entry"
	"github.com/hacdias/eagle/v3/entry/mf2"
	"github.com/hacdias/eagle/v3/fs"
	"github.com/hacdias/eagle/v3/loctools"
	"github.com/hacdias/eagle/v3/log"
	"github.com/hacdias/eagle/v3/notifier"
	"github.com/hacdias/eagle/v3/syndicator"
	"github.com/robfig/cron/v3"
	"github.com/tdewolff/minify/v2"
	"github.com/thoas/go-funk"

	"github.com/yuin/goldmark"
	"go.uber.org/zap"
	"willnorris.com/go/webmention"
)

const (
	AssetsDirectory  string = "assets"
	ContentDirectory string = "content"
)

type Eagle struct {
	notifier.Notifier

	log          *zap.SugaredLogger
	httpClient   *http.Client
	wmClient     *webmention.Client
	fs           *fs.FS
	syndication  *syndicator.Manager
	allowedTypes []mf2.Type
	db           database.Database
	cache        *ristretto.Cache
	media        *Media
	miniflux     *Miniflux
	lastfm       *Lastfm
	Parser       *entry.Parser
	Config       *config.Config
	loctools     *loctools.LocTools

	// This can be changed while in development mode.
	assets    *Assets
	templates map[string]*template.Template

	// Things that are initialized once.
	redirects        map[string]string
	markdown         goldmark.Markdown
	absoluteMarkdown goldmark.Markdown
	minifier         *minify.M
	cron             *cron.Cron

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
		srcSync = fs.NewGitSync(conf.SourceDirectory)
	}
	srcFs := fs.NewFS(conf.SourceDirectory, srcSync)

	e := &Eagle{
		log:          log.S().Named("eagle"),
		httpClient:   httpClient,
		fs:           srcFs,
		wmClient:     webmention.New(httpClient),
		Config:       conf,
		allowedTypes: []mf2.Type{},
		syndication:  syndicator.NewManager(),
		Parser:       entry.NewParser(conf.Site.BaseURL),
		minifier:     initMinifier(),
		loctools:     loctools.NewLocTools(httpClient),
		cron:         cron.New(),
	}

	for typ := range conf.Site.MicropubTypes {
		e.allowedTypes = append(e.allowedTypes, typ)
	}

	if conf.BunnyCDN != nil {
		e.media = &Media{
			BunnyCDN: conf.BunnyCDN,
			httpClient: &http.Client{
				Timeout: time.Minute * 10,
			},
		}
	}

	if conf.Telegram != nil {
		notifications, err := notifier.NewTelegramNotifier(conf.Telegram)
		if err != nil {
			return nil, err
		}
		e.Notifier = notifications
	} else {
		e.Notifier = notifier.NewLogNotifier()
	}

	var err error

	e.db, err = database.NewDatabase(&conf.PostgreSQL)
	if err != nil {
		return nil, err
	}

	e.markdown = newMarkdown(e, false)
	e.absoluteMarkdown = newMarkdown(e, true)

	if conf.Twitter != nil {
		e.syndication.Add(syndicator.NewTwitter(conf.Twitter))
	}

	if conf.Miniflux != nil {
		e.miniflux = &Miniflux{Miniflux: conf.Miniflux}
	}

	if conf.Lastfm != nil {
		e.lastfm = &Lastfm{Lastfm: conf.Lastfm}
	}

	err = e.initCache()
	if err != nil {
		return nil, err
	}

	err = e.initRedirects()
	if err != nil {
		return nil, err
	}

	err = e.initAssets()
	if err != nil {
		return nil, err
	}

	err = e.initTemplates()
	if err != nil {
		return nil, err
	}

	err = e.initBlogrollCron()
	if err != nil {
		return nil, err
	}

	err = e.initScrobbleCron()
	if err != nil {
		return nil, err
	}

	go e.indexAll()
	return e, nil
}

func (e *Eagle) GetSyndicators() []*syndicator.Config {
	return e.syndication.Config()
}

func (e *Eagle) Close() {
	if e.db != nil {
		e.db.Close()
	}

	if e.cron != nil {
		e.cron.Stop()
	}
}

func (e *Eagle) userAgent(comment string) string {
	return fmt.Sprintf("Eagle/0.0 %s", comment)
}

func (e *Eagle) SyncStorage() {
	changedFiles, err := e.fs.Sync()
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
			e.db.Remove(id)
			continue
		} else if err != nil {
			e.Notifier.Error(fmt.Errorf("cannot open entry to update %s: %w", id, err))
			continue
		}
		entries = append(entries, entry)
		e.RemoveCache(entry)
	}

	err = e.db.Add(entries...)
	if err != nil {
		e.Notifier.Error(fmt.Errorf("sync failed: %w", err))
	}
}
