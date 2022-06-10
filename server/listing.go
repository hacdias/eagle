package server

import (
	"bytes"
	"fmt"
	"net/http"
	urlpkg "net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hacdias/eagle/v4/contenttype"
	"github.com/hacdias/eagle/v4/database"
	"github.com/hacdias/eagle/v4/eagle"
	"github.com/hacdias/eagle/v4/entry"
	"github.com/jlelse/feeds"
)

func (s *Server) allGet(w http.ResponseWriter, r *http.Request) {
	s.listingGet(w, r, &listingSettings{
		exec: func(opts *database.QueryOptions) ([]*entry.Entry, error) {
			return s.GetAll(opts)
		},
	})
}

func (s *Server) indexGet(w http.ResponseWriter, r *http.Request) {
	s.listingGet(w, r, &listingSettings{
		rd: &eagle.RenderData{
			IsHome: true,
		},
		exec: func(opts *database.QueryOptions) ([]*entry.Entry, error) {
			return s.GetBySection(opts, s.Config.Site.IndexSection)
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

	s.listingGet(w, r, &listingSettings{
		rd: &eagle.RenderData{
			Entry: s.getListingEntryOrEmpty(r.URL.Path, "#"+tag),
		},
		exec: func(opts *database.QueryOptions) ([]*entry.Entry, error) {
			return s.GetByTag(opts, tag)
		},
	})
}

func (s *Server) emojiGet(w http.ResponseWriter, r *http.Request) {
	emoji := chi.URLParam(r, "emoji")
	if emoji == "" {
		s.serveErrorHTML(w, r, http.StatusNotFound, nil)
		return
	}

	s.listingGet(w, r, &listingSettings{
		rd: &eagle.RenderData{
			Entry: s.getListingEntryOrEmpty(r.URL.Path, emoji),
		},
		exec: func(opts *database.QueryOptions) ([]*entry.Entry, error) {
			return s.GetByEmoji(opts, emoji)
		},
	})
}

func (s *Server) bookGet(w http.ResponseWriter, r *http.Request) {
	ee, err := s.GetEntry(r.URL.Path)
	if os.IsNotExist(err) {
		s.serveErrorHTML(w, r, http.StatusNotFound, nil)
		return
	}

	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	s.listingGet(w, r, &listingSettings{
		rd: &eagle.RenderData{
			Entry: ee,
		},
		exec: func(opts *database.QueryOptions) ([]*entry.Entry, error) {
			return s.GetByProperty(opts, "read-of", ee.ID)
		},
		templates: []string{eagle.TemplateBook},
	})
}

func (s *Server) sectionGet(section string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ee := s.getListingEntryOrEmpty(r.URL.Path, section)
		if len(ee.Sections) == 0 {
			ee.Sections = []string{section}
		}

		s.listingGet(w, r, &listingSettings{
			rd: &eagle.RenderData{
				Entry: ee,
			},
			exec: func(opts *database.QueryOptions) ([]*entry.Entry, error) {
				return s.GetBySection(opts, section)
			},
			templates: []string{},
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
			Entry: &entry.Entry{
				Frontmatter: entry.Frontmatter{
					Title:     title.String(),
					IsListing: true,
				},
			},
		},
		exec: func(opts *database.QueryOptions) ([]*entry.Entry, error) {
			return s.GetByDate(opts, year, month, day)
		},
	})
}

func (s *Server) searchGet(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")

	ee := s.getListingEntryOrEmpty(r.URL.Path, "Search")
	if ee.ID == "" {
		ee.ID = strings.TrimSuffix(r.URL.Path, filepath.Ext(r.URL.Path))
	}

	if query == "" {
		s.serveHTML(w, r, &eagle.RenderData{
			Entry: ee,
			Data:  &listingPage{},
		}, []string{eagle.TemplateSearch})
		return
	}

	s.listingGet(w, r, &listingSettings{
		rd: &eagle.RenderData{
			Entry: ee,
		},
		lp: listingPage{
			SearchQuery: query,
		},
		exec: func(opts *database.QueryOptions) ([]*entry.Entry, error) {
			if s.isAdmin(r) {
				opts.WithDrafts = true
				opts.WithDeleted = true
				opts.Visibility = nil
			}

			return s.Search(opts, query)
		},
		templates: []string{eagle.TemplateSearch},
	})
}

func (s *Server) privateGet(w http.ResponseWriter, r *http.Request) {
	s.listingGet(w, r, &listingSettings{
		rd: &eagle.RenderData{
			Entry:   s.getListingEntryOrEmpty(r.URL.Path, "Private"),
			NoIndex: true,
		},
		exec: func(opts *database.QueryOptions) ([]*entry.Entry, error) {
			return s.GetPrivate(&opts.PaginationOptions, s.getUser(r))
		},
	})
}

func (s *Server) deletedGet(w http.ResponseWriter, r *http.Request) {
	s.listingGet(w, r, &listingSettings{
		rd: &eagle.RenderData{
			Entry:   s.getListingEntryOrEmpty(r.URL.Path, "Deleted"),
			NoIndex: true,
		},
		exec: func(opts *database.QueryOptions) ([]*entry.Entry, error) {
			return s.GetDeleted(&opts.PaginationOptions)
		},
	})
}

func (s *Server) draftsGet(w http.ResponseWriter, r *http.Request) {
	s.listingGet(w, r, &listingSettings{
		rd: &eagle.RenderData{
			Entry:   s.getListingEntryOrEmpty(r.URL.Path, "Drafts"),
			NoIndex: true,
		},
		exec: func(opts *database.QueryOptions) ([]*entry.Entry, error) {
			return s.GetDrafts(&opts.PaginationOptions)
		},
	})
}

func (s *Server) unlistedGet(w http.ResponseWriter, r *http.Request) {
	s.listingGet(w, r, &listingSettings{
		rd: &eagle.RenderData{
			Entry:   s.getListingEntryOrEmpty(r.URL.Path, "Unlisted"),
			NoIndex: true,
		},
		exec: func(opts *database.QueryOptions) ([]*entry.Entry, error) {
			return s.GetUnlisted(&opts.PaginationOptions)
		},
	})
}

func (s *Server) getListingEntryOrEmpty(id, title string) *entry.Entry {
	id = strings.TrimSuffix(id, filepath.Ext(id))
	if ee, err := s.GetEntry(id); err == nil {
		if !ee.IsListing {
			s.log.Warnf("entry %s should be marked as listing", ee.ID)
			ee.IsListing = true
		}
		return ee
	}

	return &entry.Entry{
		Frontmatter: entry.Frontmatter{
			Title:     title,
			IsListing: true,
		},
	}
}

type listingSettings struct {
	exec      func(*database.QueryOptions) ([]*entry.Entry, error)
	rd        *eagle.RenderData
	lp        listingPage
	templates []string
}

type listingPage struct {
	SearchQuery string
	Entries     []*entry.Entry
	Page        int
	NextPage    string
	Terms       []string
}

func (s *Server) listingGet(w http.ResponseWriter, r *http.Request, ls *listingSettings) {
	opts := &database.QueryOptions{
		PaginationOptions: database.PaginationOptions{
			Limit: s.Config.Site.Paginate,
		},
	}

	user := s.getUser(r)
	if user == "" {
		opts.Visibility = []entry.Visibility{entry.VisibilityPublic}
	} else {
		opts.Visibility = []entry.Visibility{entry.VisibilityPublic, entry.VisibilityPrivate}
		if !s.isAdmin(r) {
			opts.Audience = s.getUser(r)
		}
	}

	if ls.rd == nil {
		ls.rd = &eagle.RenderData{}
	}

	if ls.rd.Entry == nil {
		ls.rd.Entry = s.getListingEntryOrEmpty(r.URL.Path, "")
	}

	if v := r.URL.Query().Get("page"); v != "" {
		vv, _ := strconv.Atoi(v)
		if vv >= 0 {
			opts.Page = vv
			ls.lp.Page = vv
		}
	}

	entries, err := ls.exec(opts)
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	ls.lp.Entries = entries

	if len(entries) != 0 {
		url, _ := urlpkg.Parse(r.URL.String())
		values := url.Query()
		values.Set("page", strconv.Itoa(opts.Page+1))
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
		templates = append(templates, eagle.TemplateList)
		path := r.URL.Path

		ls.rd.Alternates = []eagle.Alternate{
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
		Title:       ls.rd.Entry.DisplayTitle(),
		Link:        &feeds.Link{Href: strings.TrimSuffix(s.AbsoluteURL(r.URL.Path), "."+feedType)},
		Description: ls.rd.Entry.DisplayDescription(),
		Author: &feeds.Author{
			Name:  s.Config.Me.Name,
			Email: s.Config.Me.Email,
		},
		// TODO: support .Tags
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
			Title:       entry.DisplayTitle(),
			Link:        &feeds.Link{Href: entry.Permalink},
			Id:          entry.ID,
			Description: entry.DisplayDescription(),
			Content:     buf.String(),
			Author: &feeds.Author{
				Name:  s.Config.Me.Name,
				Email: s.Config.Me.Email,
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
		s.Notifier.Error(fmt.Errorf("error while serving feed: %w", err))
	}
}
