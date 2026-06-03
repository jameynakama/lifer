package auth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jameynakama/flockdeck/internal/auth"
	"github.com/stretchr/testify/assert"
)

func okHandler(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }

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
