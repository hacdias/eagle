package server

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/jwtauth/v5"
	"github.com/hacdias/eagle/v4/cache"
	"github.com/hacdias/eagle/v4/eagle"
	"github.com/hacdias/eagle/v4/fs"
	"github.com/hacdias/eagle/v4/indexer"
	"github.com/hacdias/eagle/v4/log"
	"github.com/hacdias/eagle/v4/media"
	"github.com/hacdias/eagle/v4/pkg/contenttype"
	"github.com/hacdias/eagle/v4/renderer"
	"github.com/hacdias/eagle/v4/services/bunny"
	"github.com/hacdias/eagle/v4/services/imgproxy"
	"github.com/hacdias/eagle/v4/services/postgres"
	"github.com/hacdias/eagle/v4/services/telegram"
	"github.com/hacdias/eagle/v4/services/twitter"
	"github.com/hacdias/eagle/v4/webmentions"
	"github.com/hacdias/indieauth/v3"
	"github.com/hashicorp/go-multierror"
	"github.com/robfig/cron/v3"
	"github.com/thoas/go-funk"

	"go.uber.org/zap"
)

type contextKey string

type httpServer struct {
	Name string
	*http.Server
}

type Server struct {
	n eagle.Notifier
	c *eagle.Config
	i *indexer.Indexer

	log          *zap.SugaredLogger
	iac          *indieauth.Client
	ias          *indieauth.Server
	jwtAuth      *jwtauth.JWTAuth
	onionAddress string
	serversLock  sync.Mutex
	servers      []*httpServer
	actions      map[string]func() error
	cron         *cron.Cron
	redirects    map[string]string

	fs          *fs.FS
	media       *media.Media
	cache       *cache.Cache
	webmentions *webmentions.Webmentions
	syndicator  *eagle.Manager
	renderer    *renderer.Renderer
	parser      *eagle.Parser

	preSaveHooks  []eagle.EntryHook
	postSaveHooks []eagle.EntryHook
}

func NewServer(c *eagle.Config) (*Server, error) {
	clientID := c.Server.BaseURL + "/"
	redirectURL := c.Server.BaseURL + "/login/callback"
	secret := base64.StdEncoding.EncodeToString([]byte(c.Server.TokensSecret))

	var notifier eagle.Notifier
	if c.Notifications.Telegram != nil {
		notifications, err := telegram.NewTelegram(c.Notifications.Telegram)
		if err != nil {
			return nil, err
		}
		notifier = notifications
	} else {
		notifier = log.NewLogNotifier()
	}

	var srcSync fs.FSSync
	if c.Development {
		srcSync = fs.NewPlaceboSync()
	} else {
		srcSync = fs.NewGitSync(c.Source.Directory)
	}
	fs := fs.NewFS(c.Source.Directory, c.Server.BaseURL, srcSync)

	var (
		m           *media.Media
		mBaseURL    string
		storage     media.Storage
		transformer media.Transformer
	)

	if c.BunnyCDN != nil {
		storage = bunny.NewBunny(c.BunnyCDN)
	}

	if c.ImgProxy != nil {
		transformer = imgproxy.NewImgProxy(c.ImgProxy)
	}

	if storage != nil {
		m = media.NewMedia(storage, transformer)
		mBaseURL = m.BaseURL()
	}

	renderer, err := renderer.NewRenderer(c, fs, mBaseURL)
	if err != nil {
		return nil, err
	}

	cache, err := cache.NewCache()
	if err != nil {
		return nil, err
	}

	backend, err := postgres.NewPostgres(&c.PostgreSQL)
	if err != nil {
		return nil, err
	}

	s := &Server{
		n:             notifier,
		c:             c,
		i:             indexer.NewIndexer(fs, backend),
		log:           log.S().Named("server"),
		iac:           indieauth.NewClient(clientID, redirectURL, &http.Client{Timeout: time.Second * 30}),
		ias:           indieauth.NewServer(false, &http.Client{Timeout: time.Second * 30}),
		jwtAuth:       jwtauth.New("HS256", []byte(secret), nil),
		servers:       []*httpServer{},
		actions:       map[string]func() error{},
		cron:          cron.New(),
		redirects:     map[string]string{},
		fs:            fs,
		media:         m,
		cache:         cache,
		webmentions:   webmentions.NewWebmentions(fs, notifier, renderer, m),
		syndicator:    eagle.NewManager(),
		renderer:      renderer,
		parser:        eagle.NewParser(c.Server.BaseURL),
		preSaveHooks:  []eagle.EntryHook{},
		postSaveHooks: []eagle.EntryHook{},
	}

	if c.Twitter != nil && c.Syndications.Twitter {
		s.syndicator.Add(twitter.NewTwitter(c.Twitter))
	}

	// var (
	// 	redditClient *reddit.Client
	// )

	// if e.Config.Reddit != nil {
	// 	credentials := reddit.Credentials{
	// 		ID:       e.Config.Reddit.App,
	// 		Secret:   e.Config.Reddit.Secret,
	// 		Username: e.Config.Reddit.User,
	// 		Password: e.Config.Reddit.Password,
	// 	}

	// 	redditClient, err = reddit.NewClient(credentials)
	// 	if err != nil {
	// 		return nil, err
	// 	}

	// 	if e.Config.Syndications.Reddit {
	// 		s.syndicator.Add(syndicator.NewReddit(redditClient))
	// 	}
	// }

	// s.PreSaveHooks = append(
	// 	s.PreSaveHooks,
	// 	&hooks.IgnoreListing{Hook: hooks.AllowedType(c.Micropub.AllowedTypes())},
	// 	&hooks.IgnoreListing{Hook: &hooks.DescriptionGenerator{}},
	// 	&hooks.IgnoreListing{Hook: hooks.SectionDeducer(c.Micropub.Sections)},
	// )

	// if e.Config.XRay != nil && e.Config.XRay.Endpoint != "" {
	// 	options := &xray.XRayOptions{
	// 		Log:       log.S().Named("xray"),
	// 		Endpoint:  e.Config.XRay.Endpoint,
	// 		UserAgent: fmt.Sprintf("Eagle/0.0 (%s) XRay", e.Config.ID()),
	// 	}

	// 	if e.Config.XRay.Twitter && e.Config.Twitter != nil {
	// 		options.Twitter = &xray.Twitter{
	// 			Key:         e.Config.Twitter.Key,
	// 			Secret:      e.Config.Twitter.Secret,
	// 			Token:       e.Config.Twitter.Token,
	// 			TokenSecret: e.Config.Twitter.TokenSecret,
	// 		}
	// 	}

	// 	if e.Config.XRay.Reddit && e.Config.Reddit != nil {
	// 		options.Reddit = redditClient
	// 	}

	// 	xray := xray.NewXRay(options)

	// 	s.PostSaveHooks = append(s.PostSaveHooks, &hooks.IgnoreListing{Hook: &hooks.ContextXRay{
	// 		XRay:  xray,
	// 		Eagle: s.Eagle,
	// 	}})
	// }

	// s.PostSaveHooks = append(
	// 	s.PostSaveHooks,
	// 	&hooks.IgnoreListing{Hook: &hooks.PhotosProcessor{
	// 		Eagle: e,
	// 	}},
	// 	&hooks.IgnoreListing{Hook: &hooks.LocationFetcher{
	// 		Language: e.Config.Site.Language,
	// 		Eagle:    e,
	// 		Maze: maze.NewMaze(&http.Client{
	// 			Timeout: 1 * time.Minute,
	// 		}),
	// 	}},
	// 	&hooks.IgnoreListing{Hook: s.Webmentions}, // if not disable sending
	// )

	// readsSummaryUpdater := &hooks.ReadsSummaryUpdater{
	// 	Eagle:    s.Eagle,
	// 	Provider: s.Eagle.DB.(*database.Postgres), // wip: dont do this
	// }
	// s.PostSaveHooks = append(s.PostSaveHooks, &hooks.IgnoreListing{Hook: readsSummaryUpdater})
	// err = s.RegisterAction("Update Reads Summary", readsSummaryUpdater.UpdateReadsSummary)
	// if err != nil {
	// 	return nil, err
	// }

	// watchesSummaryUpdater := &hooks.WatchesSummaryUpdater{
	// 	Eagle:    s.Eagle,
	// 	Provider: s.Eagle.DB.(*database.Postgres), // wip: dont do this
	// }
	// s.PostSaveHooks = append(s.PostSaveHooks, &hooks.IgnoreListing{Hook: watchesSummaryUpdater})
	// err = s.RegisterAction("Update Watches Summary", watchesSummaryUpdater.UpdateWatchesSummary)
	// if err != nil {
	// 	return nil, err
	// }

	// if e.Config.Miniflux != nil {
	// 	mf := blogroll.MinifluxBlogrollUpdater{
	// 		Eagle:  e,
	// 		Client: miniflux.NewMiniflux(e.Config.Miniflux.Endpoint, e.Config.Miniflux.Key),
	// 	}

	// 	err = s.RegisterCron("00 00 * * *", "Miniflux Blogroll", mf.UpdateMinifluxBlogroll)
	// 	if err != nil {
	// 		return nil, err
	// 	}

	// 	err = s.RegisterAction("Miniflux Blogroll", mf.UpdateMinifluxBlogroll)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// }

	// if e.Config.Lastfm != nil {
	// 	lastfm := lastfm.NewLastFm(e.Config.Lastfm.Key, e.Config.Lastfm.User, e)
	// 	err = s.RegisterCron("00 05 * * *", "LastFm Daily", lastfm.DailyJob)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// }

	// err = s.RegisterCron("00 02 * * *", "Sync Storage", func() error {
	// 	s.SyncStorage()
	// 	return nil
	// })
	// if err != nil {
	// 	return nil, err
	// }

	err = s.initRedirects()
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Server) RegisterAction(name string, action func() error) error {
	if _, ok := s.actions[name]; ok {
		return errors.New("action already registered")
	}

	s.actions[name] = action
	return nil
}

func (s *Server) RegisterCron(schedule, name string, job func() error) error {
	_, err := s.cron.AddFunc(schedule, func() {
		err := job()
		if err != nil {
			s.n.Error(fmt.Errorf("%s cron job: %w", name, err))
		}
	})
	return err
}

func (s *Server) Start() error {
	go s.indexAll()
	s.cron.Start()

	errCh := make(chan error)
	router := s.makeRouter()

	// Start server(s)
	err := s.startRegularServer(errCh, router)
	if err != nil {
		return err
	}

	if s.c.Server.Tor != nil {
		err = s.startTor(errCh, router)
		if err != nil {
			err = fmt.Errorf("onion service failed to start: %w", err)
			s.log.Error(err)
		}
	}

	// Collect errors when the server stops
	var errs *multierror.Error
	for i := 0; i < len(s.servers); i++ {
		errs = multierror.Append(errs, <-errCh)
	}
	return errs.ErrorOrNil()
}

func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var errs *multierror.Error
	for _, srv := range s.servers {
		s.log.Infof("shutting down %s", srv.Name)
		errs = multierror.Append(errs, srv.Shutdown(ctx))
	}

	<-s.cron.Stop().Done()
	errs = multierror.Append(errs, s.i.Close())
	return errs.ErrorOrNil()
}

func (s *Server) indexAll() {
	entries, err := s.fs.GetEntries(false)
	if err != nil {
		s.n.Error(err)
		return
	}

	start := time.Now()
	err = s.i.Add(entries...)
	if err != nil {
		s.n.Error(err)
	}
	s.log.Infof("database update took %dms", time.Since(start).Milliseconds())
}

func (s *Server) registerServer(srv *http.Server, name string) {
	s.serversLock.Lock()
	defer s.serversLock.Unlock()

	s.servers = append(s.servers, &httpServer{
		Server: srv,
		Name:   name,
	})
}

func (s *Server) startRegularServer(errCh chan error, h http.Handler) error {
	addr := ":" + strconv.Itoa(s.c.Server.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	srv := &http.Server{Handler: h}
	s.registerServer(srv, "public")

	go func() {
		s.log.Infof("listening on %s", ln.Addr().String())
		errCh <- srv.Serve(ln)
	}()

	return nil
}

func (s *Server) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rvr := recover(); rvr != nil && rvr != http.ErrAbortHandler {
				err := fmt.Errorf("panic while serving: %v: %s", rvr, string(debug.Stack()))
				s.n.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// borrowed from chi + redirection.
func cleanPath(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.RouteContext(r.Context())

		routePath := rctx.RoutePath
		if routePath == "" {
			if r.URL.RawPath != "" {
				routePath = r.URL.RawPath
			} else {
				routePath = r.URL.Path
			}
			routePath = path.Clean(routePath)
		}

		if r.URL.Path != routePath {
			http.Redirect(w, r, routePath, http.StatusTemporaryRedirect)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) serveJSON(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", contenttype.JSONUTF8)
	w.WriteHeader(code)
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		s.n.Error(fmt.Errorf("serving html: %w", err))
	}
}

func (s *Server) serveErrorJSON(w http.ResponseWriter, code int, err, errDescription string) {
	s.serveJSON(w, code, map[string]interface{}{
		"error":             err,
		"error_description": errDescription,
	})
}

func (s *Server) serveHTMLWithStatus(w http.ResponseWriter, r *http.Request, data *renderer.RenderData, tpls []string, code int) {
	if data.Entry.ID == "" {
		data.Entry.ID = r.URL.Path
	}

	data.TorUsed = s.isUsingTor(r)
	data.OnionAddress = s.onionAddress
	data.IsLoggedIn = s.getUser(r) != ""
	data.IsAdmin = s.isAdmin(r)
	data.User = s.getUser(r)

	setCacheHTML(w)
	w.Header().Set("Content-Type", contenttype.HTMLUTF8)
	w.WriteHeader(code)

	var (
		buf bytes.Buffer
		cw  io.Writer
	)

	if code == http.StatusOK && s.isCacheable(r) {
		cw = io.MultiWriter(w, &buf)
	} else {
		cw = w
	}

	err := s.renderer.Render(cw, data, tpls)
	if err != nil {
		s.n.Error(fmt.Errorf("serving html: %w", err))
	} else {
		data := buf.Bytes()
		if len(data) > 0 {
			s.saveCache(r, data)
		}
	}
}

func (s *Server) serveHTML(w http.ResponseWriter, r *http.Request, data *renderer.RenderData, tpls []string) {
	s.serveHTMLWithStatus(w, r, data, tpls, http.StatusOK)
}

func (s *Server) serveErrorHTML(w http.ResponseWriter, r *http.Request, code int, err error) {
	if err != nil {
		s.log.Error(err)
	}

	w.Header().Del("Cache-Control")

	data := map[string]interface{}{
		"Code": code,
	}

	if err != nil {
		data["Error"] = err.Error()
	}

	rd := &renderer.RenderData{
		Entry: &eagle.Entry{
			FrontMatter: eagle.FrontMatter{
				Title: fmt.Sprintf("%d %s", code, http.StatusText(code)),
			},
		},
		NoIndex: true,
		Data:    data,
	}

	s.serveHTMLWithStatus(w, r, rd, []string{renderer.TemplateError}, code)
}

func (s *Server) SyncStorage() {
	changedFiles, err := s.fs.Sync()
	if err != nil {
		s.n.Error(fmt.Errorf("sync storage: %w", err))
		return
	}

	if len(changedFiles) == 0 {
		return
	}

	ids := []string{}

	for _, file := range changedFiles {
		if !strings.HasPrefix(file, fs.ContentDirectory) {
			continue
		}

		id := strings.TrimPrefix(file, fs.ContentDirectory)
		id = filepath.Dir(id)
		ids = append(ids, id)
	}

	// NOTE: we do not reload the templates and assets because
	// doing so is not concurrent-safe.
	ids = funk.UniqString(ids)
	entries := []*eagle.Entry{}

	for _, id := range ids {
		entry, err := s.fs.GetEntry(id)
		if os.IsNotExist(err) {
			s.i.Remove(id)
			continue
		} else if err != nil {
			s.n.Error(fmt.Errorf("cannot open entry to update %s: %w", id, err))
			continue
		}
		entries = append(entries, entry)

		if s.cache != nil {
			s.cache.Delete(entry)
		}
	}

	err = s.i.Add(entries...)
	if err != nil {
		s.n.Error(fmt.Errorf("sync failed: %w", err))
	}
}
