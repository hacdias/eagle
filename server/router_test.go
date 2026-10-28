package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.hacdias.com/eagle/core"
)

func TestRedirectFragment(t *testing.T) {
	s := &Server{}
	s.redirects.Store(&map[string]core.Redirect{
		"/contact": {To: "/more/#contact", Status: http.StatusFound},
	})

	w := httptest.NewRecorder()
	s.everythingBagelHandler(w, httptest.NewRequest("GET", "/contact", nil))

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "/more/#contact", w.Header().Get("Location"))
}
