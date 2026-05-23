package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jameynakama/lifer/internal/auth"
	"github.com/jameynakama/lifer/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubQuerier embeds store.Querier (nil) so unimplemented methods panic if called.
// Tests override only the methods they need.
type stubQuerier struct {
	store.Querier
	getNextDueCard     func(ctx context.Context, arg store.GetNextDueCardParams) (store.GetNextDueCardRow, error)
	getRandomRecording func(ctx context.Context, speciesID int64) (string, error)
	getRandomImage     func(ctx context.Context, speciesID int64) (string, error)
	getCard            func(ctx context.Context, arg store.GetCardParams) (store.Card, error)
	updateCardSchedule func(ctx context.Context, arg store.UpdateCardScheduleParams) (store.Card, error)
}

func (s *stubQuerier) GetNextDueCard(ctx context.Context, arg store.GetNextDueCardParams) (store.GetNextDueCardRow, error) {
	return s.getNextDueCard(ctx, arg)
}
func (s *stubQuerier) GetRandomRecording(ctx context.Context, speciesID int64) (string, error) {
	return s.getRandomRecording(ctx, speciesID)
}
func (s *stubQuerier) GetRandomImage(ctx context.Context, speciesID int64) (string, error) {
	return s.getRandomImage(ctx, speciesID)
}
func (s *stubQuerier) GetCard(ctx context.Context, arg store.GetCardParams) (store.Card, error) {
	return s.getCard(ctx, arg)
}
func (s *stubQuerier) UpdateCardSchedule(ctx context.Context, arg store.UpdateCardScheduleParams) (store.Card, error) {
	return s.updateCardSchedule(ctx, arg)
}

func makeHandler(q store.Querier) *Handler {
	return &Handler{queries: q}
}

// injectUserID adds a user ID to the request context, simulating RequireAuth.
func injectUserID(r *http.Request, userID int64) *http.Request {
	ctx := context.WithValue(r.Context(), auth.UserIDKey(), userID)
	return r.WithContext(ctx)
}

// withChiParam sets a chi URL param on the request.
func withChiParam(r *http.Request, key, val string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, val)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func TestGetNextCard_Audio_ReturnsDueCard(t *testing.T) {
	due := pgtype.Timestamptz{}
	require.NoError(t, due.Scan(time.Now().Add(-time.Hour)))

	q := &stubQuerier{
		getNextDueCard: func(_ context.Context, arg store.GetNextDueCardParams) (store.GetNextDueCardRow, error) {
			assert.Equal(t, int64(1), arg.UserID)
			assert.Equal(t, int64(42), arg.GroupID)
			assert.Equal(t, "audio", arg.Lane)
			return store.GetNextDueCardRow{
				SpeciesID:      99,
				Lane:           "audio",
				CommonName:     "Spotted Towhee",
				ScientificName: "Pipilo maculatus",
				Due:            due,
			}, nil
		},
		getRandomRecording: func(_ context.Context, speciesID int64) (string, error) {
			assert.Equal(t, int64(99), speciesID)
			return "https://xeno-canto.org/123/download", nil
		},
		getRandomImage: func(_ context.Context, speciesID int64) (string, error) {
			return "https://cdn.example.com/photo.jpg", nil
		},
	}

	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/groups/42/next?lane=audio", nil)
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.getNextCard(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var body nextCardResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, int64(99), body.SpeciesID)
	assert.Equal(t, "Spotted Towhee", body.CommonName)
	assert.Equal(t, "https://xeno-canto.org/123/download", body.MediaURL)
	assert.Equal(t, "https://cdn.example.com/photo.jpg", body.PhotoURL)
	assert.Equal(t, "audio", body.Lane)
}

func TestGetNextCard_NothingDue_Returns204(t *testing.T) {
	q := &stubQuerier{
		getNextDueCard: func(_ context.Context, _ store.GetNextDueCardParams) (store.GetNextDueCardRow, error) {
			return store.GetNextDueCardRow{}, pgx.ErrNoRows
		},
	}

	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/groups/42/next?lane=audio", nil)
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.getNextCard(w, r)

	assert.Equal(t, http.StatusNoContent, w.Code)
}
