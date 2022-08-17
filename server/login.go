package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/go-chi/jwtauth/v5"
	"github.com/hacdias/eagle/v4/eagle"
	"github.com/hacdias/eagle/v4/entry"
	"github.com/hacdias/indieauth/v2"
	"github.com/lestrrat-go/jwx/jwt"
)

const (
	SessionSubject string = "Eagle Session"

	OAuthSubject    string = "Eagle OAuth Client"
	OAuthCookieName string = "eagle-oauth"

	userContextKey contextKey = "user"
)

func (s *Server) loginGet(w http.ResponseWriter, r *http.Request) {
	if user := s.getUser(r); user != "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	s.serveHTMLWithStatus(w, r, &eagle.RenderData{
		Entry: &entry.Entry{
			Frontmatter: entry.Frontmatter{
				Title: "Login",
			},
		},
		NoIndex: true,
	}, []string{eagle.TemplateLogin}, http.StatusOK)
}

func (s *Server) saveAuthInfo(w http.ResponseWriter, r *http.Request, i *indieauth.AuthInfo) error {
	data, err := json.Marshal(i)
	if err != nil {
		return err
	}

	expiration := time.Now().Add(time.Minute * 10)

	_, signed, err := s.jwtAuth.Encode(map[string]interface{}{
		jwt.SubjectKey:    OAuthSubject,
		jwt.IssuedAtKey:   time.Now().Unix(),
		jwt.ExpirationKey: expiration,
		"data":            string(data),
		"redirect":        r.URL.Query().Get("redirect"),
	})
	if err != nil {
		return err
	}

	cookie := &http.Cookie{
		Name:     OAuthCookieName,
		Value:    string(signed),
		Expires:  expiration,
		Secure:   r.URL.Scheme == "https",
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	}

	http.SetCookie(w, cookie)
	return nil
}

func (s *Server) getInformation(w http.ResponseWriter, r *http.Request) (*indieauth.AuthInfo, string, error) {
	cookie, err := r.Cookie(OAuthCookieName)
	if err != nil {
		return nil, "", err
	}

	token, err := jwtauth.VerifyToken(s.jwtAuth, cookie.Value)
	if err != nil {
		return nil, "", err
	}

	err = jwt.Validate(token)
	if err != nil {
		return nil, "", err
	}

	if token.Subject() != OAuthSubject {
		return nil, "", errors.New("invalid subject for oauth token")
	}

	data, ok := token.Get("data")
	if !ok || data == nil {
		return nil, "", errors.New("cannot find 'data' property in token")
	}

	dataStr, ok := data.(string)
	if !ok || dataStr == "" {
		return nil, "", errors.New("cannot find 'data' property in token")
	}

	var i *indieauth.AuthInfo
	err = json.Unmarshal([]byte(dataStr), &i)
	if err != nil {
		return nil, "", err
	}

	// Delete cookie
	http.SetCookie(w, &http.Cookie{
		Name:     OAuthCookieName,
		MaxAge:   -1,
		Secure:   r.URL.Scheme == "https",
		Path:     "/",
		HttpOnly: true,
	})

	redirect, ok := token.Get("redirect")
	if !ok {
		return i, "", nil
	}

	redirectStr, ok := redirect.(string)
	if !ok || redirectStr == "" {
		return i, "", nil
	}

	return i, redirectStr, nil
}

func (s *Server) loginPost(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, err)
		return
	}

	profile := r.FormValue("profile")
	if profile == "" {
		s.serveErrorHTML(w, r, http.StatusBadRequest, errors.New("empty profile"))
		return
	}

	profile = indieauth.CanonicalizeURL(profile)
	if err := indieauth.IsValidProfileURL(profile); err != nil {
		err = fmt.Errorf("invalid profile url: %w", err)
		s.serveErrorHTML(w, r, http.StatusBadRequest, err)
		return
	}

	i, redirect, err := s.iac.Authenticate(profile, "")
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, err)
		return
	}

	err = s.saveAuthInfo(w, r, i)
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, err)
		return
	}

	http.Redirect(w, r, redirect, http.StatusSeeOther)
}

func (s *Server) loginCallbackGet(w http.ResponseWriter, r *http.Request) {
	i, redirect, err := s.getInformation(w, r)
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, err)
		return
	}

	code, err := s.iac.ValidateCallback(i, r)
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, err)
		return
	}

	profile, err := s.iac.FetchProfile(i, code)
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, err)
		return
	}

	if err := indieauth.IsValidProfileURL(profile.Me); err != nil {
		err = fmt.Errorf("invalid 'me': %w", err)
		s.serveErrorHTML(w, r, http.StatusBadRequest, err)
		return
	}

	expiration := time.Now().Add(time.Hour * 24 * 7)

	_, signed, err := s.jwtAuth.Encode(map[string]interface{}{
		jwt.SubjectKey:    SessionSubject,
		jwt.IssuedAtKey:   time.Now().Unix(),
		jwt.ExpirationKey: expiration,
		"user":            profile.Me,
	})
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, err)
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
		valid := !(err != nil || token == nil || jwt.Validate(token) != nil || token.Subject() != SessionSubject)
		ctx := r.Context()

		if valid {
			if userToken, ok := token.Get("user"); ok {
				if user, ok := userToken.(string); ok {
					ctx = context.WithValue(r.Context(), userContextKey, user)
				}
			}
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) mustLoggedIn(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if user := s.getUser(r); user == "" {
			newPath := "/login?redirect=" + url.QueryEscape(r.URL.String())
			http.Redirect(w, r, newPath, http.StatusSeeOther)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) mustAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.isAdmin(r) {
			s.serveErrorHTML(w, r, http.StatusForbidden, nil)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) getUser(r *http.Request) (me string) {
	if user, ok := r.Context().Value(userContextKey).(string); ok {
		return user
	}

	return ""
}

func (s *Server) isAdmin(r *http.Request) bool {
	user := s.getUser(r)
	return user == s.Config.ID()
}
