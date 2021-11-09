package server

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	urlpkg "net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hacdias/eagle/v2/eagle"
	"github.com/jlelse/feeds"
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

	loggedIn := s.isLoggedIn(w, r)
	if entry.Deleted && !loggedIn {
		s.serveErrorHTML(w, r, http.StatusGone, nil)
		return
	}

	if (entry.Draft || entry.Private) && !loggedIn {
		// Do not give away it exists!
		s.serveErrorHTML(w, r, http.StatusNotFound, nil)
		return
	}

	if r.URL.Query().Get("edit") == "true" && loggedIn {
		s.serveHTML(w, r, &eagle.RenderData{
			Entry: entry,
		}, []string{eagle.TemplateEditor})
	} else {
		s.serveEntry(w, r, entry)
	}
}

func (s *Server) entryPost(w http.ResponseWriter, r *http.Request) {
	// TODO(v2): request has action. Action can be editing the post itself
	// or hiding a webmention.

	_, err := s.GetEntry(r.URL.Path)
	if os.IsNotExist(err) {
		s.serveErrorHTML(w, r, http.StatusNotFound, nil)
		return
	}

	err = r.ParseForm()
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, err)
		return
	}

	content := r.Form.Get("content")
	if content == "" {
		s.serveErrorHTML(w, r, http.StatusBadRequest, errors.New("content cannot be empty"))
		return
	}

	entry, err := s.ParseEntry(r.URL.Path, content)
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, err)
		return
	}

	lastmod := r.FormValue("lastmod") == "on"
	if lastmod {
		entry.Updated = time.Now()
	}

	err = s.SaveEntry(entry)
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	go s.postSavePost(entry, &postSaveOptions{
		twitter: r.FormValue("twitter") == "on",
	})

	s.serveEntry(w, r, entry)
}

func (s *Server) serveEntry(w http.ResponseWriter, r *http.Request, entry *eagle.Entry) {
	s.serveHTML(w, r, &eagle.RenderData{
		Entry: entry,
	}, entry.Templates())
}

func (s *Server) allGet(w http.ResponseWriter, r *http.Request) {
	s.listingGet(w, r, &listingSettings{
		exec: func(opts *eagle.QueryOptions) ([]*eagle.Entry, error) {
			return s.QueryEntries(opts)
		},
	})
}

func (s *Server) indexGet(w http.ResponseWriter, r *http.Request) {
	s.listingGet(w, r, &listingSettings{
		rd: &eagle.RenderData{
			IsHome: true,
		},
		exec: func(opts *eagle.QueryOptions) ([]*eagle.Entry, error) {
			return s.QuerySection(s.Config.Site.IndexSections, opts)
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
		rd: &eagle.RenderData{
			Entry: entry,
		},
		exec: func(opts *eagle.QueryOptions) ([]*eagle.Entry, error) {
			return s.QueryTag(tag, opts)
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
			rd: &eagle.RenderData{
				Entry: entry,
			},
			exec: func(opts *eagle.QueryOptions) ([]*eagle.Entry, error) {
				return s.QuerySection([]string{section}, opts)
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
		rd: &eagle.RenderData{
			Entry: &eagle.Entry{
				Frontmatter: eagle.Frontmatter{
					Title: title.String(),
				},
			},
		},
		exec: func(opts *eagle.QueryOptions) ([]*eagle.Entry, error) {
			return s.QueryDate(year, month, day, opts)
		},
	})
}

func (s *Server) searchGet(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")

	entry := s.getEntryOrEmpty(r.URL.Path)
	if entry.Title == "" {
		entry.Title = "Search"
	}
	if entry.ID == "" {
		entry.ID = strings.TrimSuffix(r.URL.Path, filepath.Ext(r.URL.Path))
	}

	if query == "" {
		s.serveHTML(w, r, &eagle.RenderData{
			Entry: entry,
		}, []string{eagle.TemplateSearch})
		return
	}

	s.listingGet(w, r, &listingSettings{
		rd: &eagle.RenderData{
			Entry:       entry,
			SearchQuery: query,
		},
		exec: func(opts *eagle.QueryOptions) ([]*eagle.Entry, error) {
			return s.SearchPostgres(query, opts)
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
	exec      func(*eagle.QueryOptions) ([]*eagle.Entry, error)
	rd        *eagle.RenderData
	templates []string
}

func (s *Server) listingGet(w http.ResponseWriter, r *http.Request, ls *listingSettings) {
	loggedIn := s.isLoggedIn(w, r)

	opts := &eagle.QueryOptions{
		Draft:   loggedIn,
		Deleted: loggedIn,
		Private: loggedIn,
	}

	if ls.rd == nil {
		ls.rd = &eagle.RenderData{}
	}

	if ls.rd.Entry == nil {
		ls.rd.Entry = s.getEntryOrEmpty(r.URL.Path)
	}

	if v := r.URL.Query().Get("page"); v != "" {
		vv, _ := strconv.Atoi(v)
		if vv >= 0 {
			opts.Page = vv
			ls.rd.Page = vv
		}
	}

	entries, err := ls.exec(opts)
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	ls.rd.Entries = entries
	ls.rd.IsListing = true

	if len(entries) == s.Config.Site.Paginate {
		url, _ := urlpkg.Parse(r.URL.String())
		url.RawQuery = "page=" + strconv.Itoa(opts.Page+1)
		ls.rd.NextPage = url.String()
	}

	feedType := chi.URLParam(r, "feed")
	if feedType == "" {
		s.serveHTML(w, r, ls.rd, append(ls.templates, eagle.TemplateList))
		return
	}

	feed := &feeds.Feed{
		Title:       ls.rd.Entry.Title,
		Link:        &feeds.Link{Href: strings.TrimSuffix(s.AbsoluteURL(r.URL.Path), "."+feedType)},
		Description: ls.rd.Entry.Summary(),
		Author: &feeds.Author{
			Name:  s.Config.User.Name,
			Email: s.Config.User.Email,
		},
		// TODO(future): support .Tags
		Created: time.Now(),
		Items:   []*feeds.Item{},
	}

	for _, entry := range entries {
		var buf bytes.Buffer
		err = s.Render(&buf, &eagle.RenderData{Entry: entry}, []string{eagle.TemplateFeed})
		if err != nil {
			s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
			return
		}

		feed.Items = append(feed.Items, &feeds.Item{
			Title:       entry.Title,
			Link:        &feeds.Link{Href: entry.Permalink},
			Id:          entry.ID,
			Description: entry.Description,
			Content:     buf.String(),
			Author: &feeds.Author{
				Name:  s.Config.User.Name,
				Email: s.Config.User.Email,
			},
			Created: entry.Published,
			Updated: entry.Updated,
		})
	}

	var feedString, feedMediaType string

	switch feedType {
	case "rss":
		feedString, err = feed.ToRss()
		feedMediaType = "application/rss+xml"

	case "atom":
		feedString, err = feed.ToAtom()
		feedMediaType = "application/atom+xml"
	case "json":
		feedString, err = feed.ToJSON()
		feedMediaType = "application/feed+json"
	}

	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	w.Header().Set("Content-Type", feedMediaType+"; charset=utf-8")
	_, err = w.Write([]byte(feedString))
	if err != nil {
		// TODO: maybe notify as these are terminal errors kinda.
		s.log.Error("error while serving feed", err)
	}
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

type postSaveOptions struct {
	twitter bool
}

func (s *Server) postSavePost(entry *eagle.Entry, opts *postSaveOptions) {
	// Invalidate cache
	s.PostSaveEntry(entry)
}
