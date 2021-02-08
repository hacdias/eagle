package server

import (
	"fmt"
	"net/http"
)

func (s *Server) recoverer(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rvr := recover(); rvr != nil && rvr != http.ErrAbortHandler {
				s.Errorf("panic while serving: %s", rvr)
				s.NotifyError(fmt.Errorf(fmt.Sprint(rvr)))
				w.WriteHeader(http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}
