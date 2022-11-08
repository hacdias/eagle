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
	"github.com/hacdias/eagle/v4/database"
	"github.com/hacdias/eagle/v4/eagle"
	"github.com/hacdias/eagle/v4/entry"
	"github.com/hacdias/eagle/v4/pkg/contenttype"
	"github.com/hacdias/eagle/v4/util"
	"github.com/jlelse/feeds"
	"github.com/thoas/go-funk"
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

	slug := util.Slugify(tag)
	if slug != tag {
		http.Redirect(w, r, "/tags/"+slug, http.StatusSeeOther)
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
					Title:   title.String(),
					Listing: &entry.Listing{},
				},
			},
		},
		exec: func(opts *database.QueryOptions) ([]*entry.Entry, error) {
			return s.GetByDate(opts, year, month, day)
		},
	})
}

func (s *Server) emojisGet(w http.ResponseWriter, r *http.Request) {
	emojis, err := s.GetEmojis()
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	s.serveHTML(w, r, &eagle.RenderData{
		Entry: s.getListingEntryOrEmpty(r.URL.Path, "Emojis"),
		Data: listingPage{
			Terms: emojis,
		},
	}, []string{eagle.TemplateEmojis})
}

func (s *Server) tagsGet(w http.ResponseWriter, r *http.Request) {
	tags, err := s.GetTags()
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	s.serveHTML(w, r, &eagle.RenderData{
		Entry: s.getListingEntryOrEmpty(r.URL.Path, "Tags"),
		Data: listingPage{
			Terms: tags,
		},
	}, []string{eagle.TemplateTags})
}

func (s *Server) searchGet(w http.ResponseWriter, r *http.Request) {
	search := &database.SearchOptions{
		Query:    r.URL.Query().Get("query"),
		Sections: []string{},
		Tags:     []string{},
	}

	if r.URL.Query().Has("section") {
		search.Sections = r.URL.Query()["section"]
		search.Sections = funk.FilterString(search.Sections, func(s string) bool { return s != "" })
	}

	if r.URL.Query().Has("tag") {
		search.Tags = r.URL.Query()["tag"]
		search.Tags = funk.FilterString(search.Tags, func(s string) bool { return s != "" })
	}

	ee := s.getListingEntryOrEmpty(r.URL.Path, "Search")
	if search.Query == "" {
		s.serveHTML(w, r, &eagle.RenderData{
			Entry:   ee,
			NoIndex: true,
			Data: &listingPage{
				Search: search,
			},
		}, []string{eagle.TemplateSearch})
		return
	}

	s.listingGet(w, r, &listingSettings{
		rd: &eagle.RenderData{
			Entry:   ee,
			NoIndex: true,
		},
		lp: listingPage{
			Search: search,
		},
		exec: func(opts *database.QueryOptions) ([]*entry.Entry, error) {
			if s.isAdmin(r) {
				opts.WithDrafts = true
				opts.WithDeleted = true
				opts.Visibility = nil
			}

			return s.Search(opts, search)
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
			return s.GetPrivate(opts.Pagination, s.getUser(r))
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
			return s.GetDeleted(opts.Pagination)
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
			return s.GetDrafts(opts.Pagination)
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
			return s.GetUnlisted(opts.Pagination)
		},
	})
}

func (s *Server) getListingEntryOrEmpty(id, title string) *entry.Entry {
	id = strings.TrimSuffix(id, filepath.Ext(id))
	if ee, err := s.GetEntry(id); err == nil {
		if ee.Listing == nil {
			s.log.Warnf("entry %s should be marked as listing", ee.ID)
			ee.Listing = &entry.Listing{}
		}
		return ee
	}

	return &entry.Entry{
		ID: id,
		Frontmatter: entry.Frontmatter{
			Title:   title,
			Listing: &entry.Listing{},
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
	Search   *database.SearchOptions
	Entries  []*entry.Entry
	Page     int
	NextPage string
	Terms    []string
}

func (s *Server) listingGet(w http.ResponseWriter, r *http.Request, ls *listingSettings) {
	opts := &database.QueryOptions{}

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

	opts.OrderByUpdated = ls.rd.Entry.Listing.OrderByUpdated

	if !ls.rd.Entry.Listing.DisablePagination {
		opts.Pagination = &database.PaginationOptions{}

		if ls.rd.Entry.Listing.ItemsPerPage > 0 {
			opts.Pagination.Limit = ls.rd.Entry.Listing.ItemsPerPage
		} else {
			opts.Pagination.Limit = s.Config.Site.Pagination
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
		Title:       ls.rd.Entry.TextTitle(),
		Link:        &feeds.Link{Href: strings.TrimSuffix(s.AbsoluteURL(r.URL.Path), "."+feedType)},
		Description: ls.rd.Entry.TextDescription(),
		Author: &feeds.Author{
			Name:  s.Config.User.Name,
			Email: s.Config.User.Email,
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
			Title:       entry.TextTitle(),
			Link:        &feeds.Link{Href: entry.Permalink},
			Id:          entry.ID,
			Description: entry.TextDescription(),
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
