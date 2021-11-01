package server

import (
	"context"
	"net/http"
	"os"

	"github.com/hacdias/eagle/v2/eagle"
)

func (s *Server) withEntry(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		entry, err := s.GetEntry(r.URL.Path)
		if err == nil {
			ctx := context.WithValue(r.Context(), entryContextKey, entry)
			next.ServeHTTP(w, r.WithContext(ctx))
		} else if os.IsNotExist(err) {
			s.serveError(w, http.StatusNotFound, nil)
		} else {
			s.serveErrorJSON(w, http.StatusInternalServerError, err)
		}
	})
}

func (s *Server) getEntry(w http.ResponseWriter, r *http.Request) *eagle.Entry {
	return r.Context().Value(entryContextKey).(*eagle.Entry)
}

func (s *Server) entryHandler(w http.ResponseWriter, r *http.Request) {
	entry := s.getEntry(w, r)

	str, err := entry.String()
	if err != nil {
		s.serveErrorJSON(w, http.StatusInternalServerError, err)
		return
	}

	w.Write([]byte(`
<html>
<head>
<link rel=authorization_endpoint href=https://indieauth.com/auth>
<link rel=token_endpoint href=https://tokens.indieauth.com/token>
<link href="mailto:hacdias@gmail.com" rel="me">
<link rel=micropub href=/micropub>
</head>

<body><pre>` + str + `</pre>
</body>
	
</html>`))
}

// // notFoundResponseWriter wraps a Response Writer to capture 404 requests.
// // In case it is a 404 request, then we do not write the body.
// type notFoundResponseWriter struct {
// 	http.ResponseWriter
// 	status int
// }

// func (w *notFoundResponseWriter) WriteHeader(status int) {
// 	w.status = status
// 	if status != http.StatusNotFound {
// 		w.ResponseWriter.WriteHeader(status)
// 	}
// }

// func (w *notFoundResponseWriter) Write(p []byte) (int, error) {
// 	if w.status != http.StatusNotFound {
// 		return w.ResponseWriter.Write(p)
// 	}
// 	// Lie that we successfully written it
// 	return len(p), nil
// }

// type adminBarResponseWriter struct {
// 	http.ResponseWriter
// 	s *Server
// 	p string
// }

// func (w *adminBarResponseWriter) WriteHeader(status int) {
// 	if status == http.StatusOK && strings.Contains(w.Header().Get("Content-Type"), "text/html") {
// 		length, _ := strconv.Atoi(w.Header().Get("Content-Length"))
// 		html, err := w.s.renderAdminBar(w.p)
// 		if err == nil {
// 			length += len(html)
// 			w.Header().Set("Content-Length", strconv.Itoa(length))
// 			w.ResponseWriter.WriteHeader(status)
// 			_, err = w.Write(html)
// 			if err != nil {
// 				w.s.log.Warn("could not write admin bar", err)
// 			}
// 		} else {
// 			w.s.log.Warn("could not render admin bar", err)
// 		}

// 		return
// 	}

// 	w.ResponseWriter.WriteHeader(status)
// }
