package eagle

import (
	"fmt"
	"html/template"
	"net/http"
	"sync"
	"time"

	"github.com/hacdias/eagle/v2/config"
	"github.com/hacdias/eagle/v2/logging"
	"github.com/hacdias/eagle/v2/pkg/mf2"
	"github.com/meilisearch/meilisearch-go"
	"github.com/spf13/afero"
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
	log        *zap.SugaredLogger
	ms         meilisearch.ClientInterface
	httpClient *http.Client

	// Maybe embed this one and ovveride WriteFile instead of persist?
	SrcFs  *afero.Afero
	srcGit *gitRepo

	// Mutexes to lock the updates to entries and sidecars.
	// Only for writes and not for reads. Hope this won't
	// become a problem with traffic and simultaneous
	// reads-writes from files. Adding a mutex for all reads
	// would probably make it much slower though.
	entriesMu  sync.Mutex
	sidecarsMu sync.Mutex

	webmentionsClient *webmention.Client
	allowedTypes      []mf2.Type
	templates         map[string]*template.Template
	markdown          goldmark.Markdown
	absoluteMarkdown  goldmark.Markdown

	Notifications
	Config      *config.Config
	PublicDirCh chan string

	// Optional services
	media *Media

	Miniflux *Miniflux
	Twitter  *Twitter
}

func NewEagle(conf *config.Config) (*Eagle, error) {
	httpClient := &http.Client{
		Timeout: time.Minute * 2,
	}

	e := &Eagle{
		log:               logging.S().Named("eagle"),
		httpClient:        httpClient,
		SrcFs:             makeAfero(conf.SourceDirectory),
		srcGit:            &gitRepo{conf.SourceDirectory},
		webmentionsClient: webmention.New(httpClient),
		Config:            conf,
		PublicDirCh:       make(chan string, 2),
		allowedTypes:      []mf2.Type{},
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
		notifications, err := newTgNotifications(conf.Telegram)
		if err != nil {
			return nil, err
		}
		e.Notifications = notifications
	} else {
		e.Notifications = newLogNotifications()
	}

	err := e.setupMeiliSearch()
	if err != nil {
		return nil, err
	}

	e.markdown = newMarkdown(false, conf.Site.BaseURL)
	e.absoluteMarkdown = newMarkdown(true, conf.Site.BaseURL)

	if conf.Twitter != nil {
		e.Twitter = NewTwitter(conf.Twitter)
	}

	if conf.Miniflux != nil {
		e.Miniflux = &Miniflux{Miniflux: conf.Miniflux}
	}

	return e, nil
}

func (e *Eagle) userAgent(comment string) string {
	return fmt.Sprintf("Eagle/0.0 %s", comment)
}

func makeAfero(path string) *afero.Afero {
	return &afero.Afero{
		Fs: afero.NewBasePathFs(afero.NewOsFs(), path),
	}
}
