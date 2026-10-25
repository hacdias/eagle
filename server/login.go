package server

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.hacdias.com/eagle/core"
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

	sessionID := uuid.New().String()
	session := &core.Token{
		ID:      core.HashToken(sessionID),
		Type:    core.TokenTypeSession,
		Expiry:  expiration,
		Created: time.Now(),
	}
	if err := s.core.DB().CreateToken(r.Context(), session); err != nil {
		s.panelTemplate(w, r, http.StatusInternalServerError, panelLoginTemplate, &loginPage{
			Title: "Login",
			Error: err.Error(),
		})
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionID,
		Expires:  expiration,
		Secure:   r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https",
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, safeRedirect(r), http.StatusSeeOther)
}

func (s *Server) logoutGet(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(sessionCookieName)
	if err == nil && cookie.Value != "" {
		_ = s.core.DB().DeleteToken(r.Context(), core.HashToken(cookie.Value))
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		MaxAge:   -1,
		Secure:   r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https",
		Path:     "/",
		HttpOnly: true,
	})

	http.Redirect(w, r, safeRedirect(r), http.StatusSeeOther)
}

// safeRedirect returns the "redirect" query parameter if it is a local path,
// or "/" otherwise, in order to prevent open redirects.
func safeRedirect(r *http.Request) string {
	redirect := r.URL.Query().Get("redirect")
	if _, err := url.Parse(redirect); err != nil ||
		!strings.HasPrefix(redirect, "/") ||
		strings.HasPrefix(redirect, "//") ||
		strings.Contains(redirect, `\`) {
		return "/"
	}

	return redirect
}

func (s *Server) withLoggedIn(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil || cookie.Value == "" {
			next.ServeHTTP(w, r)
			return
		}

		_, err = s.getToken(r.Context(), cookie.Value, core.TokenTypeSession)
		if err != nil {
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
