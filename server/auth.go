package server

import (
	"context"
	"net/http"
	urlpkg "net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/jwtauth/v5"
	"github.com/hacdias/eagle/eagle"
	"github.com/hacdias/eagle/renderer"
	"github.com/hacdias/indieauth/v3"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

// https://indieauth.spec.indieweb.org

const (
	authCodeSubject string = "Eagle Auth Code"
	tokenSubject    string = "Eagle Token"

	scopesContextKey contextKey = "scopes"
	clientContextKey contextKey = "client"
)

func (s *Server) indieauthGet(w http.ResponseWriter, r *http.Request) {
	s.serveJSON(w, http.StatusOK, map[string]interface{}{
		"issuer":                           s.c.ID(),
		"authorization_endpoint":           s.c.Server.AbsoluteURL("/auth"),
		"token_endpoint":                   s.c.Server.AbsoluteURL("/token"),
		"introspection_endpoint":           s.c.Server.AbsoluteURL("/token/verify"),
		"userinfo_endpoint":                s.c.Server.AbsoluteURL("/userinfo"),
		"code_challenge_methods_supported": indieauth.CodeChallengeMethods,
		"grant_types_supported":            []string{"authorization_code"},
		"response_types_supported":         []string{"code"},
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

	s.serveHTML(w, r, &renderer.RenderData{
		Entry: &eagle.Entry{
			FrontMatter: eagle.FrontMatter{
				Title: "Authorization",
			},
		},
		Data:    req,
		NoIndex: true,
	}, []string{renderer.TemplateAuth})
}

func (s *Server) authPost(w http.ResponseWriter, r *http.Request) {
	s.authorizationCodeExchange(w, r, false)
}

func (s *Server) authAcceptPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, err)
		return
	}

	req, err := s.ias.ParseAuthorization(r)
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, err)
		return
	}

	_, signed, err := s.jwtAuth.Encode(map[string]interface{}{
		jwt.SubjectKey:          authCodeSubject,
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
	query.Set("iss", s.c.ID())

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
}

type tokenUser struct {
	Name  string `json:"name,omitempty"`
	URL   string `json:"url,omitempty"`
	Photo string `json:"photo,omitempty"`
	Email string `json:"email,omitempty"`
}

func (s *Server) tokenGet(w http.ResponseWriter, r *http.Request) {
	// NOTE: this is kept for backwards compatibility with prior versions of IndieAuth
	// - Old Access Token Verifications: https://indieauth.spec.indieweb.org/20201126/#access-token-verification
	// - New Access Token Verifications: https://indieauth.spec.indieweb.org/#access-token-verification
	s.serveJSON(w, http.StatusOK, &tokenResponse{
		Me:       s.c.ID(),
		Scope:    strings.Join(s.getScopes(r), " "),
		ClientID: s.getClient(r),
	})
}

func (s *Server) tokenPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.serveErrorJSON(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	if r.Form.Get("grant_type") == "refresh_token" {
		// TODO: implement refresh token: https://indieauth.spec.indieweb.org/#refresh-tokens
		w.WriteHeader(http.StatusNotImplemented)
		return
	}

	if r.Form.Get("action") == "revoke" {
		// NOTE: this is kept for backwards compatibility with prior versions
		// of IndieAuth specification. Revocation endpoints are now separate.
		w.WriteHeader(http.StatusOK)
		return
	}

	s.authorizationCodeExchange(w, r, true)
}

func (s *Server) tokenVerifyPost(w http.ResponseWriter, r *http.Request) {
	token, _, err := jwtauth.FromContext(r.Context())
	isValid := !(err != nil || token == nil || jwt.Validate(token) != nil || token.Subject() != tokenSubject)
	if !isValid {
		s.serveJSON(w, http.StatusOK, map[string]interface{}{
			"active": false,
		})
		return
	}

	info := map[string]interface{}{
		"active":    true,
		"me":        s.c.ID(),
		"client_id": getString(token, "client_id"),
		"scope":     getString(token, "scope"),
		"iat":       token.IssuedAt().Unix(),
	}

	exp := token.Expiration()
	if !exp.IsZero() && exp.Unix() != 0 {
		info["exp"] = exp.Unix()
	}

	s.serveJSON(w, http.StatusOK, info)
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

	if token.Subject() != authCodeSubject {
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
		Me: s.c.ID(),
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

	at.Profile = s.buildProfile(scope)
	s.serveJSON(w, http.StatusOK, at)
}

func (s *Server) userInfoGet(w http.ResponseWriter, r *http.Request) {
	if !s.checkScope(w, r, "profile") {
		return
	}

	scope := strings.Join(s.getScopes(r), " ")
	profile := s.buildProfile(scope)
	s.serveJSON(w, http.StatusOK, profile)
}

func (s *Server) buildProfile(scope string) *tokenUser {
	var profile *tokenUser

	if strings.Contains(scope, "profile") {
		profile = &tokenUser{
			Name:  s.c.User.Name,
			URL:   s.c.ID(),
			Photo: s.c.User.Photo,
		}
	}

	if strings.Contains(scope, "email") {
		if profile == nil {
			profile = &tokenUser{}
		}
		profile.Email = s.c.User.Email
	}

	return profile
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
		jwt.SubjectKey:  tokenSubject,
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
		isValid := !(err != nil || token == nil || jwt.Validate(token) != nil || token.Subject() != tokenSubject)
		if !isValid {
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
