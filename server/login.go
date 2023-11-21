package server

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/go-chi/jwtauth/v5"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"golang.org/x/crypto/bcrypt"
)

const (
	sessionSubject     string     = "Eagle Session 2"
	loggedInContextKey contextKey = "logged-in"

	loginPath  = "/panel/login"
	logoutPath = "/panel/logout"
)

type loginPage struct {
	Title string
	Error string
}

func (s *Server) loginGet(w http.ResponseWriter, r *http.Request) {
	if s.isLoggedIn(r) {
		http.Redirect(w, r, panelPath, http.StatusSeeOther)
		return
	}

	s.panelTemplate(w, r, http.StatusOK, panelLoginTemplate, &loginPage{
		Title: "Login",
	})
}

func (s *Server) loginPost(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		s.panelTemplate(w, r, http.StatusBadRequest, panelLoginTemplate, &loginPage{
			Title: "Login",
			Error: err.Error(),
		})
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")
	correctPassword := bcrypt.CompareHashAndPassword([]byte(s.c.Login.Password), []byte(password)) == nil

	if username != s.c.Login.Username || !correctPassword {
		s.panelTemplate(w, r, http.StatusUnauthorized, panelLoginTemplate, &loginPage{
			Title: "Login",
			Error: "Invalid credentials.",
		})
		return
	}

	expiration := time.Now().Add(time.Hour * 24 * 7)

	_, signed, err := s.jwtAuth.Encode(map[string]interface{}{
		jwt.SubjectKey:    sessionSubject,
		jwt.IssuedAtKey:   time.Now().Unix(),
		jwt.ExpirationKey: expiration,
	})
	if err != nil {
		s.panelTemplate(w, r, http.StatusInternalServerError, panelLoginTemplate, &loginPage{
			Title: "Login",
			Error: err.Error(),
		})
		return
	}

	cookie := &http.Cookie{
		Name:     "jwt",
		Value:    string(signed),
		Expires:  expiration,
		Secure:   r.URL.Scheme == "https",
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	}

	http.SetCookie(w, cookie)

	redirect := r.URL.Query().Get("redirect")
	if redirect == "" {
		redirect = "/"
	}

	http.Redirect(w, r, redirect, http.StatusSeeOther)
}

func (s *Server) logoutGet(w http.ResponseWriter, r *http.Request) {
	cookie := http.Cookie{
		Name:     "jwt",
		Value:    "",
		MaxAge:   -1,
		Secure:   r.URL.Scheme == "https",
		Path:     "/",
		HttpOnly: true,
	}
	http.SetCookie(w, &cookie)
	if redirect := r.URL.Query().Get("redirect"); redirect != "" {
		http.Redirect(w, r, redirect, http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func (s *Server) withLoggedIn(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, _, err := jwtauth.FromContext(r.Context())
		valid := !(err != nil || token == nil || jwt.Validate(token) != nil || token.Subject() != sessionSubject)
		ctx := r.Context()

		if valid {
			ctx = context.WithValue(r.Context(), loggedInContextKey, true)
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) mustLoggedIn(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.isLoggedIn(r) {
			newPath := loginPath + "?redirect=" + url.QueryEscape(r.URL.String())
			http.Redirect(w, r, newPath, http.StatusSeeOther)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) isLoggedIn(r *http.Request) bool {
	if loggedIn, ok := r.Context().Value(loggedInContextKey).(bool); ok {
		return loggedIn
	}

	return false
}
