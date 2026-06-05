package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jameynakama/flockdeck/internal/auth"
	"github.com/jameynakama/flockdeck/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests exercise the real router: middleware attachment (RequireAuth,
// RequireAdmin) and URL-param wiring that the direct handler tests bypass.

var testRouterSecret = []byte("router-test-secret")

func newTestRouter(q store.Querier) http.Handler {
	return NewRouter(RouterConfig{
		Queries:     q,
		JWTSecret:   testRouterSecret,
		FrontendURL: "http://localhost:5173",
	})
}

func routerCookie(t *testing.T, userID int64, isAdmin bool) *http.Cookie {
	t.Helper()
	token, err := auth.SignToken(userID, isAdmin, testRouterSecret)
	require.NoError(t, err)
	return &http.Cookie{Name: auth.CookieName, Value: token}
}

func routerGet(h http.Handler, path string, cookie *http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestRouter_Health_IsPublic(t *testing.T) {
	rec := routerGet(newTestRouter(&stubQuerier{}), "/health", nil)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRouter_Presets_IsPublic(t *testing.T) {
	q := &stubQuerier{
		listPresetDecks: func(context.Context) ([]store.ListPresetDecksRow, error) {
			return []store.ListPresetDecksRow{}, nil
		},
	}
	rec := routerGet(newTestRouter(q), "/api/v1/decks/presets", nil)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRouter_ProtectedRoute_NoCookie_401(t *testing.T) {
	// stubQuerier with no funcs set: any handler reaching the store fails the
	// test, proving RequireAuth rejected the request first.
	rec := routerGet(newTestRouter(&stubQuerier{}), "/api/v1/decks", nil)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestRouter_ProtectedRoute_GarbageCookie_401(t *testing.T) {
	cookie := &http.Cookie{Name: auth.CookieName, Value: "not-a-jwt"}
	rec := routerGet(newTestRouter(&stubQuerier{}), "/api/v1/decks", cookie)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestRouter_ProtectedRoute_WiresParamsAndUser(t *testing.T) {
	var gotDeckID int64
	q := &stubQuerier{
		getDeck: deckOwnedBy(7),
		getDeckPracticeCards: func(_ context.Context, deckID int64) ([]store.GetDeckPracticeCardsRow, error) {
			gotDeckID = deckID
			return []store.GetDeckPracticeCardsRow{}, nil
		},
	}
	rec := routerGet(newTestRouter(q), "/api/v1/decks/42/practice?lane=audio", routerCookie(t, 7, false))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, int64(42), gotDeckID, "Should pass the {id} URL param through to the handler")
}

func TestRouter_AdminRoute_NonAdmin_403(t *testing.T) {
	rec := routerGet(newTestRouter(&stubQuerier{}), "/api/v1/admin/users", routerCookie(t, 7, false))
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestRouter_AdminRoute_NoCookie_401(t *testing.T) {
	rec := routerGet(newTestRouter(&stubQuerier{}), "/api/v1/admin/users", nil)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestRouter_Stats_RequiresAuth(t *testing.T) {
	rec := routerGet(newTestRouter(&stubQuerier{}), "/api/v1/stats", nil)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestRouter_AdminRoute_Admin_200(t *testing.T) {
	q := &stubQuerier{
		getUsers: func(context.Context) ([]store.GetUsersRow, error) {
			return []store.GetUsersRow{}, nil
		},
	}
	rec := routerGet(newTestRouter(q), "/api/v1/admin/users", routerCookie(t, 7, true))
	assert.Equal(t, http.StatusOK, rec.Code)
}
