package server

import (
	"context"
	"errors"
	"net/http"
	urlpkg "net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/jwtauth/v5"
	"github.com/hacdias/eagle/v2/eagle"
	"github.com/hacdias/eagle/v2/entry"
	"github.com/hacdias/indieauth"
	"github.com/lestrrat-go/jwx/jwt"
	"golang.org/x/crypto/bcrypt"
)

// https://indieauth.spec.indieweb.org

const (
	AuthCodeSubject string = "Eagle Auth Code"
	TokenSubject    string = "Eagle Token"

	scopesContextKey contextKey = "scopes"
	clientContextKey contextKey = "client"
)

func (s *Server) indieauthGet(w http.ResponseWriter, r *http.Request) {
	s.serveJSON(w, http.StatusOK, map[string]interface{}{
		"issuer":                 s.Config.ID(),
		"authorization_endpoint": s.AbsoluteURL("/auth"),
		"token_endpoint":         s.AbsoluteURL("/token"),
		// "introspection_endpoint":           "TODO",
		// "userinfo_endpoint":                "TODO",
		"code_challenge_methods_supported": indieauth.CodeChallengeMethods,
	})
}

func (s *Server) authGet(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, err)
		return
	}

	req, err := s.ias.ParseAuthorization(r)
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, err)
		return
	}

	s.serveHTML(w, r, &eagle.RenderData{
		Entry:   &entry.Entry{},
		Data:    req,
		NoIndex: true,
	}, []string{eagle.TemplateAuth})
}

func (s *Server) authPost(w http.ResponseWriter, r *http.Request) {
	s.authorizationCodeExchange(w, r, false)
}

func (s *Server) authAcceptPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, err)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")
	correctPassword := bcrypt.CompareHashAndPassword([]byte(s.Config.Auth.Password), []byte(password)) == nil

	if username != s.Config.Auth.Username || !correctPassword {
		s.serveErrorHTML(w, r, http.StatusUnauthorized, errors.New("wrong credentials"))
		return
	}

	req, err := s.ias.ParseAuthorization(r)
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, err)
		return
	}

	_, signed, err := s.jwtAuth.Encode(map[string]interface{}{
		jwt.SubjectKey:          AuthCodeSubject,
		jwt.IssuedAtKey:         time.Now().Unix(),
		jwt.ExpirationKey:       time.Now().Add(time.Minute * 5),
		"scope":                 strings.Join(req.Scopes, " "),
		"expiry":                r.Form.Get("expiry"),
		"client_id":             req.ClientID,
		"redirect_uri":          req.RedirectURI,
		"code_challenge":        req.CodeChallenge,
		"code_challenge_method": req.CodeChallengeMethod,
	})
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	query := urlpkg.Values{}
	query.Set("code", signed)
	query.Set("state", req.State)
	query.Set("iss", s.Config.ID())

	http.Redirect(w, r, req.RedirectURI+"?"+query.Encode(), http.StatusFound)
}

type tokenResponse struct {
	Me          string     `json:"me"`
	ClientID    string     `json:"client_id,omitempty"`
	AccessToken string     `json:"access_token,omitempty"`
	TokenType   string     `json:"token_type,omitempty"`
	Scope       string     `json:"scope,omitempty"`
	Profile     *tokenUser `json:"profile,omitempty"`
	ExpiresIn   int64      `json:"expires_in,omitempty"`
	// TODO: implement refresh token (https://indieauth.spec.indieweb.org/#access-token-response-li-2).
}

type tokenUser struct {
	Name  string `json:"name,omitempty"`
	URL   string `json:"url,omitempty"`
	Photo string `json:"photo,omitempty"`
	Email string `json:"email,omitempty"`
}

func (s *Server) tokenGet(w http.ResponseWriter, r *http.Request) {
	s.serveJSON(w, http.StatusOK, &tokenResponse{
		Me:       s.Config.ID(),
		Scope:    strings.Join(s.getScopes(r), " "),
		ClientID: s.getClient(r),
	})
}

func (s *Server) tokenPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.serveErrorJSON(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	if r.Form.Get("action") == "revoke" {
		// TODO: currently, tokens have one week validity, otherwise
		// specified during the authorization request.
		w.WriteHeader(http.StatusOK)
		return
	}

	s.authorizationCodeExchange(w, r, true)
}

func (s *Server) authorizationCodeExchange(w http.ResponseWriter, r *http.Request, withToken bool) {
	if err := r.ParseForm(); err != nil {
		s.serveErrorJSON(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	code := r.Form.Get("code")
	token, err := jwtauth.VerifyToken(s.jwtAuth, code)
	if err != nil {
		s.serveErrorJSON(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	if token.Subject() != AuthCodeSubject {
		s.serveErrorJSON(w, http.StatusBadRequest, "invalid_request", "token has invalid subject")
		return
	}

	authRequest := &indieauth.AuthenticationRequest{
		ClientID:            getString(token, "client_id"),
		RedirectURI:         getString(token, "redirect_uri"),
		CodeChallenge:       getString(token, "code_challenge"),
		CodeChallengeMethod: getString(token, "code_challenge_method"),
	}

	err = s.ias.ValidateTokenExchange(authRequest, r)
	if err != nil {
		s.serveErrorJSON(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	at := &tokenResponse{
		Me: s.Config.ID(),
	}

	scope := getString(token, "scope")

	if withToken {
		expiry, err := handleExpiry(getString(token, "expiry"))
		if err != nil {
			s.serveErrorJSON(w, http.StatusBadRequest, "invalid_request", "expiry param is not a valid number")
		}

		signed, err := s.generateToken(authRequest.ClientID, scope, expiry)
		if err != nil {
			s.serveErrorJSON(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		at.AccessToken = signed
		at.TokenType = "Bearer"
		at.ExpiresIn = int64(expiry.Seconds())
		at.Scope = scope
	}

	if strings.Contains(scope, "profile") {
		at.Profile = &tokenUser{
			Name:  s.Config.Me.Name,
			URL:   s.Config.ID(),
			Photo: s.Config.Me.Photo,
		}
	}

	if strings.Contains(scope, "email") {
		if at.Profile == nil {
			at.Profile = &tokenUser{}
		}
		at.Profile.Email = s.Config.Me.Email
	}

	s.serveJSON(w, http.StatusOK, at)
}

func handleExpiry(expiry string) (time.Duration, error) {
	if expiry == "" {
		expiry = "0"
	}

	days, err := strconv.Atoi(expiry)
	if err != nil {
		return 0, nil
	}

	return time.Hour * 24 * time.Duration(days), nil
}

func (s *Server) generateToken(client, scope string, expiry time.Duration) (string, error) {
	claims := map[string]interface{}{
		jwt.SubjectKey:  TokenSubject,
		jwt.IssuedAtKey: time.Now().Unix(),
		"client_id":     client,
		"scope":         scope,
	}

	if expiry > 0 {
		claims[jwt.ExpirationKey] = time.Now().Add(expiry)
	}

	_, signed, err := s.jwtAuth.Encode(claims)
	return signed, err
}

func getString(token jwt.Token, prop string) string {
	v, ok := token.Get(prop)
	if !ok {
		return ""
	}
	vv, ok := v.(string)
	if !ok {
		return ""
	}
	return vv
}

func (s *Server) mustIndieAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, _, err := jwtauth.FromContext(r.Context())
		isAuthd := !(err != nil || token == nil || jwt.Validate(token) != nil || token.Subject() != TokenSubject)
		if !isAuthd {
			s.serveErrorJSON(w, http.StatusUnauthorized, "invalid_request", "invalid token")
			return
		}

		scopes := strings.Split(getString(token, "scope"), " ")
		clientID := getString(token, "client_id")

		ctx := context.WithValue(r.Context(), scopesContextKey, scopes)
		ctx = context.WithValue(ctx, clientContextKey, clientID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) getScopes(r *http.Request) []string {
	if scopes, ok := r.Context().Value(scopesContextKey).([]string); ok {
		return scopes
	}

	return []string{}
}

func (s *Server) getClient(r *http.Request) string {
	if clientID, ok := r.Context().Value(clientContextKey).(string); ok {
		return clientID
	}

	return ""
}
