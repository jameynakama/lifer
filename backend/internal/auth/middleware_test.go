package auth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jameynakama/flockdeck/internal/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func okHandler(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }

// RequireAuth

func TestRequireAuth_ValidCookie_PassesAndInjectsContext(t *testing.T) {
	var gotUserID int64
	var gotIsAdmin bool
	h := auth.RequireAuth(testSecret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID = auth.UserIDFromCtx(r.Context())
		gotIsAdmin = auth.IsAdminFromCtx(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	token, err := auth.SignToken(42, true, testSecret)
	require.NoError(t, err)
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(&http.Cookie{Name: "flockdeck_token", Value: token})
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, int64(42), gotUserID)
	assert.True(t, gotIsAdmin)
}

func TestRequireAuth_MissingCookie_Returns401(t *testing.T) {
	called := false
	h := auth.RequireAuth(testSecret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.False(t, called, "handler must not run without a valid token")
}

func TestRequireAuth_InvalidToken_Returns401(t *testing.T) {
	called := false
	h := auth.RequireAuth(testSecret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(&http.Cookie{Name: "flockdeck_token", Value: "not.a.jwt"})
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.False(t, called, "handler must not run without a valid token")
}

func TestRequireAuth_TokenSignedWithOtherSecret_Returns401(t *testing.T) {
	h := auth.RequireAuth(testSecret)(http.HandlerFunc(okHandler))

	token, err := auth.SignToken(42, true, []byte("attacker-secret"))
	require.NoError(t, err)
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(&http.Cookie{Name: "flockdeck_token", Value: token})
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// RequireAdmin

func TestRequireAdmin_AdminUserPasses(t *testing.T) {
	h := auth.RequireAdmin(http.HandlerFunc(okHandler))
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := context.WithValue(r.Context(), auth.UserIDKey(), int64(1))
	ctx = context.WithValue(ctx, auth.IsAdminKey(), true)
	r = r.WithContext(ctx)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequireAdmin_NonAdminReturns403(t *testing.T) {
	h := auth.RequireAdmin(http.HandlerFunc(okHandler))
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := context.WithValue(r.Context(), auth.UserIDKey(), int64(1))
	ctx = context.WithValue(ctx, auth.IsAdminKey(), false)
	r = r.WithContext(ctx)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestRequireAdmin_MissingAdminContextReturns403(t *testing.T) {
	h := auth.RequireAdmin(http.HandlerFunc(okHandler))
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)

	assert.Equal(t, http.StatusForbidden, w.Code)
}
