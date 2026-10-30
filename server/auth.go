package server

import (
	"context"
	"errors"
	"net/http"
	urlpkg "net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"go.hacdias.com/eagle/core"
	"go.hacdias.com/indielib/indieauth"
	"gorm.io/gorm"
)

// https://indieauth.spec.indieweb.org

const (
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
		"issuer":                                s.c.ID(),
		"authorization_endpoint":                s.c.AbsoluteURL(authPath),
		"token_endpoint":                        s.c.AbsoluteURL(tokenPath),
		"introspection_endpoint":                s.c.AbsoluteURL(tokenVerifyPath),
		"userinfo_endpoint":                     s.c.AbsoluteURL(userInfoPath),
		"code_challenge_methods_supported":      indieauth.CodeChallengeMethods,
		"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
		"response_types_supported":              []string{"code"},
		"token_endpoint_auth_methods_supported": []string{"none"},
		"client_id_metadata_document_supported": true,
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

	lifetime, err := handleExpiry(r.Form.Get("expiry"))
	if err != nil {
		s.panelError(w, r, http.StatusBadRequest, errors.New("expiry param is not a positive number"))
		return
	}

	code := uuid.New().String()
	err = s.core.DB().CreateAuthCode(r.Context(), &core.AuthCode{
		ID:                  core.HashToken(code),
		ClientID:            req.ClientID,
		RedirectURI:         req.RedirectURI,
		Scope:               strings.Join(req.Scopes, " "),
		CodeChallenge:       req.CodeChallenge,
		CodeChallengeMethod: req.CodeChallengeMethod,
		Lifetime:            lifetime,
		Expiry:              time.Now().Add(time.Minute * 5),
	})
	if err != nil {
		s.panelError(w, r, http.StatusInternalServerError, err)
		return
	}

	// The redirect URI has been validated by ParseAuthorization. It may already
	// have a query.
	redirect, _ := urlpkg.Parse(req.RedirectURI)
	query := redirect.Query()
	query.Set("code", code)
	query.Set("state", req.State)
	query.Set("iss", s.c.ID())
	redirect.RawQuery = query.Encode()

	http.Redirect(w, r, redirect.String(), http.StatusFound)
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
			_ = s.core.DB().DeleteToken(r.Context(), core.HashToken(tokenID))
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

	token, err := s.getToken(r.Context(), tokenID, core.TokenTypeAccess)
	if err != nil {
		s.serveJSON(w, http.StatusOK, map[string]any{"active": false})
		return
	}

	info := map[string]any{
		"active":    true,
		"me":        s.c.ID(),
		"client_id": token.ClientID,
		"scope":     token.Scope,
		"iat":       token.Created.Unix(),
		"exp":       token.Expiry.Unix(),
	}

	s.serveJSON(w, http.StatusOK, info)
}

func (s *Server) authorizationCodeExchange(w http.ResponseWriter, r *http.Request, withToken bool) {
	if err := r.ParseForm(); err != nil {
		s.serveErrorJSON(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	// The code is consumed before validating the exchange, so that a failed
	// attempt also invalidates it.
	code, err := s.core.DB().ConsumeAuthCode(r.Context(), core.HashToken(r.Form.Get("code")))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		s.serveErrorJSON(w, http.StatusBadRequest, "invalid_grant", "invalid code")
		return
	} else if err != nil {
		s.serveErrorJSON(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}

	if time.Now().After(code.Expiry) {
		s.serveErrorJSON(w, http.StatusBadRequest, "invalid_grant", "code expired")
		return
	}

	authRequest := &indieauth.AuthenticationRequest{
		ClientID:            code.ClientID,
		RedirectURI:         code.RedirectURI,
		CodeChallenge:       code.CodeChallenge,
		CodeChallengeMethod: code.CodeChallengeMethod,
	}

	err = s.ias.ValidateTokenExchange(authRequest, r)
	if err != nil {
		s.serveErrorJSON(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	at := &tokenResponse{
		Me: s.c.ID(),
	}

	if withToken {
		accessToken, err := s.generateToken(r.Context(), authRequest.ClientID, code.Scope, code.Lifetime)
		if err != nil {
			s.serveErrorJSON(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		at.AccessToken = accessToken
		at.TokenType = "Bearer"
		at.ExpiresIn = int64(code.Lifetime.Seconds())
		at.Scope = code.Scope

		refreshToken, err := s.generateRefreshToken(r.Context(), authRequest.ClientID, code.Scope, code.Lifetime*2)
		if err != nil {
			s.serveErrorJSON(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		at.RefreshToken = refreshToken
	}

	at.Profile = s.buildProfile(code.Scope)
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
	days, err := strconv.Atoi(expiry)
	if err != nil {
		return 0, err
	}

	if days <= 0 {
		return 0, errors.New("expiry must be positive")
	}

	return time.Hour * 24 * time.Duration(days), nil
}

func (s *Server) generateToken(ctx context.Context, client, scope string, expiry time.Duration) (string, error) {
	id := uuid.New().String()
	return id, s.core.DB().CreateToken(ctx, &core.Token{
		ID:       core.HashToken(id),
		Type:     core.TokenTypeAccess,
		ClientID: client,
		Scope:    scope,
		Expiry:   time.Now().Add(expiry),
		Created:  time.Now(),
	})
}

func (s *Server) generateRefreshToken(ctx context.Context, client, scope string, expiry time.Duration) (string, error) {
	id := uuid.New().String()
	return id, s.core.DB().CreateToken(ctx, &core.Token{
		ID:       core.HashToken(id),
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

	rt, err := s.getToken(r.Context(), tokenID, core.TokenTypeRefresh)
	if errors.Is(err, errTokenExpired) {
		s.serveErrorJSON(w, http.StatusBadRequest, "invalid_grant", "refresh token expired")
		return
	} else if err != nil {
		s.serveErrorJSON(w, http.StatusBadRequest, "invalid_grant", "invalid refresh token")
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
		ExpiresIn:    int64(accessExpiry.Round(time.Second).Seconds()),
	})
}

var errTokenExpired = errors.New("token expired")

// getToken returns the token with the given raw value and type. Expired tokens
// are deleted, in which case errTokenExpired is returned.
func (s *Server) getToken(ctx context.Context, id string, tokenType core.TokenType) (*core.Token, error) {
	token, err := s.core.DB().GetToken(ctx, core.HashToken(id), tokenType)
	if err != nil {
		return nil, err
	}

	if time.Now().After(token.Expiry) {
		_ = s.core.DB().DeleteToken(ctx, token.ID)
		return nil, errTokenExpired
	}

	return token, nil
}

func bearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(auth, "Bearer ")
}

func (s *Server) mustAccessToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenID := bearerToken(r)
		if tokenID == "" {
			s.serveErrorJSON(w, http.StatusUnauthorized, "invalid_request", "invalid token")
			return
		}

		token, err := s.getToken(r.Context(), tokenID, core.TokenTypeAccess)
		if errors.Is(err, errTokenExpired) {
			s.serveErrorJSON(w, http.StatusUnauthorized, "invalid_request", "token expired")
			return
		} else if err != nil {
			s.serveErrorJSON(w, http.StatusUnauthorized, "invalid_request", "invalid token")
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
