package server

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"path"
	"runtime/debug"
	"strconv"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/jwtauth/v5"
	"github.com/hacdias/eagle/v4/database"
	"github.com/hacdias/eagle/v4/eagle"
	"github.com/hacdias/eagle/v4/entry"
	"github.com/hacdias/eagle/v4/hooks"
	"github.com/hacdias/eagle/v4/log"
	"github.com/hacdias/eagle/v4/pkg/contenttype"
	"github.com/hacdias/eagle/v4/pkg/maze"
	"github.com/hacdias/eagle/v4/pkg/mf2"
	"github.com/hacdias/eagle/v4/pkg/xray"
	"github.com/hacdias/eagle/v4/syndicator"
	"github.com/hacdias/indieauth/v3"
	"github.com/hashicorp/go-multierror"
	"github.com/vartanbeno/go-reddit/v2/reddit"

	"go.uber.org/zap"
)

type contextKey string

type httpServer struct {
	Name string
	*http.Server
}

type Server struct {
	*eagle.Eagle
	log         *zap.SugaredLogger
	iac         *indieauth.Client
	ias         *indieauth.Server
	serversLock sync.Mutex
	servers     []*httpServer

	onionAddress string
	jwtAuth      *jwtauth.JWTAuth

	syndicator    *syndicator.Manager
	PreSaveHooks  []hooks.EntryHook
	PostSaveHooks []hooks.EntryHook
}

func NewServer(e *eagle.Eagle) (*Server, error) {
	clientID := e.Config.Server.BaseURL + "/"
	redirectURL := e.Config.Server.BaseURL + "/login/callback"

	s := &Server{
		Eagle:   e,
		log:     log.S().Named("server"),
		servers: []*httpServer{},
		iac: indieauth.NewClient(clientID, redirectURL, &http.Client{
			Timeout: time.Second * 30,
		}),
		ias: indieauth.NewServer(false, &http.Client{
			Timeout: time.Second * 30,
		}),
		syndicator: syndicator.NewManager(),
	}

	secret := base64.StdEncoding.EncodeToString([]byte(e.Config.Server.TokensSecret))
	s.jwtAuth = jwtauth.New("HS256", []byte(secret), nil)

	var allowedTypes []mf2.Type
	for typ := range e.Config.Micropub.Sections {
		allowedTypes = append(allowedTypes, typ)
	}

	if e.Config.Twitter != nil && e.Config.Syndications.Twitter {
		s.syndicator.Add(syndicator.NewTwitter(e.Config.Twitter))
	}

	var (
		err          error
		redditClient *reddit.Client
	)

	if e.Config.Reddit != nil {
		credentials := reddit.Credentials{
			ID:       e.Config.Reddit.App,
			Secret:   e.Config.Reddit.Secret,
			Username: e.Config.Reddit.User,
			Password: e.Config.Reddit.Password,
		}

		redditClient, err = reddit.NewClient(credentials)
		if err != nil {
			return nil, err
		}

		if e.Config.Syndications.Reddit {
			s.syndicator.Add(syndicator.NewReddit(redditClient))
		}
	}

	s.PreSaveHooks = append(
		s.PreSaveHooks,
		&hooks.IgnoreListing{Hook: hooks.AllowedType(allowedTypes)},
		&hooks.IgnoreListing{Hook: &hooks.DescriptionGenerator{}},
		&hooks.IgnoreListing{Hook: hooks.SectionDeducer(e.Config.Micropub.Sections)},
	)

	if e.Config.XRay != nil && e.Config.XRay.Endpoint != "" {
		options := &xray.XRayOptions{
			Log:       log.S().Named("xray"),
			Endpoint:  e.Config.XRay.Endpoint,
			UserAgent: fmt.Sprintf("Eagle/0.0 (%s) XRay", e.Config.ID()),
		}

		if e.Config.XRay.Twitter && e.Config.Twitter != nil {
			options.Twitter = &xray.Twitter{
				Key:         e.Config.Twitter.Key,
				Secret:      e.Config.Twitter.Secret,
				Token:       e.Config.Twitter.Token,
				TokenSecret: e.Config.Twitter.TokenSecret,
			}
		}

		if e.Config.XRay.Reddit && e.Config.Reddit != nil {
			options.Reddit = redditClient
		}

		xray := xray.NewXRay(options)

		s.PostSaveHooks = append(s.PostSaveHooks, &hooks.IgnoreListing{Hook: &hooks.ContextXRay{
			XRay:  xray,
			Eagle: s.Eagle,
		}})
	}

	s.PostSaveHooks = append(
		s.PostSaveHooks,
		// Process Photos
		&hooks.IgnoreListing{Hook: &hooks.LocationFetcher{
			Language: e.Config.Site.Language,
			Eagle:    e,
			Maze: maze.NewMaze(&http.Client{
				Timeout: 1 * time.Minute,
			}),
		}},
		// Webmentions
		&hooks.IgnoreListing{Hook: &hooks.ReadsSummaryUpdater{
			Eagle:    s.Eagle,
			Provider: s.Eagle.DB.(*database.Postgres), // wip: dont do this
		}},
		&hooks.IgnoreListing{Hook: &hooks.WatchesSummaryUpdater{
			Eagle:    s.Eagle,
			Provider: s.Eagle.DB.(*database.Postgres), // wip: dont do this
		}},
	)

	return s, nil
}

func (s *Server) Start() error {
	errCh := make(chan error)
	router := s.makeRouter()

	// Start server(s)
	err := s.startRegularServer(errCh, router)
	if err != nil {
		return err
	}

	if s.Config.Server.Tor != nil {
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
	return errs.ErrorOrNil()
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
	addr := ":" + strconv.Itoa(s.Config.Server.Port)
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
				s.Error(err)
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
		s.Notifier.Error(fmt.Errorf("serving html: %w", err))
	}
}

func (s *Server) serveErrorJSON(w http.ResponseWriter, code int, err, errDescription string) {
	s.serveJSON(w, code, map[string]interface{}{
		"error":             err,
		"error_description": errDescription,
	})
}

func (s *Server) serveHTMLWithStatus(w http.ResponseWriter, r *http.Request, data *eagle.RenderData, tpls []string, code int) {
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

	err := s.Render(cw, data, tpls)
	if err != nil {
		s.Notifier.Error(fmt.Errorf("serving html: %w", err))
	} else {
		data := buf.Bytes()
		if len(data) > 0 {
			s.saveCache(r, data)
		}
	}
}

func (s *Server) serveHTML(w http.ResponseWriter, r *http.Request, data *eagle.RenderData, tpls []string) {
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

	rd := &eagle.RenderData{
		Entry: &entry.Entry{
			FrontMatter: entry.FrontMatter{
				Title: fmt.Sprintf("%d %s", code, http.StatusText(code)),
			},
		},
		NoIndex: true,
		Data:    data,
	}

	s.serveHTMLWithStatus(w, r, rd, []string{eagle.TemplateError}, code)
}
