package server

import (
	"bytes"
	"fmt"
	"net/http"
	urlpkg "net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hacdias/eagle/eagle"
	"github.com/hacdias/eagle/indexer"
	"github.com/hacdias/eagle/pkg/contenttype"
	"github.com/hacdias/eagle/renderer"
	"github.com/jlelse/feeds"
	"github.com/thoas/go-funk"
)

func (s *Server) allGet(w http.ResponseWriter, r *http.Request) {
	s.listingGet(w, r, &listingSettings{
		exec: func(opts *indexer.Query) ([]*eagle.Entry, error) {
			return s.i.GetAll(opts)
		},
	})
}

func (s *Server) indexGet(w http.ResponseWriter, r *http.Request) {
	if s.ap != nil && isActivityPub(r) {
		s.serveActivity(w, http.StatusOK, s.ap.GetSelf())
		return
	}

	s.listingGet(w, r, &listingSettings{
		rd: &renderer.RenderData{
			IsHome: true,
		},
		exec: func(opts *indexer.Query) ([]*eagle.Entry, error) {
			return s.i.GetBySection(opts, s.c.Site.IndexSection)
		},
		templates: []string{renderer.TemplateIndex},
	})
}

func (s *Server) sectionGet(section string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ee := s.getListingEntryOrEmpty(r.URL.Path, section)
		if len(ee.Sections) == 0 {
			ee.Sections = []string{section}
		}

		s.listingGet(w, r, &listingSettings{
			rd: &renderer.RenderData{
				Entry: ee,
			},
			exec: func(opts *indexer.Query) ([]*eagle.Entry, error) {
				return s.i.GetBySection(opts, section)
			},
			templates: []string{},
		})
	}
}

func (s *Server) onThisDayGet(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	http.Redirect(w, r, fmt.Sprintf("/x/%02d/%02d", now.Month(), now.Day()), http.StatusSeeOther)
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
		rd: &renderer.RenderData{
			Entry: &eagle.Entry{
				FrontMatter: eagle.FrontMatter{
					Title:   title.String(),
					Listing: &eagle.Listing{},
				},
			},
		},
		exec: func(opts *indexer.Query) ([]*eagle.Entry, error) {
			return s.i.GetByDate(opts, year, month, day)
		},
	})
}

func (s *Server) taxonomyGet(id string, taxonomy eagle.Taxonomy) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		terms, err := s.i.GetTaxonomyTerms(id)
		if err != nil {
			s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
			return
		}

		e := s.getListingEntryOrEmpty(r.URL.Path, taxonomy.Title)
		templates := []string{}
		if e.Template != "" {
			templates = append(templates, e.Template)
		}
		templates = append(templates, renderer.TemplateTerms)

		s.serveHTML(w, r, &renderer.RenderData{
			Entry: e,
			Data: listingPage{
				Taxonomy: id,
				Terms:    terms,
			},
		}, templates)
	}
}

func (s *Server) taxonomyTermGet(id string, taxonomy eagle.Taxonomy) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		term := chi.URLParam(r, "term")
		if term == "" {
			s.serveErrorHTML(w, r, http.StatusNotFound, nil)
			return
		}

		s.listingGet(w, r, &listingSettings{
			rd: &renderer.RenderData{
				Entry: s.getListingEntryOrEmpty(r.URL.Path, taxonomy.Singular+": "+term),
			},
			exec: func(opts *indexer.Query) ([]*eagle.Entry, error) {
				return s.i.GetByTaxonomy(opts, id, term)
			},
			templates: []string{},
		})
	}
}

func (s *Server) searchGet(w http.ResponseWriter, r *http.Request) {
	search := &indexer.Search{
		Query:    r.URL.Query().Get("query"),
		Sections: []string{},
	}

	if r.URL.Query().Has("section") {
		search.Sections = r.URL.Query()["section"]
		search.Sections = funk.FilterString(search.Sections, func(s string) bool { return s != "" })
	}

	ee := s.getListingEntryOrEmpty(r.URL.Path, "Search")
	if search.Query == "" {
		s.serveHTML(w, r, &renderer.RenderData{
			Entry:   ee,
			NoIndex: true,
			Data: &listingPage{
				Search: search,
			},
		}, []string{renderer.TemplateSearch})
		return
	}

	s.listingGet(w, r, &listingSettings{
		rd: &renderer.RenderData{
			Entry:   ee,
			NoIndex: true,
		},
		lp: listingPage{
			Search: search,
		},
		exec: func(opts *indexer.Query) ([]*eagle.Entry, error) {
			if s.isAdmin(r) {
				opts.WithDrafts = true
				opts.WithDeleted = true
				opts.Visibility = nil
			}

			return s.i.Search(opts, search)
		},
		templates: []string{renderer.TemplateSearch},
	})
}

func (s *Server) privateGet(w http.ResponseWriter, r *http.Request) {
	s.listingGet(w, r, &listingSettings{
		rd: &renderer.RenderData{
			Entry:   s.getListingEntryOrEmpty(r.URL.Path, "Private"),
			NoIndex: true,
		},
		exec: func(opts *indexer.Query) ([]*eagle.Entry, error) {
			return s.i.GetPrivate(opts.Pagination, s.getUser(r))
		},
	})
}

func (s *Server) deletedGet(w http.ResponseWriter, r *http.Request) {
	s.listingGet(w, r, &listingSettings{
		rd: &renderer.RenderData{
			Entry:   s.getListingEntryOrEmpty(r.URL.Path, "Deleted"),
			NoIndex: true,
		},
		exec: func(opts *indexer.Query) ([]*eagle.Entry, error) {
			return s.i.GetDeleted(opts.Pagination)
		},
	})
}

func (s *Server) draftsGet(w http.ResponseWriter, r *http.Request) {
	s.listingGet(w, r, &listingSettings{
		rd: &renderer.RenderData{
			Entry:   s.getListingEntryOrEmpty(r.URL.Path, "Drafts"),
			NoIndex: true,
		},
		exec: func(opts *indexer.Query) ([]*eagle.Entry, error) {
			return s.i.GetDrafts(opts.Pagination)
		},
	})
}

func (s *Server) unlistedGet(w http.ResponseWriter, r *http.Request) {
	s.listingGet(w, r, &listingSettings{
		rd: &renderer.RenderData{
			Entry:   s.getListingEntryOrEmpty(r.URL.Path, "Unlisted"),
			NoIndex: true,
		},
		exec: func(opts *indexer.Query) ([]*eagle.Entry, error) {
			return s.i.GetUnlisted(opts.Pagination)
		},
	})
}

func (s *Server) getListingEntryOrEmpty(id, title string) *eagle.Entry {
	id = strings.TrimSuffix(id, filepath.Ext(id))
	if ee, err := s.fs.GetEntry(id); err == nil {
		if ee.Listing == nil {
			s.log.Warnf("entry %s should be marked as listing", ee.ID)
			ee.Listing = &eagle.Listing{}
		}
		return ee
	}

	return &eagle.Entry{
		ID: id,
		FrontMatter: eagle.FrontMatter{
			Title:   title,
			Listing: &eagle.Listing{},
		},
	}
}

type listingSettings struct {
	exec      func(*indexer.Query) ([]*eagle.Entry, error)
	rd        *renderer.RenderData
	lp        listingPage
	templates []string
}

type listingPage struct {
	Search   *indexer.Search
	Taxonomy string
	Terms    eagle.Terms
	Entries  []*eagle.Entry
	Page     int
	NextPage string
}

func (s *Server) listingGet(w http.ResponseWriter, r *http.Request, ls *listingSettings) {
	opts := &indexer.Query{}

	user := s.getUser(r)
	if user == "" {
		opts.Visibility = []eagle.Visibility{eagle.VisibilityPublic}
	} else {
		opts.Visibility = []eagle.Visibility{eagle.VisibilityPublic, eagle.VisibilityPrivate}
		if !s.isAdmin(r) {
			opts.Audience = s.getUser(r)
		}
	}

	if ls.rd == nil {
		ls.rd = &renderer.RenderData{}
	}

	if ls.rd.Entry == nil {
		ls.rd.Entry = s.getListingEntryOrEmpty(r.URL.Path, "")
	}

	opts.OrderByUpdated = ls.rd.Entry.Listing.OrderByUpdated

	if !ls.rd.Entry.Listing.DisablePagination {
		opts.Pagination = &indexer.Pagination{}

		if ls.rd.Entry.Listing.ItemsPerPage > 0 {
			opts.Pagination.Limit = ls.rd.Entry.Listing.ItemsPerPage
		} else {
			opts.Pagination.Limit = s.c.Site.Pagination
		}

		if v := r.URL.Query().Get("page"); v != "" {
			vv, _ := strconv.Atoi(v)
			if vv >= 0 {
				opts.Pagination.Page = vv
				ls.lp.Page = vv
			}
		}
	}

	entries, err := ls.exec(opts)
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	ls.lp.Entries = entries

	if len(entries) != 0 && !ls.rd.Entry.Listing.DisablePagination {
		url, _ := urlpkg.Parse(r.URL.String())
		values := url.Query()
		values.Set("page", strconv.Itoa(opts.Pagination.Page+1))
		url.RawQuery = values.Encode()
		ls.lp.NextPage = url.String()
	}

	ls.rd.Data = ls.lp

	feedType := chi.URLParam(r, "feed")
	if feedType == "" {
		templates := ls.templates
		if ls.rd.Template != "" {
			templates = append(templates, ls.rd.Template)
		}
		templates = append(templates, renderer.TemplateList)
		path := r.URL.Path

		ls.rd.Alternates = []renderer.Alternate{
			{
				Type: contenttype.JSONFeed,
				Href: path + ".json",
			},
			{
				Type: contenttype.ATOM,
				Href: path + ".atom",
			},
			{
				Type: contenttype.RSS,
				Href: path + ".rss",
			},
		}

		s.serveHTML(w, r, ls.rd, templates)
		return
	}

	feed := &feeds.Feed{
		Title:       ls.rd.Entry.TextTitle(),
		Link:        &feeds.Link{Href: strings.TrimSuffix(s.c.Server.AbsoluteURL(r.URL.Path), "."+feedType)},
		Description: ls.rd.Entry.TextDescription(),
		Author: &feeds.Author{
			Name:  s.c.User.Name,
			Email: s.c.User.Email,
		},
		Created: time.Now(),
		Items:   []*feeds.Item{},
	}

	for _, entry := range entries {
		var buf bytes.Buffer
		err = s.renderer.Render(&buf, &renderer.RenderData{Entry: entry}, []string{renderer.TemplateFeed})
		if err != nil {
			s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
			return
		}

		feed.Items = append(feed.Items, &feeds.Item{
			Title:       entry.TextTitle(),
			Link:        &feeds.Link{Href: entry.Permalink},
			Id:          entry.ID,
			Description: entry.TextDescription(),
			Content:     buf.String(),
			Author: &feeds.Author{
				Name:  s.c.User.Name,
				Email: s.c.User.Email,
			},
			Created: entry.Published,
			Updated: entry.Updated,
		})
	}

	var feedString, feedMediaType string

	switch feedType {
	case "rss":
		feedString, err = feed.ToRss()
		feedMediaType = contenttype.RSS
	case "atom":
		feedString, err = feed.ToAtom()
		feedMediaType = contenttype.ATOM
	case "json":
		feedString, err = feed.ToJSON()
		feedMediaType = contenttype.JSONFeed
	}

	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	w.Header().Set("Content-Type", feedMediaType+contenttype.CharsetUtf8Suffix)
	_, err = w.Write([]byte(feedString))
	if err != nil {
		s.n.Error(fmt.Errorf("serving feed %s to %s: %w", r.URL.Path, r.RemoteAddr, err))
	}
}
