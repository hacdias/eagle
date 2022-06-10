package server

import (
	"bytes"
	"net/http"
	"time"

	"github.com/hacdias/eagle/v4/eagle"
)

func (s *Server) cacheScope(r *http.Request) eagle.CacheScope {
	if s.isUsingTor(r) {
		return eagle.CacheTor
	}

	return eagle.CacheRegular
}

func (s *Server) isCacheable(r *http.Request) bool {
	return s.getUser(r) == "" && r.URL.RawQuery == ""
}

func (s *Server) isCached(r *http.Request) ([]byte, time.Time, bool) {
	if !s.isCacheable(r) {
		return nil, time.Time{}, false
	}

	return s.IsCached(s.cacheScope(r), r.URL.Path)
}

func (s *Server) saveCache(r *http.Request, data []byte) {
	s.SaveCache(s.cacheScope(r), r.URL.Path, data, time.Now())
}

func (s *Server) withCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if data, modtime, ok := s.isCached(r); ok {
			setCacheHTML(w)
			http.ServeContent(w, r, "index.html", modtime, bytes.NewReader(data))
			return
		}

		next.ServeHTTP(w, r)
	})
}
