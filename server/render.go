package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const (
	// User provided templates.
	searchTemplate string = "search.html"
	errorTemplate  string = "error.html"

	// Our templates.
	panelTemplate         string = "panel.html"
	panelErrorTemplate    string = "error.html"
	panelAuthTemplate     string = "authorization.html"
	panelLoginTemplate    string = "login.html"
	panelMentionsTemplate string = "mentions.html"
	panelTokensTemplate   string = "tokens.html"
	panelEditorTemplate   string = "editor.html"
	panelBrowserTemplate  string = "browser.html"
)

type errorPage struct {
	Title   string
	Status  int
	Message string
}

func (s *Server) serveErrorHTML(w http.ResponseWriter, r *http.Request, code int, err error) {
	data := &errorPage{
		Title:  fmt.Sprintf("%d %s", code, http.StatusText(code)),
		Status: code,
	}

	if err != nil && code < 500 {
		s.log.Error(err)
		data.Message = err.Error()
	}

	doc, err := s.getDocument("404.html")
	if err != nil {
		w.WriteHeader(code)
		_, _ = w.Write([]byte(data.Message))
		return
	}

	txt := doc.Find("title").Text()
	txt = strings.Replace(txt, "404 Page not found", data.Title, 1) // Hugo 404.html Title
	doc.Find("title").SetText(txt)
	s.renderDocument(w, r, doc, code, errorTemplate, data)
}

func (s *Server) getDocument(path string) (*goquery.Document, error) {
	fileContent, err := s.staticFs.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return goquery.NewDocumentFromReader(bytes.NewReader(fileContent))
}

func (s *Server) renderDocument(w http.ResponseWriter, r *http.Request, doc *goquery.Document, code int, template string, data any) {
	var buf bytes.Buffer
	err := s.templates.ExecuteTemplate(&buf, template, data)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		s.log.Errorw("failed to serve html", "url", r.URL.Path, "err", err)
		return
	}

	pageNode := doc.Find("eagle-page")
	pageNode.ReplaceWithHtml(buf.String())

	html, err := doc.Html()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		s.log.Errorw("failed to get document html", "url", r.URL.Path, "err", err)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(code)
	_, err = w.Write([]byte(html))
	if err != nil {
		s.log.Errorw("failed to write html", "url", r.URL.Path, "err", err)
	}
}

func (s *Server) panelError(w http.ResponseWriter, r *http.Request, code int, reqErr error) {
	data := &errorPage{
		Title:  fmt.Sprintf("%d %s", code, http.StatusText(code)),
		Status: code,
	}

	if reqErr != nil {
		s.log.Error(reqErr)
		data.Message = reqErr.Error()
	}

	s.panelTemplate(w, r, code, panelErrorTemplate, data)
}

func (s *Server) panelTemplate(w http.ResponseWriter, r *http.Request, code int, template string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(code)
	err := panelTemplates.ExecuteTemplate(w, template, data)
	if err != nil {
		s.log.Errorw("failed to execute template", "url", r.URL.Path, "err", err)
	}
}

func (s *Server) serveJSON(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		s.log.Errorw("failed to write JSON", "err", err)
	}
}

func (s *Server) serveErrorJSON(w http.ResponseWriter, code int, err, errDescription string) {
	s.serveJSON(w, code, map[string]string{
		"error":             err,
		"error_description": errDescription,
	})
}
