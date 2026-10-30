package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.hacdias.com/eagle/core"
	"go.hacdias.com/eagle/log"
	"go.hacdias.com/indielib/indieauth"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	dir := t.TempDir()
	c := &core.Config{
		ServerConfig: core.ServerConfig{
			Development:     true,
			SourceDirectory: dir,
			PublicDirectory: dir,
			DataDirectory:   dir,
		},
		Site: core.SiteConfig{
			BaseURL: "https://example.com",
		},
	}

	co, err := core.NewCore(c)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = co.Close()
	})

	return &Server{
		c:    c,
		log:  log.S(),
		ias:  indieauth.NewServer(false, nil),
		core: co,
	}
}

type tokenTestResponse struct {
	tokenResponse
	Error string `json:"error"`
}

func postToken(t *testing.T, s *Server, form url.Values) (int, *tokenTestResponse) {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, tokenPath, strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	s.tokenPost(w, r)

	res := &tokenTestResponse{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), res))
	return w.Code, res
}

func TestTokenEndpoint(t *testing.T) {
	const (
		clientID    = "https://client.example/"
		redirectURI = "https://client.example/callback"
	)

	createCode := func(t *testing.T, s *Server, expiry time.Time) string {
		t.Helper()
		code := uuid.New().String()
		require.NoError(t, s.core.DB().CreateAuthCode(context.Background(), &core.AuthCode{
			ID:          core.HashToken(code),
			ClientID:    clientID,
			RedirectURI: redirectURI,
			Scope:       "profile mcp",
			Lifetime:    time.Hour,
			Expiry:      expiry,
		}))
		return code
	}

	exchange := func(code string) url.Values {
		return url.Values{
			"grant_type":   {"authorization_code"},
			"code":         {code},
			"client_id":    {clientID},
			"redirect_uri": {redirectURI},
		}
	}

	refresh := func(token, scope string) url.Values {
		return url.Values{
			"grant_type":    {"refresh_token"},
			"refresh_token": {token},
			"scope":         {scope},
		}
	}

	t.Run("code can only be used once", func(t *testing.T) {
		s := newTestServer(t)
		code := createCode(t, s, time.Now().Add(time.Minute))

		status, res := postToken(t, s, exchange(code))
		require.Equal(t, http.StatusOK, status)
		assert.NotEmpty(t, res.AccessToken)
		assert.Equal(t, "profile mcp", res.Scope)

		status, res = postToken(t, s, exchange(code))
		assert.Equal(t, http.StatusBadRequest, status)
		assert.Equal(t, "invalid_grant", res.Error)
	})

	t.Run("expired code is rejected", func(t *testing.T) {
		s := newTestServer(t)
		code := createCode(t, s, time.Now().Add(-time.Minute))

		status, res := postToken(t, s, exchange(code))
		assert.Equal(t, http.StatusBadRequest, status)
		assert.Equal(t, "invalid_grant", res.Error)
	})

	t.Run("refresh cannot widen scope", func(t *testing.T) {
		s := newTestServer(t)
		_, res := postToken(t, s, exchange(createCode(t, s, time.Now().Add(time.Minute))))
		require.NotEmpty(t, res.RefreshToken)

		status, res := postToken(t, s, refresh(res.RefreshToken, "profile mcp email"))
		assert.Equal(t, http.StatusBadRequest, status)
		assert.Equal(t, "invalid_scope", res.Error)
	})

	t.Run("refresh token is rotated", func(t *testing.T) {
		s := newTestServer(t)
		_, res := postToken(t, s, exchange(createCode(t, s, time.Now().Add(time.Minute))))
		require.NotEmpty(t, res.RefreshToken)
		oldRefreshToken := res.RefreshToken

		status, res := postToken(t, s, refresh(oldRefreshToken, ""))
		require.Equal(t, http.StatusOK, status)
		assert.NotEmpty(t, res.AccessToken)
		assert.NotEqual(t, oldRefreshToken, res.RefreshToken)
		assert.Equal(t, "profile mcp", res.Scope)

		status, res = postToken(t, s, refresh(oldRefreshToken, ""))
		assert.Equal(t, http.StatusBadRequest, status)
		assert.Equal(t, "invalid_grant", res.Error)
	})
}
