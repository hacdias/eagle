package server

import (
	"context"
	"errors"
	"net/http"
	urlpkg "net/url"
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

	http.Redirect(w, r, req.RedirectURI+"?"+query.Encode(), http.StatusFound)
}

type tokenResponse struct {
	Me          string     `json:"me"`
	ClientID    string     `json:"client_id,omitempty"`
	AccessToken string     `json:"access_token,omitempty"`
	TokenType   string     `json:"token_type,omitempty"`
	Scope       string     `json:"scope,omitempty"`
	Profile     *tokenUser `json:"profile,omitempty"`
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

	var (
		grantType = r.Form.Get("grant_type")
		code      = r.Form.Get("code")
	)

	if grantType == "" {
		// Default to support legacy clients.
		grantType = "authorization_code"
	}

	if grantType != "authorization_code" {
		s.serveErrorJSON(w, http.StatusBadRequest, "invalid_request", "grant_type must be authorization_code")
		return
	}

	token, err := jwtauth.VerifyToken(s.jwtAuth, code)
	if err != nil {
		s.serveErrorJSON(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	if token.Subject() != AuthCodeSubject {
		s.serveErrorJSON(w, http.StatusBadRequest, "invalid_request", "token has invalid subject")
		return
	}

	err = validateAuthorizationCode(r, token)
	if err != nil {
		s.serveErrorJSON(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	at := &tokenResponse{
		Me: s.Config.ID(),
	}

	scope := getString(token, "scope")

	if withToken {
		clientID := getString(token, "client_id")
		expires := getString(token, "expiry") != "infinity"
		signed, err := s.generateToken(clientID, scope, expires)
		if err != nil {
			s.serveErrorJSON(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		at.AccessToken = signed
		at.TokenType = "Bearer"
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

func (s *Server) generateToken(client, scope string, expires bool) (string, error) {
	claims := map[string]interface{}{
		jwt.SubjectKey:  TokenSubject,
		jwt.IssuedAtKey: time.Now().Unix(),
		"client_id":     client,
		"scope":         scope,
	}

	if expires {
		claims[jwt.ExpirationKey] = time.Now().Add(time.Hour * 24 * 7)
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

func validateAuthorizationCode(r *http.Request, token jwt.Token) error {
	var (
		clientID     = r.Form.Get("client_id")
		redirectURI  = r.Form.Get("redirect_uri")
		codeVerifier = r.Form.Get("code_verifier")
	)

	if getString(token, "client_id") != clientID {
		return errors.New("client_id differs")
	}

	if getString(token, "redirect_uri") != redirectURI {
		return errors.New("redirect_uri differs")
	}

	cc := getString(token, "code_challenge")
	if cc != "" {
		ccm := getString(token, "code_challenge_method")
		if cc == "" {
			return errors.New("code_challenge_method missing from token")
		}

		if !indieauth.IsValidCodeChallengeMethod(ccm) {
			return errors.New("code_challenge_method invalid")
		}

		if !indieauth.ValidateCodeChallenge(ccm, cc, codeVerifier) {
			return errors.New("code challenge failed")
		}
	}

	return nil
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
