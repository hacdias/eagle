package server

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
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
	"time"

	"github.com/go-chi/jwtauth/v5"
	"github.com/hacdias/eagle/core"
	"github.com/hacdias/eagle/core/helpers"
	"github.com/hacdias/eagle/log"
	"github.com/hacdias/eagle/services/bunny"
	"github.com/hacdias/eagle/services/imgproxy"
	"github.com/hacdias/eagle/services/linkding"
	"github.com/hacdias/eagle/services/media"
	"github.com/hacdias/eagle/services/miniflux"
	"github.com/hacdias/eagle/services/postgres"
	"github.com/hacdias/eagle/services/telegram"
	"github.com/hacdias/eagle/services/webmentions"
	"github.com/hacdias/indieauth/v3"
	"github.com/hacdias/maze"
	"github.com/hashicorp/go-multierror"
	"github.com/robfig/cron/v3"
	"github.com/samber/lo"

	"go.uber.org/zap"
)

type contextKey string

type Server struct {
	n core.Notifier
	c *core.Config
	i *core.Indexer

	log        *zap.SugaredLogger
	ias        *indieauth.Server
	jwtAuth    *jwtauth.JWTAuth
	actions    map[string]func() error
	cron       *cron.Cron
	redirects  map[string]string
	archetypes map[string]core.Archetype
	webFinger  *core.WebFinger

	server *http.Server

	fs          *core.FS
	hugo        *core.Hugo
	media       *media.Media
	webmentions *webmentions.Webmentions
	parser      *core.Parser
	maze        *maze.Maze
	guestbook   guestbookStorage

	locationFetcher *helpers.LocationFetcher
	contextFetcher  *helpers.ContextFetcher

	staticFsLock sync.RWMutex
	staticFs     *staticFs
}

func NewServer(c *core.Config) (*Server, error) {
	secret := base64.StdEncoding.EncodeToString([]byte(c.TokensSecret))

	var notifier core.Notifier
	if c.Notifications.Telegram != nil {
		notifications, err := telegram.NewTelegram(c.Notifications.Telegram)
		if err != nil {
			return nil, err
		}
		notifier = notifications
	} else {
		notifier = log.NewLogNotifier()
	}

	var srcSync core.Sync
	if c.Development {
		srcSync = &core.NopSync{}
	} else {
		srcSync = core.NewGitSync(c.SourceDirectory)
	}
	fs := core.NewFS(c.SourceDirectory, c.BaseURL, srcSync)
	hugo := core.NewHugo(c.SourceDirectory, c.PublicDirectory, c.BaseURL)

	var (
		m           *media.Media
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
	}

	postgres, err := postgres.NewPostgres(&c.PostgreSQL)
	if err != nil {
		return nil, err
	}

	s := &Server{
		n:           notifier,
		c:           c,
		i:           core.NewIndexer(fs, postgres),
		log:         log.S().Named("server"),
		ias:         indieauth.NewServer(false, &http.Client{Timeout: time.Second * 30}),
		jwtAuth:     jwtauth.New("HS256", []byte(secret), nil),
		cron:        cron.New(),
		redirects:   map[string]string{},
		archetypes:  core.DefaultArchetypes,
		fs:          fs,
		hugo:        hugo,
		media:       m,
		webmentions: webmentions.NewWebmentions(fs, hugo, notifier),
		parser:      core.NewParser(c.BaseURL),
		maze: maze.NewMaze(&http.Client{
			Timeout: time.Minute,
		}),
		guestbook: postgres,

		locationFetcher: helpers.NewLocationFetcher(fs, c.Language),
	}

	s.hugo.BuildHook = s.buildHook
	s.initActions()

	if c.XRay != nil && c.XRay.Endpoint != "" {
		xray, err := helpers.NewContextFetcher(c, s.fs)
		if err != nil {
			return nil, err
		}
		s.contextFetcher = xray
	}

	var errs *multierror.Error

	if c.Miniflux != nil {
		mf := miniflux.NewBlogrollUpdater(c.Miniflux, s.fs)

		errs = multierror.Append(
			errs,
			s.RegisterCron("00 00 * * *", "Miniflux Blogroll", mf.UpdateBlogroll),
			s.RegisterAction("Update Miniflux Blogroll", mf.UpdateBlogroll),
		)
	}

	if c.Linkding != nil {
		ld := linkding.NewBookmarksUpdater(c.Linkding, s.fs)

		errs = multierror.Append(
			errs,
			s.RegisterCron("00 00 * * *", "Linkding Bookmarks", ld.UpdateBookmarks),
			s.RegisterAction("Update Linkding Bookmarks", ld.UpdateBookmarks),
		)
	}

	s.initWebFinger()

	errs = multierror.Append(errs, s.RegisterCron("00 02 * * *", "Sync Storage", func() error {
		s.syncStorage()
		return nil
	}), s.loadRedirects())

	err = errs.ErrorOrNil()
	return s, err
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
	}()

	// Make sure we have a built version to serve
	should, err := s.hugo.ShouldBuild()
	if err != nil {
		return err
	}

	if should {
		err = s.hugo.Build(false)
		if err != nil {
			return err
		}
	}

	s.cron.Start()

	// Start server
	addr := ":" + strconv.Itoa(s.c.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	errCh := make(chan error)
	s.server = &http.Server{Handler: s.makeRouter()}
	go func() {
		s.log.Infof("listening on %s", ln.Addr().String())
		errCh <- s.server.Serve(ln)
	}()

	return <-errCh
}

func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	<-s.cron.Stop().Done()

	var errs *multierror.Error
	errs = multierror.Append(errs, s.server.Shutdown(ctx))
	errs = multierror.Append(errs, s.i.Close())
	return errs.ErrorOrNil()
}

func (s *Server) initActions() {
	s.actions = map[string]func() error{
		"Build Website": func() error {
			return s.hugo.Build(false)
		},
		"Build Website (Clean)": func() error {
			return s.hugo.Build(true)
		},
		"Sync Storage": func() error {
			go s.syncStorage()
			return nil
		},
		"Reload Redirects": func() error {
			return s.loadRedirects()
		},
		"Reset Index": func() error {
			s.i.ClearEntries()
			s.indexAll()
			return nil
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

func (s *Server) withRecoverer(next http.Handler) http.Handler {
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

func withCleanPath(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := path.Clean(r.URL.Path)
		if path != "/" && strings.HasSuffix(r.URL.Path, "/") {
			path += "/"
		}

		if r.URL.Path != path {
			http.Redirect(w, r, path, http.StatusTemporaryRedirect)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) withSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		next.ServeHTTP(w, r)
	})
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
		if !strings.HasPrefix(file, core.ContentDirectory) {
			continue
		}

		id := strings.TrimPrefix(file, core.ContentDirectory)
		id = filepath.Dir(id)
		ids = append(ids, id)
	}

	ids = lo.Uniq(ids)
	entries := core.Entries{}
	buildClean := false

	for _, id := range ids {
		entry, err := s.fs.GetEntry(id)
		if os.IsNotExist(err) {
			s.i.Remove(id)
			buildClean = true
			continue
		} else if err != nil {
			s.n.Error(fmt.Errorf("cannot open entry to update %s: %w", id, err))
			continue
		}
		entries = append(entries, entry)
	}

	err = s.i.Add(entries...)
	if err != nil {
		s.n.Error(fmt.Errorf("sync failed: %w", err))
	}

	s.buildNotify(buildClean)
}

func (s *Server) buildNotify(clean bool) {
	err := s.hugo.Build(clean)
	if err != nil {
		s.n.Error(fmt.Errorf("build failed: %w", err))
	}
}

func (s *Server) buildHook(dir string) {
	s.log.Infof("received new public directory: %s", dir)

	s.staticFsLock.Lock()
	oldFs := s.staticFs
	s.staticFs = newStaticFs(dir)
	s.staticFsLock.Unlock()

	if oldFs != nil {
		err := os.RemoveAll(oldFs.dir)
		if err != nil {
			s.n.Error(fmt.Errorf("could not delete old directory: %w", err))
		}
	}
}

func setCacheControl(w http.ResponseWriter, isHTML bool) {
	if isHTML {
		w.Header().Set("Cache-Control", "no-cache, no-store, max-age=0")
	} else {
		w.Header().Set("Cache-Control", "public, max-age=15552000")
	}
}

var etagHeaders = []string{
	"ETag",
	"If-Modified-Since",
	"If-Match",
	"If-None-Match",
	"If-Range",
	"If-Unmodified-Since",
}

func delEtagHeaders(r *http.Request) {
	for _, v := range etagHeaders {
		if r.Header.Get(v) != "" {
			r.Header.Del(v)
		}
	}
}
