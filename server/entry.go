package server

import (
	"fmt"
	"net/http"
	urlpkg "net/url"
	"os"
	"strconv"
	"strings"

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
	entry, err := s.GetEntry(r.URL.Path)
	if os.IsNotExist(err) {
		s.serveErrorHTML(w, r, http.StatusNotFound, nil)
		return
	}

	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	tpls := []string{}
	if entry.Section != "" {
		tpls = append(tpls, eagle.TemplateSingle+"."+entry.Section)
	}
	tpls = append(tpls, eagle.TemplateSingle)

	s.serveHTML(w, r, &eagle.RenderData{
		Entry: entry,
	}, tpls)
}

func (s *Server) entryPost(w http.ResponseWriter, r *http.Request) {
	// TODO: request has action. Action can be editing the post itself
	// or hiding a webmention.
}

func (s *Server) indexGet(w http.ResponseWriter, r *http.Request) {
	s.listingGet(w, r, &listingSettings{
		query: &eagle.SearchQuery{
			Sections: s.Config.Site.IndexSections,
		},
		templates: []string{eagle.TemplateIndex},
	})
}

func (s *Server) tagGet(w http.ResponseWriter, r *http.Request) {
	tag := chi.URLParam(r, "tag")
	if tag == "" {
		s.serveErrorHTML(w, r, http.StatusNotFound, nil)
		return
	}

	entry := s.getEntryOrEmpty(r.URL.Path)
	if entry.Title == "" {
		entry.Title = "#" + tag
	}

	s.listingGet(w, r, &listingSettings{
		query: &eagle.SearchQuery{
			Tags: []string{tag},
		},
		rd: &eagle.RenderData{
			Entry: entry,
		},
	})
}

func (s *Server) sectionGet(section string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		entry := s.getEntryOrEmpty(r.URL.Path)
		if entry.Title == "" {
			entry.Title = section
		}

		if entry.Section == "" {
			entry.Section = section
		}

		s.listingGet(w, r, &listingSettings{
			query: &eagle.SearchQuery{
				Sections: []string{section},
			},
			rd: &eagle.RenderData{
				Entry: entry,
			},
			templates: []string{eagle.TemplateList + "." + section},
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
		s.serveErrorHTML(w, r, http.StatusNotFound, nil)
		return
	}

	var title strings.Builder
	if year != 0 {
		ys := fmt.Sprintf("%0004d", year)
		title.WriteString(ys)
	} else {
		title.WriteString("XXXX")
	}

	if month != 0 {
		title.WriteString(fmt.Sprintf("-%02d", month))
	} else if day != 0 {
		title.WriteString("-XX")
	}

	if day != 0 {
		title.WriteString(fmt.Sprintf("-%02d", day))
	}

	s.listingGet(w, r, &listingSettings{
		query: &eagle.SearchQuery{
			Year:  year,
			Month: month,
			Day:   day,
		},
		rd: &eagle.RenderData{
			Entry: &eagle.Entry{
				Frontmatter: eagle.Frontmatter{
					Title: title.String(),
				},
			},
		},
	})
}

func (s *Server) searchGet(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")

	entry := s.getEntryOrEmpty(r.URL.Path)
	if entry.Title == "" {
		entry.Title = "Search"
	}

	if query == "" {
		s.serveHTML(w, r, &eagle.RenderData{
			Entry: entry,
		}, []string{eagle.TemplateSearch})
		return
	}

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

	s.listingGet(w, r, &listingSettings{
		query: &eagle.SearchQuery{
			Query:    query,
			Sections: sections,
		},
		rd: &eagle.RenderData{
			Entry:       entry,
			SearchQuery: query,
		},
		templates: []string{eagle.TemplateSearch},
	})
}

func (s *Server) getEntryOrEmpty(id string) *eagle.Entry {
	if entry, err := s.GetEntry(id); err == nil {
		return entry
	} else {
		return &eagle.Entry{
			Frontmatter: eagle.Frontmatter{},
		}
	}
}

type listingSettings struct {
	query     *eagle.SearchQuery
	rd        *eagle.RenderData
	templates []string
}

func (s *Server) listingGet(w http.ResponseWriter, r *http.Request, ls *listingSettings) {
	ls.query.ByDate = true
	ls.query.Private = false // TODO true if logged in
	ls.query.Draft = false   // TODO true if logged in
	ls.query.Deleted = false // TODO true if logged in

	if ls.rd == nil {
		ls.rd = &eagle.RenderData{}
	}

	if ls.rd.Entry == nil {
		ls.rd.Entry = s.getEntryOrEmpty(r.URL.Path)
	}

	page := 0
	if v := r.URL.Query().Get("page"); v != "" {
		vv, _ := strconv.Atoi(v)
		if vv >= 0 {
			page = vv
		}
	}

	entries, err := s.Search(ls.query, page)
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	ls.rd.Entries = entries
	ls.rd.IsListing = true

	if len(entries) == s.Config.Site.Paginate {
		url, _ := urlpkg.Parse(r.URL.String())
		url.RawQuery = "page=" + strconv.Itoa(page+1)
		ls.rd.NextPage = url.String()
	}

	if feed := chi.URLParam(r, "feed"); feed != "" {
		// https://github.com/jlelse/feeds
		w.WriteHeader(http.StatusNotImplemented)
		return
	}

	s.serveHTML(w, r, ls.rd, append(ls.templates, eagle.TemplateList))
}

// func (s *Server) goSyndicate(entry *eagle.Entry) {
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
// }

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
