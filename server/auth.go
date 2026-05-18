package server

import (
	"context"
	"net/http"
	urlpkg "net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/jwtauth/v5"
	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v4/jwt"
	"github.com/samber/lo"
	"go.hacdias.com/eagle/core"
	"go.hacdias.com/indielib/indieauth"
)

// https://indieauth.spec.indieweb.org

const (
	authCodeSubject string = "Eagle Auth Code"

	scopesContextKey contextKey = "scopes"
	clientContextKey contextKey = "client"

	wellKnownOAuthServer = "/.well-known/oauth-authorization-server"
	authPath             = "/auth"
	authAcceptPath       = authPath + "/accept"
	tokenPath            = "/token"
	tokenVerifyPath      = tokenPath + "/verify"
	userInfoPath         = "/userinfo"
)

func (s *Server) indieauthGet(w http.ResponseWriter, r *http.Request) {
	s.serveJSON(w, http.StatusOK, map[string]any{
		"issuer":                           s.c.ID(),
		"authorization_endpoint":           s.c.AbsoluteURL(authPath),
		"token_endpoint":                   s.c.AbsoluteURL(tokenPath),
		"introspection_endpoint":           s.c.AbsoluteURL(tokenVerifyPath),
		"userinfo_endpoint":                s.c.AbsoluteURL(userInfoPath),
		"code_challenge_methods_supported": indieauth.CodeChallengeMethods,
		"grant_types_supported":            []string{"authorization_code", "refresh_token"},
		"response_types_supported":         []string{"code"},
	})
}

type authPage struct {
	Title   string
	Request *indieauth.AuthenticationRequest
}

func (s *Server) authGet(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.panelError(w, r, http.StatusBadRequest, err)
		return
	}

	req, err := s.ias.ParseAuthorization(r)
	if err != nil {
		s.panelError(w, r, http.StatusBadRequest, err)
		return
	}

	s.panelTemplate(w, r, http.StatusOK, panelAuthTemplate, &authPage{
		Title:   "Authorization",
		Request: req,
	})
}

func (s *Server) authPost(w http.ResponseWriter, r *http.Request) {
	s.authorizationCodeExchange(w, r, false)
}

func (s *Server) authAcceptPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.panelError(w, r, http.StatusBadRequest, err)
		return
	}

	req, err := s.ias.ParseAuthorization(r)
	if err != nil {
		s.panelError(w, r, http.StatusBadRequest, err)
		return
	}

	_, signed, err := s.jwtAuth.Encode(map[string]any{
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
		s.panelError(w, r, http.StatusInternalServerError, err)
		return
	}

	query := urlpkg.Values{}
	query.Set("code", signed)
	query.Set("state", req.State)
	query.Set("iss", s.c.ID())

	http.Redirect(w, r, req.RedirectURI+"?"+query.Encode(), http.StatusFound)
}

type tokenResponse struct {
	Me           string     `json:"me"`
	ClientID     string     `json:"client_id,omitempty"`
	AccessToken  string     `json:"access_token,omitempty"`
	RefreshToken string     `json:"refresh_token,omitempty"`
	TokenType    string     `json:"token_type,omitempty"`
	Scope        string     `json:"scope,omitempty"`
	Profile      *tokenUser `json:"profile,omitempty"`
	ExpiresIn    int64      `json:"expires_in,omitempty"`
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
		s.refreshTokenGrant(w, r)
		return
	}

	if r.Form.Get("action") == "revoke" {
		// NOTE: this is kept for backwards compatibility with prior versions
		// of IndieAuth specification. Revocation endpoints are now separate.
		tokenID := r.Form.Get("token")
		if tokenID != "" {
			_ = s.core.DB().DeleteToken(r.Context(), tokenID)
		}
		w.WriteHeader(http.StatusOK)
		return
	}

	s.authorizationCodeExchange(w, r, true)
}

func (s *Server) tokenVerifyPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.serveJSON(w, http.StatusOK, map[string]any{"active": false})
		return
	}

	tokenID := r.Form.Get("token")
	if tokenID == "" {
		tokenID = bearerToken(r)
	}
	if tokenID == "" {
		s.serveJSON(w, http.StatusOK, map[string]any{"active": false})
		return
	}

	token, err := s.core.DB().GetToken(r.Context(), tokenID, core.TokenTypeAccess)
	if err != nil {
		s.serveJSON(w, http.StatusOK, map[string]any{"active": false})
		return
	}

	if !token.Expiry.IsZero() && token.Expiry.Before(time.Now()) {
		s.serveJSON(w, http.StatusOK, map[string]any{"active": false})
		return
	}

	info := map[string]any{
		"active":    true,
		"me":        s.c.ID(),
		"client_id": token.ClientID,
		"scope":     token.Scope,
		"iat":       token.Created.Unix(),
	}

	if !token.Expiry.IsZero() {
		info["exp"] = token.Expiry.Unix()
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

	if subject, _ := token.Subject(); subject != authCodeSubject {
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
			return
		}

		accessToken, err := s.generateToken(r.Context(), authRequest.ClientID, scope, expiry)
		if err != nil {
			s.serveErrorJSON(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		at.AccessToken = accessToken
		at.TokenType = "Bearer"
		at.ExpiresIn = int64(expiry.Seconds())
		at.Scope = scope

		if expiry > 0 {
			refreshToken, err := s.generateRefreshToken(r.Context(), authRequest.ClientID, scope, expiry*2)
			if err != nil {
				s.serveErrorJSON(w, http.StatusInternalServerError, "server_error", err.Error())
				return
			}

			at.RefreshToken = refreshToken
		}
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
			Name:  s.c.Site.Params.Author.Name,
			URL:   s.c.ID(),
			Photo: s.c.Site.Params.Author.Photo,
		}
	}

	if strings.Contains(scope, "email") {
		if profile == nil {
			profile = &tokenUser{}
		}
		profile.Email = s.c.Site.Params.Author.Email
	}

	return profile
}

func handleExpiry(expiry string) (time.Duration, error) {
	if expiry == "" {
		expiry = "0"
	}

	days, err := strconv.Atoi(expiry)
	if err != nil {
		return 0, err
	}

	return time.Hour * 24 * time.Duration(days), nil
}

func (s *Server) generateToken(ctx context.Context, client, scope string, expiry time.Duration) (string, error) {
	id := uuid.New().String()

	var expiresAt time.Time
	if expiry > 0 {
		expiresAt = time.Now().Add(expiry)
	}

	return id, s.core.DB().CreateToken(ctx, &core.Token{
		ID:       id,
		Type:     core.TokenTypeAccess,
		ClientID: client,
		Scope:    scope,
		Expiry:   expiresAt,
		Created:  time.Now(),
	})
}

func (s *Server) generateRefreshToken(ctx context.Context, client, scope string, expiry time.Duration) (string, error) {
	id := uuid.New().String()
	return id, s.core.DB().CreateToken(ctx, &core.Token{
		ID:       id,
		Type:     core.TokenTypeRefresh,
		ClientID: client,
		Scope:    scope,
		Expiry:   time.Now().Add(expiry),
		Created:  time.Now(),
	})
}

func (s *Server) refreshTokenGrant(w http.ResponseWriter, r *http.Request) {
	tokenID := r.Form.Get("refresh_token")
	if tokenID == "" {
		s.serveErrorJSON(w, http.StatusBadRequest, "invalid_request", "missing refresh_token")
		return
	}

	rt, err := s.core.DB().GetToken(r.Context(), tokenID, core.TokenTypeRefresh)
	if err != nil {
		s.serveErrorJSON(w, http.StatusBadRequest, "invalid_grant", "invalid refresh token")
		return
	}

	if rt.Expiry.Before(time.Now()) {
		_ = s.core.DB().DeleteToken(r.Context(), rt.ID)
		s.serveErrorJSON(w, http.StatusBadRequest, "invalid_grant", "refresh token expired")
		return
	}

	clientID := r.Form.Get("client_id")
	if clientID != "" && clientID != rt.ClientID {
		s.serveErrorJSON(w, http.StatusBadRequest, "invalid_grant", "client_id mismatch")
		return
	}

	scope := r.Form.Get("scope")
	if scope == "" {
		scope = rt.Scope
	} else {
		allowedScopes := strings.Fields(rt.Scope)
		for _, sc := range strings.Fields(scope) {
			if !lo.Contains(allowedScopes, sc) {
				s.serveErrorJSON(w, http.StatusBadRequest, "invalid_scope", "requested scope exceeds original scope")
				return
			}
		}
	}

	// Rotate the refresh token.
	if err := s.core.DB().DeleteToken(r.Context(), rt.ID); err != nil {
		s.serveErrorJSON(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}

	// Derive original access token duration: refresh expiry was set to 2x access expiry.
	accessExpiry := rt.Expiry.Sub(rt.Created) / 2

	accessToken, err := s.generateToken(r.Context(), rt.ClientID, scope, accessExpiry)
	if err != nil {
		s.serveErrorJSON(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}

	newRefreshToken, err := s.generateRefreshToken(r.Context(), rt.ClientID, scope, accessExpiry*2)
	if err != nil {
		s.serveErrorJSON(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}

	s.serveJSON(w, http.StatusOK, &tokenResponse{
		Me:           s.c.ID(),
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		TokenType:    "Bearer",
		Scope:        scope,
	})
}

func getString(token jwt.Token, prop string) string {
	if !token.Has(prop) {
		return ""
	}

	var v string

	err := token.Get(prop, &v)
	if err != nil {
		return ""
	}

	return v
}

func bearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(auth, "Bearer ")
}

func (s *Server) mustIndieAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenID := bearerToken(r)
		if tokenID == "" {
			s.serveErrorJSON(w, http.StatusUnauthorized, "invalid_request", "invalid token")
			return
		}

		token, err := s.core.DB().GetToken(r.Context(), tokenID, core.TokenTypeAccess)
		if err != nil {
			s.serveErrorJSON(w, http.StatusUnauthorized, "invalid_request", "invalid token")
			return
		}

		if !token.Expiry.IsZero() && token.Expiry.Before(time.Now()) {
			s.serveErrorJSON(w, http.StatusUnauthorized, "invalid_request", "token expired")
			return
		}

		ctx := context.WithValue(r.Context(), scopesContextKey, strings.Fields(token.Scope))
		ctx = context.WithValue(ctx, clientContextKey, token.ClientID)

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

func (s *Server) checkScope(w http.ResponseWriter, r *http.Request, scope string) bool {
	scopes := s.getScopes(r)
	if !lo.Contains(scopes, scope) {
		s.serveErrorJSON(w, http.StatusForbidden, "insufficient_scope", "Insufficient scope.")
		return false
	}

	return true
}
