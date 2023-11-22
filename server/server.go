package server

import (
	"context"
	"embed"
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
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

var (
	//go:embed templates/*.html
	panelTemplatesFS embed.FS
	panelTemplates   = template.Must(template.ParseFS(panelTemplatesFS, "templates/*.html"))

	//go:embed assets/*
	panelAssetsFS embed.FS
)

type Server struct {
	n core.Notifier
	c *core.Config

	log      *zap.SugaredLogger
	ias      *indieauth.Server
	jwtAuth  *jwtauth.JWTAuth
	actions  map[string]func() error
	plugins  map[string]Plugin
	cron     *cron.Cron
	cronJobs []func() error

	redirects map[string]string
	gone      map[string]bool

	server      *http.Server
	meilisearch *meilisearch.MeiliSearch
	core        *core.Core
	media       *media.Media
	bolt        *database.Database

	staticFsLock sync.RWMutex
	staticFs     *staticFs
	templates    *template.Template
}

func NewServer(c *core.Config) (*Server, error) {
	co, err := core.NewCore(c)
	if err != nil {
		return nil, err
	}

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

		core:  co,
		media: initMedia(c),
	}

	co.BuildHook = s.buildHook

	err = errors.Join(
		s.initNotifier(),
		s.initTemplates(),
		s.initBolt(),
		s.initMeiliSearch(),
		s.initPlugins(),
		s.initActions(),
		s.loadRedirects(),
		s.loadGone(),
		s.initCron(),
	)

	return s, err
}

func (s *Server) Start() error {
	go func() {
		s.indexAll()
	}()

	// Make sure we have a built version to serve
	should, err := s.core.ShouldBuild()
	if err != nil {
		return err
	}

	if should {
		err = s.core.Build(false)
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
		s.bolt.Close(),
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

func (s *Server) loadRedirects() error {
	redirects, err := s.core.GetRedirects(true)
	if err != nil {
		return err
	}
	s.redirects = redirects
	return nil
}

func (s *Server) loadGone() error {
	gone, err := s.core.GetGone()
	if err != nil {
		return err
	}
	s.gone = gone
	return nil
}

func (s *Server) indexAll() {
	if s.meilisearch == nil && s.c.Micropub == nil {
		return
	}

	entries, err := s.core.GetEntries(false)
	if err != nil {
		s.n.Error(err)
		return
	}

	if s.meilisearch != nil {
		err := s.meilisearch.ResetIndex()
		if err != nil {
			s.n.Error(err)
		}

		start := time.Now()
		err = s.meilisearch.Add(entries...)
		if err != nil {
			s.n.Error(err)
		}
		s.log.Infof("meilisearch update took %dms", time.Since(start).Milliseconds())
	}

	if s.c.Micropub != nil {
		err := s.bolt.ResetTaxonomies(context.Background())
		if err != nil {
			s.n.Error(err)
		}

		start := time.Now()

		if s.c.Micropub.CategoriesTaxonomy != "" {
			err = s.indexAllTaxonomies(entries, s.c.Micropub.CategoriesTaxonomy)
			if err != nil {
				s.n.Error(err)
			}
		}

		if s.c.Micropub.ChannelsTaxonomy != "" {
			err = s.indexAllTaxonomies(entries, s.c.Micropub.ChannelsTaxonomy)
			if err != nil {
				s.n.Error(err)
			}
		}

		s.log.Infof("bolt taxonomies update took %dms", time.Since(start).Milliseconds())
	}
}

func (s *Server) indexAllTaxonomies(ee core.Entries, taxonomy string) error {
	taxons := []string{}

	for _, e := range ee {
		taxons = append(taxons, e.Taxonomy(taxonomy)...)
	}

	return s.bolt.AddTaxonomy(context.Background(), taxonomy, taxons...)
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
	changedFiles, err := s.core.Sync()
	if err != nil {
		s.n.Error(fmt.Errorf("sync storage: %w", err))
		return
	}

	if len(changedFiles) == 0 {
		return
	}

	// TODO: detect if redirects and gone have changed, reload.

	ids := idsFromChangedFiles(changedFiles)
	ee := core.Entries{}
	previousLinks := map[string][]string{}
	buildClean := false

	for _, id := range ids {
		e, err := s.core.GetEntry(id)
		if os.IsNotExist(err) {
			if s.meilisearch != nil {
				_ = s.meilisearch.Remove(id)
			}
			buildClean = true
			continue
		} else if err != nil {
			s.n.Error(fmt.Errorf("cannot open entry to update %s: %w", id, err))
			continue
		} else {
			ee = append(ee, e)

			// Attempt to collect links that were in the entries before the updates.
			targets, err := s.core.GetEntryLinks(e.Permalink)
			if err == nil {
				previousLinks[e.Permalink] = targets
			}
		}
	}

	// Sync meilisearch.
	if s.meilisearch != nil {
		err = s.meilisearch.Add(ee...)
		if err != nil {
			s.n.Error(fmt.Errorf("meilisearch sync failed: %w", err))
		}
	}

	s.buildNotify(buildClean)

	// After building, send webmentions with new information and old links.
	// This is a best effort to send webmentions to deleted links. Only works
	// with deletions that use expiryDate.
	for _, e := range ee {
		if e.Draft || e.NoWebmentions {
			continue
		}

		err = s.core.SendWebmentions(e.Permalink, previousLinks[e.Permalink]...)
		if err != nil {
			s.n.Error(fmt.Errorf("send webmentions: %w", err))
		}
	}
}

func idsFromChangedFiles(changedFiles []string) []string {
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
	return ids
}

func (s *Server) buildNotify(clean bool) {
	err := s.core.Build(clean)
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
