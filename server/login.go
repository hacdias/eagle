package server

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
	"go.hacdias.com/eagle/services/database"
	"golang.org/x/crypto/bcrypt"
)

const (
	sessionCookieName             = "session"
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

	session := &database.Session{
		ID:      uuid.New().String(),
		Expiry:  expiration,
		Created: time.Now(),
	}
	if err := s.bolt.AddSession(r.Context(), session); err != nil {
		s.panelTemplate(w, r, http.StatusInternalServerError, panelLoginTemplate, &loginPage{
			Title: "Login",
			Error: err.Error(),
		})
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    session.ID,
		Expires:  expiration,
		Secure:   r.URL.Scheme == "https",
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	})

	redirect := r.URL.Query().Get("redirect")
	if redirect == "" {
		redirect = "/"
	}

	http.Redirect(w, r, redirect, http.StatusSeeOther)
}

func (s *Server) logoutGet(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(sessionCookieName)
	if err == nil && cookie.Value != "" {
		_ = s.bolt.DeleteSession(r.Context(), cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		MaxAge:   -1,
		Secure:   r.URL.Scheme == "https",
		Path:     "/",
		HttpOnly: true,
	})

	redirect := r.URL.Query().Get("redirect")
	if redirect == "" {
		redirect = "/"
	}

	http.Redirect(w, r, redirect, http.StatusSeeOther)
}

func (s *Server) withLoggedIn(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil || cookie.Value == "" {
			next.ServeHTTP(w, r)
			return
		}

		session, err := s.bolt.GetSession(r.Context(), cookie.Value)
		if err != nil || session == nil {
			next.ServeHTTP(w, r)
			return
		}

		if time.Now().After(session.Expiry) {
			_ = s.bolt.DeleteSession(r.Context(), cookie.Value)
			next.ServeHTTP(w, r)
			return
		}

		ctx := context.WithValue(r.Context(), loggedInContextKey, true)
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
