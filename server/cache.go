package server

import (
	"bytes"
	"net/http"
	"time"

	"github.com/hacdias/eagle/cache"
)

func (s *Server) cacheScope(r *http.Request) cache.CacheScope {
	if s.isUsingTor(r) {
		return cache.CacheTor
	}

	return cache.CacheRegular
}

func (s *Server) isCacheable(r *http.Request) bool {
	return s.getUser(r) == "" && r.URL.RawQuery == ""
}

func (s *Server) isCached(r *http.Request) ([]byte, time.Time, bool) {
	if !s.isCacheable(r) {
		return nil, time.Time{}, false
	}

	return s.cache.Cached(s.cacheScope(r), r.URL.Path)
}

func (s *Server) saveCache(r *http.Request, data []byte) {
	s.cache.Save(s.cacheScope(r), r.URL.Path, data, time.Now())
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
