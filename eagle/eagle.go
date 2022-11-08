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
	"github.com/hacdias/eagle/v4/config"
	"github.com/hacdias/eagle/v4/database"
	"github.com/hacdias/eagle/v4/entry"
	"github.com/hacdias/eagle/v4/fs"
	"github.com/hacdias/eagle/v4/log"
	"github.com/hacdias/eagle/v4/notifier"
	"github.com/hacdias/eagle/v4/pkg/lastfm"
	"github.com/hacdias/eagle/v4/pkg/maze"
	"github.com/hacdias/eagle/v4/pkg/mf2"
	"github.com/hacdias/eagle/v4/pkg/miniflux"
	"github.com/hacdias/eagle/v4/syndicator"
	"github.com/hacdias/eagle/v4/xray"
	"github.com/robfig/cron/v3"
	"github.com/spf13/afero"
	"github.com/tdewolff/minify/v2"
	"github.com/thoas/go-funk"
	"github.com/vartanbeno/go-reddit/v2/reddit"

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
	imgProxy     *ImgProxy
	miniflux     *miniflux.Miniflux
	lastfm       *lastfm.LastFm
	Parser       *entry.Parser
	Config       *config.Config
	maze         *maze.Maze
	reddit       *reddit.Client
	XRay         *xray.XRay

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
		srcSync = fs.NewGitSync(conf.Source.Directory)
	}
	srcFs := fs.NewFS(conf.Source.Directory, srcSync)

	e := &Eagle{
		log:          log.S().Named("eagle"),
		httpClient:   httpClient,
		fs:           srcFs,
		wmClient:     webmention.New(httpClient),
		Config:       conf,
		allowedTypes: []mf2.Type{},
		syndication:  syndicator.NewManager(),
		Parser:       entry.NewParser(conf.Server.BaseURL),
		minifier:     initMinifier(),
		maze:         maze.NewMaze(httpClient),
		cron:         cron.New(),
	}

	for typ := range conf.Micropub.Sections {
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

	e.db, err = database.NewDatabase(&conf.PostgreSQL)
	if err != nil {
		return nil, err
	}

	e.markdown = newMarkdown(e, false)
	e.absoluteMarkdown = newMarkdown(e, true)

	if conf.Twitter != nil && conf.Syndications.Twitter {
		e.syndication.Add(syndicator.NewTwitter(conf.Twitter))
	}

	if conf.Reddit != nil {
		credentials := reddit.Credentials{
			ID:       conf.Reddit.App,
			Secret:   conf.Reddit.Secret,
			Username: conf.Reddit.User,
			Password: conf.Reddit.Password,
		}

		e.reddit, err = reddit.NewClient(credentials)
		if err != nil {
			return nil, err
		}

		if conf.Syndications.Reddit {
			e.syndication.Add(syndicator.NewReddit(e.reddit))
		}
	}

	if conf.Miniflux != nil {
		e.miniflux = miniflux.NewMiniflux(conf.Miniflux.Endpoint, conf.Miniflux.Key)
	}

	if conf.Lastfm != nil {
		e.lastfm = lastfm.NewLastFm(conf.Lastfm.Key, conf.Lastfm.User)
	}

	if conf.XRay != nil && conf.XRay.Endpoint != "" {
		e.XRay = &xray.XRay{
			HttpClient: httpClient,
			Log:        log.S().Named("xray"),
			Endpoint:   conf.XRay.Endpoint,
			UserAgent:  fmt.Sprintf("Eagle/0.0 (%s) XRay", conf.ID()),
		}

		if conf.XRay.Twitter {
			e.XRay.Twitter = conf.Twitter
		}

		if conf.XRay.Reddit {
			e.XRay.Reddit = e.reddit
		}
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

	err = e.initMinifluxCron()
	if err != nil {
		return nil, err
	}

	err = e.initScrobbleCron()
	if err != nil {
		return nil, err
	}

	_, err = e.cron.AddFunc("CRON_TZ=UTC 00 02 * * *", e.SyncStorage)
	if err != nil {
		return nil, err
	}

	e.cron.Start()
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
