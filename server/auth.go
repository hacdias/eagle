package server

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/go-chi/jwtauth"
	"github.com/lestrrat-go/jwx/jwt"
	"golang.org/x/crypto/bcrypt"
)

var authContextKey = "auth"

func (s *Server) loginGetHandler(w http.ResponseWriter, r *http.Request) {
	s.renderDashboard(w, "login", &dashboardData{IsLogin: true})
}

func (s *Server) loginPostHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		s.renderDashboard(w, "login", &dashboardData{Data: err.Error()})
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")
	correctPassword := bcrypt.CompareHashAndPassword([]byte(s.Config.Auth.Password), []byte(password)) == nil

	if username != s.Config.Auth.Username || !correctPassword {
		w.WriteHeader(http.StatusInternalServerError)
		s.renderDashboard(w, "login", &dashboardData{IsLogin: true, Data: "wrong credentials"})
		return
	}

	expiration := time.Now().Add(time.Hour * 24 * 7)

	_, signed, err := s.token.Encode(map[string]interface{}{
		jwt.SubjectKey:    "Eagle",
		jwt.IssuedAtKey:   time.Now().Unix(),
		jwt.ExpirationKey: expiration,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		s.renderDashboard(w, "login", &dashboardData{IsLogin: true, Data: err.Error()})
		return
	}

	cookie := &http.Cookie{
		Name:     "jwt",
		Value:    string(signed),
		Expires:  expiration,
		Secure:   r.URL.Scheme == "https",
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteStrictMode,
	}

	http.SetCookie(w, cookie)
	redirectTo := "/"
	if r.URL.Query().Get("redirect") != "" {
		redirectTo = r.URL.Query().Get("redirect")
	}
	http.Redirect(w, r, redirectTo, http.StatusSeeOther)
}

func (s *Server) logoutGetHandler(w http.ResponseWriter, r *http.Request) {
	cookie := http.Cookie{
		Name:     "jwt",
		Value:    "",
		MaxAge:   0,
		Secure:   r.URL.Scheme == "https",
		Path:     "/",
		HttpOnly: true,
	}
	http.SetCookie(w, &cookie)
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func (s *Server) isAuthenticated(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var isAuthd bool

		if s.Config.Auth != nil {
			token, _, err := jwtauth.FromContext(r.Context())
			isAuthd = !(err != nil || token == nil || jwt.Validate(token) != nil)
		} else {
			isAuthd = true
		}

		ctx := context.WithValue(r.Context(), &authContextKey, isAuthd)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) mustAuthenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		isAuthd := r.Context().Value(&authContextKey).(bool)
		if !isAuthd {
			newPath := "/login?redirect=" + url.PathEscape(r.URL.String())
			http.Redirect(w, r, newPath, http.StatusTemporaryRedirect)
			return
		}

		next.ServeHTTP(w, r)
	})
}
