package server

import (
	"bytes"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"go.hacdias.com/eagle/core"
)

const (
	searchPath = "/search/"
)

type searchPage struct {
	Entries  core.Entries
	Query    string
	Previous string
	Next     string
}

func (s *Server) searchGet(w http.ResponseWriter, r *http.Request) {
	page := 0
	if v := r.URL.Query().Get("page"); v != "" {
		p, _ := strconv.Atoi(v)
		if p >= 0 {
			page = p
		}
	}

	data := &searchPage{
		Query: r.URL.Query().Get("query"),
	}

	if data.Query != "" {
		ee, err := s.meilisearch.Search(int64(page), int64(s.c.Site.Paginate), data.Query)
		if err != nil {
			s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
			return
		}

		rq := r.URL.Query()
		rq.Set("page", strconv.Itoa(page+1))
		if len(ee) == s.c.Site.Paginate {
			data.Next = r.URL.Path + "?" + rq.Encode()
		}

		if page != 0 {
			rq.Set("page", strconv.Itoa(page-1))
			data.Previous = r.URL.Path + "?" + rq.Encode()
		}

		data.Entries = ee
	}

	s.renderTemplateWithContent(w, r, "Search", "search.html", data)
}

func (s *Server) renderTemplateWithContent(w http.ResponseWriter, r *http.Request, title, template string, data interface{}) {
	fd, err := s.staticFs.ReadFile(filepath.Join("/eagle/", "index.html"))
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(fd))
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	var buf bytes.Buffer
	err = s.templates.ExecuteTemplate(&buf, template, data)
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	doc.Find("title").SetText(strings.Replace(doc.Find("title").Text(), "Eagle", title, 1))

	pageNode := doc.Find("eagle-page")
	pageNode.ReplaceWithHtml(buf.String())

	html, err := doc.Html()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		s.n.Error(fmt.Errorf("serving html for %s: %w", r.URL.Path, err))
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte(html))
	if err != nil {
		s.n.Error(fmt.Errorf("serving html for %s: %w", r.URL.Path, err))
	}
}
