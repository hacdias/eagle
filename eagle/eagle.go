package eagle

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/hacdias/eagle/v2/config"
	"github.com/hacdias/eagle/v2/contenttype"
	"github.com/hacdias/eagle/v2/entry"
	"github.com/hacdias/eagle/v2/entry/mf2"
	"github.com/hacdias/eagle/v2/fs"
	"github.com/hacdias/eagle/v2/log"
	"github.com/hacdias/eagle/v2/notifier"
	"github.com/hacdias/eagle/v2/syndicator"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/tdewolff/minify/v2"
	mCss "github.com/tdewolff/minify/v2/css"
	mHtml "github.com/tdewolff/minify/v2/html"
	mJs "github.com/tdewolff/minify/v2/js"
	mJson "github.com/tdewolff/minify/v2/json"
	mXml "github.com/tdewolff/minify/v2/xml"

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
	notifier.Notifier

	log          *zap.SugaredLogger
	httpClient   *http.Client
	wmClient     *webmention.Client
	fs           *fs.FS
	conn         *pgxpool.Pool
	syndication  *syndicator.Manager
	allowedTypes []mf2.Type

	templates        map[string]*template.Template
	markdown         goldmark.Markdown
	absoluteMarkdown goldmark.Markdown
	minifier         *minify.M

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
		log:          log.S().Named("eagle"),
		httpClient:   httpClient,
		fs:           srcFs,
		wmClient:     webmention.New(httpClient),
		Config:       conf,
		allowedTypes: []mf2.Type{},
		syndication:  syndicator.NewManager(),
		Parser:       entry.NewParser(conf.Site.BaseURL),
		minifier:     getMinifier(),
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

func (e *Eagle) GetSyndicators() []*syndicator.Config {
	return e.syndication.Config()
}

func (e *Eagle) GetRedirects() (map[string]string, error) {
	redirects := map[string]string{}

	data, err := e.fs.ReadFile("redirects")
	if err != nil {
		return nil, err
	}

	strs := strings.Split(string(data), "\n")

	for _, str := range strs {
		if strings.TrimSpace(str) == "" {
			continue
		}

		parts := strings.Split(str, " ")
		if len(parts) != 2 {
			e.log.Warnf("found invalid redirect entry: %s", str)
		}

		redirects[parts[0]] = parts[1]
	}

	return redirects, nil
}

func (e *Eagle) Close() {
	if e.conn != nil {
		e.conn.Close()
	}
}

func (e *Eagle) userAgent(comment string) string {
	return fmt.Sprintf("Eagle/0.0 %s", comment)
}

func getMinifier() *minify.M {
	m := minify.New()
	m.AddFunc(contenttype.HTML, mHtml.Minify)
	m.AddFunc(contenttype.CSS, mCss.Minify)
	m.AddFunc(contenttype.XML, mXml.Minify)
	m.AddFunc(contenttype.JS, mJs.Minify)
	m.AddFunc(contenttype.RSS, mXml.Minify)
	m.AddFunc(contenttype.ATOM, mXml.Minify)
	m.AddFunc(contenttype.JSONFeed, mJson.Minify)
	m.AddFunc(contenttype.AS, mJson.Minify)
	return m
}
