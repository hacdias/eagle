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
	"sort"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/jwtauth/v5"
	"github.com/hacdias/eagle/activitypub"
	"github.com/hacdias/eagle/cache"
	"github.com/hacdias/eagle/eagle"
	"github.com/hacdias/eagle/fs"
	"github.com/hacdias/eagle/hooks"
	"github.com/hacdias/eagle/indexer"
	"github.com/hacdias/eagle/log"
	"github.com/hacdias/eagle/media"
	"github.com/hacdias/eagle/pkg/contenttype"
	"github.com/hacdias/eagle/renderer"
	"github.com/hacdias/eagle/services/bunny"
	"github.com/hacdias/eagle/services/imgproxy"
	"github.com/hacdias/eagle/services/lastfm"
	"github.com/hacdias/eagle/services/miniflux"
	"github.com/hacdias/eagle/services/postgres"
	"github.com/hacdias/eagle/services/reddit"
	"github.com/hacdias/eagle/services/telegram"
	"github.com/hacdias/eagle/services/twitter"
	"github.com/hacdias/eagle/webmentions"
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
	archetypes   map[string]*template.Template
	webfinger    *webfinger

	fs          *fs.FS
	media       *media.Media
	cache       *cache.Cache
	webmentions *webmentions.Webmentions
	syndicator  *eagle.Manager
	renderer    *renderer.Renderer
	parser      *eagle.Parser
	ap          *activitypub.ActivityPub

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

	var srcSync fs.Sync
	if c.Development {
		srcSync = &fs.NopSync{}
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

	postgres, err := postgres.NewPostgres(&c.PostgreSQL)
	if err != nil {
		return nil, err
	}

	s := &Server{
		n:             notifier,
		c:             c,
		i:             indexer.NewIndexer(fs, postgres),
		log:           log.S().Named("server"),
		iac:           indieauth.NewClient(clientID, redirectURL, &http.Client{Timeout: time.Second * 30}),
		ias:           indieauth.NewServer(false, &http.Client{Timeout: time.Second * 30}),
		jwtAuth:       jwtauth.New("HS256", []byte(secret), nil),
		servers:       []*httpServer{},
		cron:          cron.New(),
		redirects:     map[string]string{},
		archetypes:    map[string]*template.Template{},
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

	fs.AfterSaveHook = s.afterSaveHook

	s.AppendPreSaveHook(
		hooks.TypeChecker(c.Micropub.AllowedTypes()),
		hooks.NewDescriptionGenerator(s.fs),
		hooks.SectionDeducer(c.Micropub.Sections),
		hooks.TagsSanitizer{},
	)

	s.initActions()

	if c.Twitter != nil && c.Syndications.Twitter {
		s.syndicator.Add(twitter.NewTwitter(c.Twitter))
	}

	if c.Reddit != nil && c.Syndications.Reddit {
		reddit, err := reddit.NewReddit(c.Reddit)
		if err != nil {
			return nil, err
		}
		s.syndicator.Add(reddit)
	}

	if c.XRay != nil && c.XRay.Endpoint != "" {
		xray, err := hooks.NewContextFetcher(c, s.fs, s.media)
		if err != nil {
			return nil, err
		}
		s.AppendPostSaveHook(xray)
	}

	if s.media != nil {
		s.AppendPostSaveHook(hooks.NewPhotosUploader(s.fs, s.media))
	}

	s.AppendPostSaveHook(hooks.NewLocationFetcher(s.fs, c.Site.Language))

	if !c.Webmentions.DisableSending {
		s.AppendPostSaveHook(s.webmentions)
	}

	var errs *multierror.Error
	readsSummaryUpdater := hooks.NewReadsSummaryUpdater(s.fs, postgres)
	watchesSummaryUpdater := hooks.NewWatchesSummaryUpdater(s.fs, postgres)
	s.AppendPostSaveHook(readsSummaryUpdater, watchesSummaryUpdater)

	errs = multierror.Append(
		errs,
		s.RegisterAction("Update Reads Summary", readsSummaryUpdater.UpdateReadsSummary),
		s.RegisterAction("Update Watches Summary", watchesSummaryUpdater.UpdateWatchesSummary),
	)

	if c.Miniflux != nil {
		mf := miniflux.NewBlogrollUpdater(c.Miniflux, s.fs)

		errs = multierror.Append(
			errs,
			s.RegisterCron("00 00 * * *", "Miniflux Blogroll", mf.UpdateBlogroll),
			s.RegisterAction("Update Miniflux Blogroll", mf.UpdateBlogroll),
		)
	}

	if c.Lastfm != nil {
		lastfm := lastfm.NewLastFm(c.Lastfm.Key, c.Lastfm.User, s.fs, s.media)
		errs = multierror.Append(errs, s.RegisterCron("00 05 * * *", "LastFm Daily", lastfm.DailyJob))
	}

	if c.Server.ActivityPub != nil {
		s.ap, err = activitypub.NewActivityPub(c, s.renderer, s.fs, s.n, s.webmentions, s.media)
		if err != nil {
			return nil, err
		}
		s.AppendPostSaveHook(s.ap)
	}

	s.initWebfinger()

	errs = multierror.Append(errs, s.RegisterCron("00 02 * * *", "Sync Storage", func() error {
		s.syncStorage()
		return nil
	}), s.loadRedirects(), s.loadArchetypes())

	err = errs.ErrorOrNil()
	return s, err
}

func (s *Server) AppendPreSaveHook(hooks ...eagle.EntryHook) {
	s.preSaveHooks = append(s.preSaveHooks, hooks...)
}

func (s *Server) AppendPostSaveHook(hooks ...eagle.EntryHook) {
	s.postSaveHooks = append(s.postSaveHooks, hooks...)
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
	go func() {
		s.indexAll()
		if s.ap != nil {
			s.ap.SendProfileUpdate()
		}
	}()

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

func (s *Server) initActions() {
	s.actions = map[string]func() error{
		"Clear Cache": func() error {
			s.cache.Clear()
			return nil
		},
		"Sync Storage": func() error {
			go s.syncStorage()
			return nil
		},
		"Reload Templates": func() error {
			return s.renderer.LoadTemplates()
		},
		"Reload Assets": func() error {
			return s.renderer.LoadAssets()
		},
		"Reload Redirects": func() error {
			return s.loadRedirects()
		},
		"Reload Archetypes": func() error {
			return s.loadArchetypes()
		},
	}
}

func (s *Server) getActions() []string {
	actions := []string{}
	for action := range s.actions {
		actions = append(actions, action)
	}
	sort.Strings(actions)
	return actions
}

func (s *Server) loadRedirects() error {
	redirects, err := s.fs.LoadRedirects(true)
	if err != nil {
		return err
	}
	s.redirects = redirects
	return nil
}

func (s *Server) loadArchetypes() error {
	archetypes, err := s.fs.LoadArchetypes()
	if err != nil {
		return err
	}
	s.archetypes = archetypes
	return nil
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
	s.serveJSON(w, code, map[string]string{
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
		s.n.Error(fmt.Errorf("serving html for %s: %w", r.URL.Path, err))
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

func (s *Server) serveActivity(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", contenttype.ASUTF8)
	w.WriteHeader(code)
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		s.n.Error(fmt.Errorf("serving html: %w", err))
	}
}

func (s *Server) syncStorage() {
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
	// doing so is not concurrent-safe. Alternatively, there is
	// an action in the dashboard to reload assets and templates
	// on-demand.
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

func (s *Server) afterSaveHook(updated, deleted []*eagle.Entry) {
	for _, e := range updated {
		_ = s.i.Add(e)
		s.cache.Delete(e)
	}

	for _, e := range deleted {
		s.i.Remove(e.ID)
		s.cache.Delete(e)
	}

	if len(updated) == len(deleted) {
		// Then it's likely a rename.
		_ = s.loadRedirects()
	}
}

func isActivityPub(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept"), contenttype.AS)
}
