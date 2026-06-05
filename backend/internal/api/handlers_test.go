package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jameynakama/flockdeck/internal/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func oauthHandler() *Handler {
	cfg := auth.NewGoogleConfig("client-id", "secret", "http://localhost/cb")
	// Dead endpoint so the post-state token exchange fails instantly instead
	// of reaching the real network in tests.
	cfg.Endpoint = oauth2.Endpoint{
		AuthURL:  "http://127.0.0.1:1/auth",
		TokenURL: "http://127.0.0.1:1/token",
	}
	return &Handler{
		oauthConfig: cfg,
		frontendURL: "http://localhost:5173",
	}
}

func TestGoogleLogin_SetsPerFlowStateCookie(t *testing.T) {
	h := oauthHandler()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/auth/google", nil)
	w := httptest.NewRecorder()

	h.googleLogin(w, r)

	require.Equal(t, http.StatusTemporaryRedirect, w.Code)
	cookies := w.Result().Cookies()
	require.Len(t, cookies, 1)
	c := cookies[0]
	// Per-flow cookie name (oauth_state_<state>) so concurrent flows -- e.g.
	// a browser tab and an installed PWA window -- cannot clobber each other.
	assert.True(t, strings.HasPrefix(c.Name, "oauth_state_"), "cookie name %q", c.Name)
	assert.Equal(t, c.Name, "oauth_state_"+c.Value)
	assert.Contains(t, w.Header().Get("Location"), "state="+c.Value)
	assert.True(t, c.HttpOnly)
}

func TestGoogleCallback_MissingStateCookie_RedirectsWithError(t *testing.T) {
	h := oauthHandler()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/auth/google/callback?state=abc&code=x", nil)
	w := httptest.NewRecorder()

	h.googleCallback(w, r)

	// A user-facing browser navigation: redirect to the frontend with an
	// error param instead of white-screening on a bare 400.
	require.Equal(t, http.StatusTemporaryRedirect, w.Code)
	loc := w.Header().Get("Location")
	assert.Contains(t, loc, "http://localhost:5173")
	assert.Contains(t, loc, "error=auth_state")
}

func TestGoogleCallback_MismatchedState_RedirectsWithError(t *testing.T) {
	h := oauthHandler()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/auth/google/callback?state=abc&code=x", nil)
	r.AddCookie(&http.Cookie{Name: "oauth_state_abc", Value: "different"})
	w := httptest.NewRecorder()

	h.googleCallback(w, r)

	require.Equal(t, http.StatusTemporaryRedirect, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "error=auth_state")
}

func TestGoogleCallback_ValidState_DeletesStateCookie(t *testing.T) {
	// With a valid state the handler proceeds to the Google token exchange,
	// which fails in tests (no network config) -- but the one-time state
	// cookie must already be expired by then (closes the replay window).
	h := oauthHandler()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/auth/google/callback?state=abc&code=x", nil)
	r.AddCookie(&http.Cookie{Name: "oauth_state_abc", Value: "abc"})
	w := httptest.NewRecorder()

	h.googleCallback(w, r)

	var deleted bool
	for _, c := range w.Result().Cookies() {
		if c.Name == "oauth_state_abc" && c.MaxAge < 0 {
			deleted = true
		}
	}
	assert.True(t, deleted, "state cookie must be deleted once consumed")
}
