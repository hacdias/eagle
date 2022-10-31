package server

import (
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

const defaultTileSource = "https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"

func (s *Server) tilesGet(w http.ResponseWriter, r *http.Request) {
	tileSource := defaultTileSource

	if s.Config.MapBox != nil {
		tileSource = s.Config.MapBox.TileSource()
	}

	urlStr := strings.ReplaceAll(tileSource, "{s}", chi.URLParam(r, "s"))
	urlStr = strings.ReplaceAll(urlStr, "{z}", chi.URLParam(r, "z"))
	urlStr = strings.ReplaceAll(urlStr, "{x}", chi.URLParam(r, "x"))
	urlStr = strings.ReplaceAll(urlStr, "{y}", chi.URLParam(r, "y"))
	rp := chi.URLParam(r, "r")
	if rp != "" {
		rp = "@" + rp
	}
	urlStr = strings.ReplaceAll(urlStr, "{r}", rp)

	req, _ := http.NewRequestWithContext(r.Context(), http.MethodGet, urlStr, nil)

	// Copy request headers
	for _, k := range []string{
		"Accept-Encoding",
		"Accept-Language",
		"Accept",
		"Cache-Control",
		"If-Modified-Since",
		"If-None-Match",
		"User-Agent",
	} {
		req.Header.Set(k, r.Header.Get(k))
	}

	// Do proxy request
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	// Copy result headers
	for _, k := range []string{
		"Accept-Ranges",
		"Access-Control-Allow-Origin",
		"Age",
		"Cache-Control",
		"Content-Length",
		"Content-Type",
		"Etag",
		"Expires",
	} {
		w.Header().Set(k, res.Header.Get(k))
	}

	// Pipe result
	w.WriteHeader(res.StatusCode)
	_, _ = io.Copy(w, res.Body)
	_ = res.Body.Close()
}
