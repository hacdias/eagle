package server

import (
	"context"
	"embed"
	"encoding/base64"
	"errors"
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
	"github.com/maypok86/otter/v2"
	"github.com/robfig/cron/v3"
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

	log         *zap.SugaredLogger
	ias         *indieauth.Server
	jwtAuth     *jwtauth.JWTAuth
	actions     map[string]func() error
	plugins     map[string]Plugin
	syndicators map[string]SyndicationPlugin
	cron        *cron.Cron

	redirects map[string]string
	gone      map[string]bool

	serversMu    sync.Mutex
	servers      map[string]*http.Server
	onionAddress string
	meilisearch  *meilisearch.Meilisearch
	core         *core.Core

	media      *media.Media
	mediaCache *otter.Cache[string, []byte]

	bolt *database.Database

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

		cron: cron.New(),

		redirects: map[string]string{},
		gone:      map[string]bool{},

		servers: map[string]*http.Server{},
		core:    co,
		media:   initMedia(c),
	}

	co.BuildHook = s.buildHook

	err = errors.Join(
		s.initMediaCache(),
		s.initNotifier(),
		s.initTemplates(),
		s.initBolt(),
		s.initMeilisearch(),
		s.initPlugins(),
		s.initSyndicators(),
		s.initActions(),
		s.initCron(),
		s.loadRedirects(),
		s.loadGone(),
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
	router := s.makeRouter()
	errCh := make(chan error)

	// Start server(s)
	err = s.startServer(errCh, router)
	if err != nil {
		return err
	}

	if s.c.Tor {
		err = s.startTor(errCh, router)
		if err != nil {
			s.log.Errorw("onion service failed to start", "err", err)
		}
	}

	return <-errCh
}

func (s *Server) registerServer(srv *http.Server, name string) {
	s.serversMu.Lock()
	defer s.serversMu.Unlock()

	s.servers[name] = srv
}

func (s *Server) startServer(errCh chan error, h http.Handler) error {
	addr := ":" + strconv.Itoa(s.c.Port)
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

func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	<-s.cron.Stop().Done()

	var err error
	for name, srv := range s.servers {
		s.log.Infof("shutting down %s", name)
		err = errors.Join(err, srv.Shutdown(ctx))
	}

	return errors.Join(err, s.bolt.Close())
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
		s.log.Errorw("failed to get entries", "err", err)
		return
	}

	if s.meilisearch != nil {
		err := s.meilisearch.ResetIndex()
		if err != nil {
			s.log.Errorw("failed to reset meilisearch index", "err", err)
		}

		start := time.Now()
		err = s.meilisearch.Add(entries...)
		if err != nil {
			s.log.Errorw("failed to add to meilisearch index", "err", err)
		}
		s.log.Infof("meilisearch update took %dms", time.Since(start).Milliseconds())
	}

	if s.c.Micropub != nil {
		err := s.bolt.ResetTaxonomies(context.Background())
		if err != nil {
			s.log.Errorw("failed to reset taxonomies", "err", err)
		}

		start := time.Now()

		tags := []string{}
		categories := []string{}

		for _, e := range entries {
			tags = append(tags, e.Tags...)
			categories = append(categories, e.Categories...)
		}

		err = errors.Join(
			s.bolt.AddTaxonomy(context.Background(), core.TagsTaxonomy, tags...),
			s.bolt.AddTaxonomy(context.Background(), core.CategoriesTaxonomy, categories...),
		)
		if err != nil {
			s.log.Errorw("failed to index all taxonomies", "err", err)
		}

		s.log.Infof("bolt taxonomies update took %dms", time.Since(start).Milliseconds())
	}
}

func (s *Server) withRecoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rvr := recover(); rvr != nil && rvr != http.ErrAbortHandler {
				s.log.Errorw("panic while serving", "rvr", rvr, "stack", string(debug.Stack()))
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
		s.log.Errorw("failed to sync storage", "err", err)
		return
	}

	if len(changedFiles) == 0 {
		return
	}

	// TODO: detect if redirects and gone have changed, reload.

	modifiedEntries := s.entriesFromModifiedFiles(changedFiles)
	deletedEntries := []*core.Entry{}
	updatedEntries := []*core.Entry{}
	previousLinks := map[string][]string{}

	// Detect update and deleted entries
	for _, modifiedEntry := range modifiedEntries {
		e, err := s.core.GetEntry(modifiedEntry.ID)
		if os.IsNotExist(err) {
			// Properly mark as expired so that post save hooks know it
			modifiedEntry.ExpiryDate = time.Now()
			deletedEntries = append(deletedEntries, modifiedEntry)
		} else if err != nil {
			s.log.Errorw("failed to open entry to update", "id", modifiedEntry.ID, "err", err)
			continue
		} else {
			updatedEntries = append(updatedEntries, e)
		}

		if links, err := s.core.GetEntryLinks(modifiedEntry, true); err == nil {
			previousLinks[e.Permalink] = links
		}
	}

	// Sync meilisearch.
	if s.meilisearch != nil {
		for _, deletedEntry := range deletedEntries {
			err = s.meilisearch.Remove(deletedEntry.ID)
			if err != nil {
				s.log.Errorw("failed to remove entry from meilisearch", "id", deletedEntry.ID, "err", err)
			}
		}

		err = s.meilisearch.Add(updatedEntries...)
		if err != nil {
			s.log.Errorw("failed to add entries to meilisearch", "err", err)
		}
	}

	s.build(len(deletedEntries) > 0)

	// For deleted entries, run syndication and send webmentions so that the
	// syndications are removed and the webmention targets are notified of the deletion.
	for _, e := range deletedEntries {
		s.Syndicate(e, nil)

		err := s.core.SendWebmentions(e, previousLinks[e.Permalink]...)
		if err != nil {
			s.log.Errorw("failed to send webmentions", "id", e.ID, "err", err)
		}

		time.Sleep(time.Second)
	}

	if len(updatedEntries) > 10 {
		s.log.Warn("not running post save hooks due to high quantity of changed entries")
		return
	}

	for _, e := range updatedEntries {
		s.postSaveEntry(e, nil, previousLinks[e.Permalink], true)
		time.Sleep(time.Second)
	}
}

func (s *Server) entriesFromModifiedFiles(modifiedFiles []core.ModifiedFile) []*core.Entry {
	var ee []*core.Entry

	for _, modifiedFile := range modifiedFiles {
		parts := strings.FieldsFunc(modifiedFile.Filename, func(r rune) bool {
			return filepath.Separator == r
		})

		if len(parts) <= 1 {
			continue
		}

		if parts[0] != core.ContentDirectory {
			continue
		}

		lastElement := parts[len(parts)-1]
		if lastElement != "index.md" && lastElement != "_index.md" {
			continue
		}

		id := filepath.Join(parts[1 : len(parts)-1]...)

		e, err := s.core.GetEntryFromContent(id, modifiedFile.Content)
		if err != nil {
			s.log.Errorw("failed to get entry from content for changed file", "id", id, "err", err)
			continue
		}

		ee = append(ee, e)
	}

	return ee
}

func (s *Server) build(clean bool) {
	err := s.core.Build(clean)
	if err != nil {
		s.log.Errorw("failed to build", "err", err)
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
			s.log.Errorw("failed to delete old build directory", "path", oldFs.dir, "err", err)
		}
	}
}
