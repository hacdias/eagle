package server

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/go-chi/jwtauth"
	"github.com/hacdias/eagle/v2/eagle"
	"github.com/lestrrat-go/jwx/jwt"
	"golang.org/x/crypto/bcrypt"
)

const (
	loggedInContextKey contextKey = "auth"
)

func (s *Server) serveLoginPage(w http.ResponseWriter, r *http.Request, code int, message string) {
	s.serveHTMLWithStatus(w, r, &eagle.RenderData{
		Entry: &eagle.Entry{
			Content: message,
			Frontmatter: eagle.Frontmatter{
				Title: "Login",
			},
		},
	}, []string{eagle.TemplateLogin}, code)
}

func (s *Server) loginGetHandler(w http.ResponseWriter, r *http.Request) {
	if s.isLoggedIn(w, r) {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
	s.serveLoginPage(w, r, http.StatusOK, "")
}

func (s *Server) loginPostHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")
	correctPassword := bcrypt.CompareHashAndPassword([]byte(s.Config.Auth.Password), []byte(password)) == nil

	if username != s.Config.Auth.Username || !correctPassword {
		s.serveLoginPage(w, r, http.StatusUnauthorized, "Wrong credentials.")
		return
	}

	expiration := time.Now().Add(time.Hour * 24 * 7)
	// TODO: 2fa or push notification

	_, signed, err := s.token.Encode(map[string]interface{}{
		jwt.SubjectKey:    "Eagle",
		jwt.IssuedAtKey:   time.Now().Unix(),
		jwt.ExpirationKey: expiration,
	})
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
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

func (s *Server) withLoggedIn(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var isAuthd bool

		if s.Config.Auth != nil {
			token, _, err := jwtauth.FromContext(r.Context())
			isAuthd = !(err != nil || token == nil || jwt.Validate(token) != nil)
		} else {
			isAuthd = false
		}

		ctx := context.WithValue(r.Context(), loggedInContextKey, isAuthd)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) mustAuthenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		isAuthd := r.Context().Value(loggedInContextKey).(bool)
		if !isAuthd {
			newPath := "/login?redirect=" + url.PathEscape(r.URL.String())
			http.Redirect(w, r, newPath, http.StatusTemporaryRedirect)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) isLoggedIn(w http.ResponseWriter, r *http.Request) bool {
	if loggedIn, ok := r.Context().Value(loggedInContextKey).(bool); ok {
		return loggedIn
	}

	return false
}
