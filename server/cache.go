package server

import "net/http"

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
