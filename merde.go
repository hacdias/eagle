package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-chi/chi"
)

type ctxKey struct{ id string }

func main() {
	r := chi.NewRouter()

	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r = r.WithContext(context.WithValue(r.Context(), ctxKey{"mw2"}, "mw2"))
			next.ServeHTTP(w, r)
		})
	})
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		chkMw2 := r.Context().Value(ctxKey{"mw2"}).(string)
		w.WriteHeader(404)
		w.Write([]byte(fmt.Sprintf("sub 404 %s", chkMw2)))
	})

	http.ListenAndServe(":4444", r)
}
