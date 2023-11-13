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
	"github.com/robfig/cron/v3"
	"github.com/samber/lo"
	"go.hacdias.com/eagle/core"
	"go.hacdias.com/eagle/log"
	"go.hacdias.com/eagle/services/database"
	"go.hacdias.com/eagle/services/media"
	"go.hacdias.com/eagle/services/meilisearch"
	"go.hacdias.com/indielib/indieauth"

	"go.uber.org/zap"
)

type contextKey string

type Server struct {
	n core.Notifier
	c *core.Config

	log     *zap.SugaredLogger
	ias     *indieauth.Server
	jwtAuth *jwtauth.JWTAuth
	actions map[string]func() error

	cron     *cron.Cron
	cronJobs []func() error

	redirects map[string]string
	gone      map[string]bool
	links     []core.Links
	linksMap  map[string]core.Links

	server      *http.Server
	meilisearch *meilisearch.MeiliSearch
	fs          *core.FS
	hugo        *core.Hugo
	media       *media.Media
	badger      *database.Database

	staticFsLock sync.RWMutex
	staticFs     *staticFs
}

func NewServer(c *core.Config) (*Server, error) {
	s := &Server{
		c: c,

		log:     log.S().Named("server"),
		ias:     indieauth.NewServer(false, &http.Client{Timeout: time.Second * 30}),
		jwtAuth: jwtauth.New("HS256", []byte(base64.StdEncoding.EncodeToString([]byte(c.TokensSecret))), nil),
		actions: map[string]func() error{},

		cron:     cron.New(),
		cronJobs: []func() error{},

		redirects: map[string]string{},
		gone:      map[string]bool{},
		links:     []core.Links{},
		linksMap:  map[string]core.Links{},

		fs:    initFS(c),
		hugo:  core.NewHugo(c.SourceDirectory, c.PublicDirectory, c.BaseURL),
		media: initMedia(c),
	}
	s.hugo.BuildHook = s.buildHook

	err := errors.Join(
		s.initNotifier(),
		s.initBadger(),
		s.initMeiliSearch(),
		s.initActions(),
		s.initMiniflux(),
		s.initLinkding(),
		s.initExternalLinks(),
		s.loadRedirects(),
		s.loadGone(),
		s.loadLinks(),
		s.initCron(),
	)

	return s, err
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

	return errors.Join(
		s.server.Shutdown(ctx),
		s.badger.Close(),
	)
}

func (s *Server) getActions() []string {
	actions := []string{}
	for action := range s.actions {
		actions = append(actions, action)
	}
	sort.Strings(actions)
	return actions
}

func (s *Server) registerAction(name string, action func() error) error {
	if _, ok := s.actions[name]; ok {
		return errors.New("action already registered")
	}

	s.actions[name] = action
	return nil
}

func (s *Server) registerActionWithRebuild(name string, action func() error) error {
	return s.registerAction(name, func() error {
		err := action()
		if err != nil {
			return err
		}
		return s.hugo.Build(false)
	})
}

func (s *Server) loadRedirects() error {
	redirects, err := s.fs.LoadRedirects(true)
	if err != nil {
		return err
	}
	s.redirects = redirects
	return nil
}

func (s *Server) loadGone() error {
	gone, err := s.fs.LoadGone()
	if err != nil {
		return err
	}
	s.gone = gone
	return nil
}

func (s *Server) loadLinks() error {
	links, err := s.fs.LoadExternalLinks()
	if err != nil {
		return err
	}
	linksMap := map[string]core.Links{}
	for _, l := range links {
		linksMap[l.Domain] = l
	}

	s.links = links
	s.linksMap = linksMap
	return nil
}

func (s *Server) indexAll() {
	if s.meilisearch == nil {
		return
	}

	err := s.meilisearch.ResetIndex()
	if err != nil {
		s.n.Error(err)
	}

	entries, err := s.fs.GetEntries(false)
	if err != nil {
		s.n.Error(err)
		return
	}

	start := time.Now()
	err = s.meilisearch.Add(entries...)
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
	// TODO: detect if redirects/gone changes, reload.

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
			if s.meilisearch != nil {
				_ = s.meilisearch.Remove(id)
			}
			buildClean = true
			continue
		} else if err != nil {
			s.n.Error(fmt.Errorf("cannot open entry to update %s: %w", id, err))
			continue
		}
		entries = append(entries, entry)
	}

	if s.meilisearch != nil {
		err = s.meilisearch.Add(entries...)
		if err != nil {
			s.n.Error(fmt.Errorf("sync failed: %w", err))
		}
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
