package server

import (
	"net/http"
	"strconv"

	"github.com/hacdias/eagle/eagle"
	"github.com/hacdias/eagle/indexer"
	"github.com/hacdias/eagle/renderer"
)

func (s *Server) searchGet(w http.ResponseWriter, r *http.Request) {
	// search := &indexer.Search{
	// 	Query:    r.URL.Query().Get("query"),
	// 	Sections: []string{},
	// }

	// if r.URL.Query().Has("section") {
	// 	search.Sections = r.URL.Query()["section"]
	// 	search.Sections = lo.Filter(search.Sections, func(s string, _ int) bool {
	// 		return s != ""
	// 	})
	// }

	// e := s.getListingEntryOrEmpty(r.URL.Path, "Search")
	// if search.Query == "" {
	// 	s.serveHTML(w, r, &renderer.RenderData{
	// 		Entry:   e,
	// 		NoIndex: true,
	// 		Data: &listingPage{
	// 			Search: search,
	// 		},
	// 	}, []string{renderer.TemplateSearch})
	// 	return
	// }

	// s.listingGet(w, r, &listingSettings{
	// 	noFeed: true,
	// 	rd: &renderer.RenderData{
	// 		Entry:   e,
	// 		NoIndex: true,
	// 	},
	// 	lp: listingPage{
	// 		Search: search,
	// 	},
	// 	exec: func(opts *indexer.Query) (eagle.Entries, error) {
	// 		if s.isLoggedIn(r) {
	// 			opts.WithDrafts = true
	// 			opts.WithDeleted = true
	// 			opts.WithUnlisted = true
	// 		}

	// 		return s.i.GetSearch(opts, search)
	// 	},
	// 	templates: []string{renderer.TemplateSearch},
	// })
}

func (s *Server) draftsGet(w http.ResponseWriter, r *http.Request) {
	// s.listingGet(w, r, &listingSettings{
	// 	noFeed: true,
	// 	rd: &renderer.RenderData{
	// 		Entry:   s.getListingEntryOrEmpty(r.URL.Path, "Drafts"),
	// 		NoIndex: true,
	// 	},
	// 	exec: func(opts *indexer.Query) (eagle.Entries, error) {
	// 		return s.i.GetDrafts(opts.Pagination)
	// 	},
	// })
}

func (s *Server) unlistedGet(w http.ResponseWriter, r *http.Request) {
	// s.listingGet(w, r, &listingSettings{
	// 	noFeed: true,
	// 	rd: &renderer.RenderData{
	// 		Entry:   s.getListingEntryOrEmpty(r.URL.Path, "Unlisted"),
	// 		NoIndex: true,
	// 	},
	// 	exec: func(opts *indexer.Query) (eagle.Entries, error) {
	// 		return s.i.GetUnlisted(opts.Pagination)
	// 	},
	// })
}

func (s *Server) deletedGet(w http.ResponseWriter, r *http.Request) {
	// s.listingGet(w, r, &listingSettings{
	// 	noFeed: true,
	// 	rd: &renderer.RenderData{
	// 		Entry:   s.getListingEntryOrEmpty(r.URL.Path, "Deleted"),
	// 		NoIndex: true,
	// 	},
	// 	exec: func(opts *indexer.Query) (eagle.Entries, error) {
	// 		return s.i.GetDeleted(opts.Pagination)
	// 	},
	// })
}

// func (s *Server) getListingEntryOrEmpty(id, title string) *eagle.Entry {
// 	id = strings.TrimSuffix(id, filepath.Ext(id))
// 	if e, err := s.fs.GetEntry(id); err == nil {
// 		if e.Listing == nil {
// 			s.log.Warnf("entry %s should be marked as listing", e.ID)
// 			e.Listing = &eagle.Listing{}
// 		}
// 		return e
// 	}

// 	return &eagle.Entry{
// 		ID: id,
// 		FrontMatter: eagle.FrontMatter{
// 			Title:   title,
// 			Listing: &eagle.Listing{},
// 		},
// 	}
// }

type listingSettings struct {
	exec      func(*indexer.Query) (eagle.Entries, error)
	rd        *renderer.RenderData
	lp        listingPage
	templates []string
	noFeed    bool
}

type listingPage struct {
	Search       *indexer.Search
	Taxonomy     string
	Terms        eagle.Terms
	Entries      eagle.Entries
	Page         int
	PreviousPage string
	NextPage     string
}

func (s *Server) listingQuery(r *http.Request, ls *listingSettings) *indexer.Query {
	opts := &indexer.Query{
		OrderByUpdated: ls.rd.Entry.Listing.OrderByUpdated,
	}

	if ls.rd.Listing.DisablePagination {
		return opts
	}

	opts.Pagination = &indexer.Pagination{}

	if ls.rd.Entry.Listing.ItemsPerPage > 0 {
		opts.Pagination.Limit = ls.rd.Entry.Listing.ItemsPerPage
	} else {
		opts.Pagination.Limit = s.c.Site.Pagination
	}

	if v := r.URL.Query().Get("page"); v != "" {
		p, _ := strconv.Atoi(v)
		if p >= 0 {
			opts.Pagination.Page = p
			ls.lp.Page = p
		}
	}

	return opts
}

// func (s *Server) listingGet(w http.ResponseWriter, r *http.Request, ls *listingSettings) {
// 	if ls.rd == nil {
// 		ls.rd = &renderer.RenderData{}
// 	}

// 	if ls.rd.Entry == nil {
// 		ls.rd.Entry = s.getListingEntryOrEmpty(r.URL.Path, "")
// 	}

// 	query := s.listingQuery(r, ls)
// 	ee, err := ls.exec(query)
// 	if err != nil {
// 		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
// 		return
// 	}

// 	ls.lp.Entries = ee

// 	if !ls.rd.Entry.Listing.DisablePagination {
// 		url, _ := urlpkg.Parse(r.URL.String())
// 		values := url.Query()

// 		if query.Pagination.Page > 0 {
// 			values.Set("page", strconv.Itoa(query.Pagination.Page-1))
// 			url.RawQuery = values.Encode()
// 			ls.lp.PreviousPage = url.String()
// 		}

// 		if len(ee) > 0 {
// 			values.Set("page", strconv.Itoa(query.Pagination.Page+1))
// 			url.RawQuery = values.Encode()
// 			ls.lp.NextPage = url.String()
// 		}
// 	}

// 	ls.rd.Data = ls.lp

// 	templates := ls.templates
// 	if ls.rd.Template != "" {
// 		templates = append(templates, ls.rd.Template)
// 	}
// 	templates = append(templates, renderer.TemplateList)
// 	path := r.URL.Path

// 	if !ls.noFeed {
// 		ls.rd.Alternates = []renderer.Alternate{
// 			{
// 				Type: contenttype.JSONFeed,
// 				Href: path + ".json",
// 			},
// 			{
// 				Type: contenttype.ATOM,
// 				Href: path + ".atom",
// 			},
// 			{
// 				Type: contenttype.RSS,
// 				Href: path + ".rss",
// 			},
// 		}
// 	}

// 	s.serveHTML(w, r, ls.rd, templates)
// }
