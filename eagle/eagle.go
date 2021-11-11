package eagle

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"sync"
	"time"

	"github.com/hacdias/eagle/v2/config"
	"github.com/hacdias/eagle/v2/entry"
	"github.com/hacdias/eagle/v2/fs"
	"github.com/hacdias/eagle/v2/logging"
	"github.com/hacdias/eagle/v2/notifier"
	"github.com/hacdias/eagle/v2/pkg/mf2"
	"github.com/hacdias/eagle/v2/syndicator"
	"github.com/jackc/pgx/v4"
	"github.com/yuin/goldmark"
	"go.uber.org/zap"
	"willnorris.com/go/webmention"
)

const (
	AssetsDirectory  string = "assets"
	ContentDirectory string = "content2" // TODO(v2): change this back to content
	StaticDirectory  string = "static"
)

type Eagle struct {
	log          *zap.SugaredLogger
	httpClient   *http.Client
	wmClient     *webmention.Client
	fs           *fs.FS
	conn         *pgx.Conn
	syndication  *syndicator.Manager
	allowedTypes []mf2.Type
	notifier     notifier.Notifier

	templates        map[string]*template.Template
	markdown         goldmark.Markdown
	absoluteMarkdown goldmark.Markdown

	// Mutexes to lock the updates to entries and sidecars.
	// Only for writes and not for reads. Hope this won't
	// become a problem with traffic and simultaneous
	// reads-writes from files. Adding a mutex for all reads
	// would probably make it much slower though.
	entriesMu  sync.Mutex
	sidecarsMu sync.Mutex

	// TODO: THINGS TO CLEAN
	Parser   *entry.Parser
	Config   *config.Config
	media    *Media
	Miniflux *Miniflux
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
		log:          logging.S().Named("eagle"),
		httpClient:   httpClient,
		fs:           srcFs,
		wmClient:     webmention.New(httpClient),
		Config:       conf,
		allowedTypes: []mf2.Type{},
		syndication:  syndicator.NewManager(),
		Parser:       entry.NewParser(conf.Site.BaseURL),
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
		e.notifier = notifications
	} else {
		e.notifier = notifier.NewLogNotifier()
	}

	err := e.setupPostgres()
	if err != nil {
		return nil, err
	}

	e.markdown = newMarkdown(false, conf.Site.BaseURL)
	e.absoluteMarkdown = newMarkdown(true, conf.Site.BaseURL)

	if conf.Twitter != nil {
		e.syndication.Add(syndicator.NewTwitter(conf.Twitter))
	}

	if conf.Miniflux != nil {
		e.Miniflux = &Miniflux{Miniflux: conf.Miniflux}
	}

	err = e.updateTemplates()
	if err != nil {
		return nil, err
	}

	return e, nil
}

func (e *Eagle) Close() {
	if e.conn != nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		_ = e.conn.Close(ctx)
	}
}

func (e *Eagle) userAgent(comment string) string {
	return fmt.Sprintf("Eagle/0.0 %s", comment)
}
