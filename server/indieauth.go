package server

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	urlpkg "net/url"
	"strings"
	"time"

	"github.com/go-chi/jwtauth/v5"
	"github.com/hacdias/eagle/v2/eagle"
	"github.com/hacdias/eagle/v2/entry"
	"github.com/lestrrat-go/jwx/jwt"
	"github.com/thoas/go-funk"
)

// https://indieauth.spec.indieweb.org

const (
	AuthCodeSubject string = "Eagle Auth Code"
	TokenSubject    string = "Eagle Token"
)

const (
	scopesContextKey contextKey = "scopes"
)

func (s *Server) indieauthGet(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, err)
		return
	}

	resType := r.FormValue("response_type")
	if resType == "" {
		// Default to 'code' to support old clients.
		resType = "code"
	}

	if resType != "code" {
		s.serveErrorHTML(w, r, http.StatusBadRequest, errors.New("response_type must be code"))
		return
	}

	req, err := getAuthorizationRequest(r)
	if err != nil {
		s.serveErrorJSON(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	s.serveHTML(w, r, &eagle.RenderData{
		Entry:   &entry.Entry{},
		Data:    req,
		NoIndex: true,
	}, []string{eagle.TemplateAuth})
}

func (s *Server) indieauthPost(w http.ResponseWriter, r *http.Request) {
	s.authorizationCodeExchange(w, r, false)
}

func (s *Server) indieauthAcceptPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, err)
		return
	}

	req, err := getAuthorizationRequest(r)
	if err != nil {
		s.serveErrorJSON(w, http.StatusBadRequest, "invalid_request", err.Error())
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

	err = getRequestInfoFromToken(r, token)
	if err != nil {
		s.serveErrorJSON(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	at := &accessToken{
		Me: s.Config.Site.BaseURL + "/",
	}

	scope := getString(token, "scope")

	if withToken {
		claims := map[string]interface{}{
			jwt.SubjectKey:  TokenSubject,
			jwt.IssuedAtKey: time.Now().Unix(),
			"scope":         scope,
		}

		// By default, tokens expire in a week, unless specified otherwise.
		if getString(token, "expiry") != "infinity" {
			claims[jwt.ExpirationKey] = time.Now().Add(time.Hour * 24 * 7)
		}

		_, signed, err := s.jwtAuth.Encode(claims)

		if err != nil {
			s.serveErrorJSON(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		at.AccessToken = signed
		at.TokenType = "Bearer"
		at.Scope = scope
	}

	if strings.Contains(scope, "profile") {
		at.Profile = &userToken{
			Name:  s.Config.User.Name,
			URL:   s.Config.User.URL,
			Photo: s.Config.User.Photo,
		}
	}

	if strings.Contains(scope, "email") {
		if at.Profile == nil {
			at.Profile = &userToken{}
		}
		at.Profile.Email = s.Config.User.Email
	}

	s.serveJSON(w, http.StatusOK, at)
}

type accessToken struct {
	AccessToken string     `json:"access_token,omitempty"`
	TokenType   string     `json:"token_type,omitempty"`
	Scope       string     `json:"scope,omitempty"`
	Me          string     `json:"me"`
	Profile     *userToken `json:"profile,omitempty"`
}

type userToken struct {
	Name  string `json:"name,omitempty"`
	URL   string `json:"url,omitempty"`
	Photo string `json:"photo,omitempty"`
	Email string `json:"email,omitempty"`
}

var (
	codeChallengeMethods = []string{
		"plain", "S256",
	}
)

func isValidCodeChallengeMethod(ccm string) bool {
	return funk.ContainsString(codeChallengeMethods, ccm)
}

func validateCodeChallenge(ccm, cc, ver string) bool {
	switch ccm {
	case "plain":
		return cc == ver
	case "S256":
		s256 := sha256.Sum256([]byte(ver))
		// trim padding
		a := strings.TrimRight(base64.URLEncoding.EncodeToString(s256[:]), "=")
		b := strings.TrimRight(cc, "=")
		return a == b
	default:
		return false
	}
}

type authRequest struct {
	RedirectURI         string
	ClientID            string
	Scopes              []string
	State               string
	CodeChallenge       string
	CodeChallengeMethod string
}

func getAuthorizationRequest(r *http.Request) (*authRequest, error) {
	redirectURI := r.FormValue("redirect_uri")
	clientID := r.FormValue("client_id")

	if !isValidProfileURL(clientID) {
		return nil, errors.New("client_id is invalid")
	}

	if !isValidProfileURL(redirectURI) {
		return nil, errors.New("redirect_uri is invalid")
	}

	var (
		cc  string
		ccm string
	)

	cc = r.Form.Get("code_challenge")
	if cc != "" {
		if len(cc) < 43 || len(cc) > 128 {
			return nil, errors.New("code_challenge length must be between 43 and 128 charachters long")
		}

		ccm = r.Form.Get("code_challenge_method")
		if !isValidCodeChallengeMethod(ccm) {
			return nil, errors.New("code_challenge_method not supported")
		}
	}

	req := &authRequest{
		RedirectURI:         redirectURI,
		ClientID:            clientID,
		State:               r.Form.Get("state"),
		Scopes:              []string{},
		CodeChallenge:       cc,
		CodeChallengeMethod: ccm,
	}

	scope := r.Form.Get("scope")
	if scope != "" {
		req.Scopes = strings.Split(scope, " ")
	} else if scopes := r.Form["scopes"]; len(scopes) > 0 {
		req.Scopes = scopes
	}

	return req, nil
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

func getRequestInfoFromToken(r *http.Request, token jwt.Token) error {
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

		if !isValidCodeChallengeMethod(ccm) {
			return errors.New("code_challenge_method invalid")
		}

		if !validateCodeChallenge(ccm, cc, codeVerifier) {
			return errors.New("code challenge failed")
		}
	}

	return nil
}

func isValidProfileURL(profileURL string) bool {
	url, err := urlpkg.Parse(profileURL)
	if err != nil {
		return false
	}

	if url.Scheme != "http" && url.Scheme != "https" {
		return false
	}

	if url.Fragment != "" {
		return false
	}

	if url.User.String() != "" {
		return false
	}

	if url.Port() != "" {
		return false
	}

	// TODO: check domain / IP.
	return true
}

func (s *Server) mustIndieAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.isLoggedIn(r) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// The verification above MUST ensure that token exists.
		token, _, _ := jwtauth.FromContext(r.Context())
		if token.Subject() != TokenSubject {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		scopes := strings.Split(getString(token, "scope"), " ")
		ctx := context.WithValue(r.Context(), scopesContextKey, scopes)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) getScopes(r *http.Request) []string {
	if scopes, ok := r.Context().Value(scopesContextKey).([]string); ok {
		return scopes
	}

	return []string{}
}
