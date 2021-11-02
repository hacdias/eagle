package server

import (
	"net/http"
	// urlpkg "net/url"

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
