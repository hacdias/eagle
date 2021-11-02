package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	// urlpkg "net/url"

	"github.com/go-chi/chi/v5"
	"github.com/hacdias/eagle/v2/eagle"
)

func (s *Server) newGet(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("new post"))
}

func (s *Server) newPost(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("new post"))
}

func (s *Server) entryGet(w http.ResponseWriter, r *http.Request) {

}

func (s *Server) entryPost(w http.ResponseWriter, r *http.Request) {
	// TODO: request has action. Action can be editing the post itself
	// or hiding a webmention.
}

func (s *Server) goSyndicate(entry *eagle.Entry) {
	// if s.Twitter == nil {
	// 	return
	// }

	// url, err := s.Twitter.Syndicate(entry)
	// if err != nil {
	// 	s.NotifyError(fmt.Errorf("failed to syndicate: %w", err))
	// 	return
	// }

	// entry.Metadata.Syndication = append(entry.Metadata.Syndication, url)
	// err = s.SaveEntry(entry)
	// if err != nil {
	// 	s.NotifyError(fmt.Errorf("failed to save entry: %w", err))
	// 	return
	// }

	// INVALIDATE CACHE OR STH
}

// func (s *Server) goWebmentions(entry *eagle.Entry) {
// 	err := s.SendWebmentions(entry)
// 	if err != nil {
// 		s.NotifyError(fmt.Errorf("webmentions: %w", err))
// 	}
// }

// func sanitizeReplyURL(replyUrl string) string {
// 	if strings.HasPrefix(replyUrl, "https://twitter.com") && strings.Contains(replyUrl, "/status/") {
// 		url, err := urlpkg.Parse(replyUrl)
// 		if err != nil {
// 			return replyUrl
// 		}

// 		url.RawQuery = ""
// 		url.Fragment = ""

// 		return url.String()
// 	}

// 	return replyUrl
// }

// func sanitizeID(id string) (string, error) {
// 	if id != "" {
// 		url, err := urlpkg.Parse(id)
// 		if err != nil {
// 			return "", err
// 		}
// 		id = path.Clean(url.Path)
// 	}
// 	return id, nil
// }

func (s *Server) indexGet(w http.ResponseWriter, r *http.Request) {
	s.listingGet(w, r, &eagle.SearchQuery{})
}

func (s *Server) tagGet(w http.ResponseWriter, r *http.Request) {
	tag := chi.URLParam(r, "tag")
	if tag == "" {
		s.serveError(w, http.StatusNotFound, nil)
		return
	}

	s.listingGet(w, r, &eagle.SearchQuery{
		Tags: []string{tag},
	})
}

func (s *Server) sectionGet(section string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.listingGet(w, r, &eagle.SearchQuery{
			Sections: []string{section},
		})
	}
}

func (s *Server) dateGet(w http.ResponseWriter, r *http.Request) {
	var year, month, day int

	if ys := chi.URLParam(r, "year"); ys != "" && ys != "x" {
		year, _ = strconv.Atoi(ys)
	}

	if ms := chi.URLParam(r, "month"); ms != "" && ms != "x" {
		month, _ = strconv.Atoi(ms)
	}

	if ds := chi.URLParam(r, "day"); ds != "" {
		day, _ = strconv.Atoi(ds)
	}

	if year == 0 && month == 0 && day == 0 {
		s.serveError(w, http.StatusNotFound, nil)
		return
	}

	s.listingGet(w, r, &eagle.SearchQuery{
		Year:  year,
		Month: month,
		Day:   day,
	})
}

func (s *Server) searchGet(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")

	sectionsQuery := strings.TrimSpace(r.URL.Query().Get("s"))
	sectionsList := strings.Split(sectionsQuery, ",")
	sections := []string{}

	for _, s := range sectionsList {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		sections = append(sections, s)
	}

	s.listingGet(w, r, &eagle.SearchQuery{
		Query:    query,
		Sections: sections,
	})
}

type listingData struct {
	eagle.Entry
	Entries []*eagle.Entry
}

func (s *Server) listingGet(w http.ResponseWriter, r *http.Request, q *eagle.SearchQuery) {
	// If logged in, q.Private = q.Draft = q.Deleted = true

	feed := chi.URLParam(r, "feed")
	if feed != "" {
		fmt.Println(feed)
	}

	q.ByDate = true

	q.Private = false // TODO true if logged in
	q.Draft = false   // TODO true if logged in
	q.Deleted = false // TODO true if logged in

	s.render(w, &eagle.RenderData{
		Data: nil,
	}, []string{"list"})
}
