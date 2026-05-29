package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jameynakama/lifer/internal/auth"
	"github.com/jameynakama/lifer/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubQuerier embeds store.Querier (nil) so unimplemented methods panic if called.
type stubQuerier struct {
	store.Querier
	getNextDueCard        func(ctx context.Context, arg store.GetNextDueCardParams) (store.GetNextDueCardRow, error)
	getRandomRecording    func(ctx context.Context, speciesCode string) (string, error)
	getRandomImage        func(ctx context.Context, speciesCode string) (string, error)
	getCard               func(ctx context.Context, arg store.GetCardParams) (store.Card, error)
	updateCardSchedule    func(ctx context.Context, arg store.UpdateCardScheduleParams) (store.Card, error)
	getGroupPracticeCards func(ctx context.Context, groupID int64) ([]store.GetGroupPracticeCardsRow, error)
	getGroup              func(ctx context.Context, id int64) (store.Group, error)
}

func (s *stubQuerier) GetNextDueCard(ctx context.Context, arg store.GetNextDueCardParams) (store.GetNextDueCardRow, error) {
	return s.getNextDueCard(ctx, arg)
}
func (s *stubQuerier) GetRandomRecording(ctx context.Context, speciesCode string) (string, error) {
	return s.getRandomRecording(ctx, speciesCode)
}
func (s *stubQuerier) GetRandomImage(ctx context.Context, speciesCode string) (string, error) {
	return s.getRandomImage(ctx, speciesCode)
}
func (s *stubQuerier) GetCard(ctx context.Context, arg store.GetCardParams) (store.Card, error) {
	return s.getCard(ctx, arg)
}
func (s *stubQuerier) UpdateCardSchedule(ctx context.Context, arg store.UpdateCardScheduleParams) (store.Card, error) {
	return s.updateCardSchedule(ctx, arg)
}
func (s *stubQuerier) GetGroupPracticeCards(ctx context.Context, groupID int64) ([]store.GetGroupPracticeCardsRow, error) {
	return s.getGroupPracticeCards(ctx, groupID)
}
func (s *stubQuerier) GetGroup(ctx context.Context, id int64) (store.Group, error) {
	return s.getGroup(ctx, id)
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

// withChiParams sets multiple chi URL params on the request in a single context.
func withChiParams(r *http.Request, params map[string]string) *http.Request {
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
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
				SpeciesCode:    "spotto",
				Lane:           "audio",
				CommonName:     "Spotted Towhee",
				ScientificName: "Pipilo maculatus",
				Due:            due,
			}, nil
		},
		getRandomRecording: func(_ context.Context, speciesCode string) (string, error) {
			assert.Equal(t, "spotto", speciesCode)
			return "https://r2.example.com/recordings/spotto/123.mp3", nil
		},
		getRandomImage: func(_ context.Context, speciesCode string) (string, error) {
			return "https://r2.example.com/images/spotto/456.jpg", nil
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
	assert.Equal(t, "spotto", body.EbirdCode)
	assert.Equal(t, "Spotted Towhee", body.CommonName)
	assert.Equal(t, "https://r2.example.com/recordings/spotto/123.mp3", body.MediaURL)
	assert.Equal(t, "https://r2.example.com/images/spotto/456.jpg", body.PhotoURL)
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

func TestGetNextCard_InvalidLane_Returns400(t *testing.T) {
	h := makeHandler(&stubQuerier{})
	r := httptest.NewRequest(http.MethodGet, "/api/v1/groups/42/next?lane=video", nil)
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.getNextCard(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetNextCard_InvalidGroupID_Returns400(t *testing.T) {
	h := makeHandler(&stubQuerier{})
	r := httptest.NewRequest(http.MethodGet, "/api/v1/groups/notanumber/next?lane=audio", nil)
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "notanumber")
	w := httptest.NewRecorder()

	h.getNextCard(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRateCard_UpdatesSchedule(t *testing.T) {
	now := time.Now()
	due := pgtype.Timestamptz{}
	require.NoError(t, due.Scan(now.Add(24*time.Hour)))

	updatedCard := store.Card{
		UserID:      1,
		SpeciesCode: "spotto",
		Lane:        "audio",
		Stability:   2.5,
		Due:         due,
		State:       2,
	}

	q := &stubQuerier{
		getCard: func(_ context.Context, arg store.GetCardParams) (store.Card, error) {
			assert.Equal(t, int64(1), arg.UserID)
			assert.Equal(t, "spotto", arg.SpeciesCode)
			assert.Equal(t, "audio", arg.Lane)
			return store.Card{
				UserID:      1,
				SpeciesCode: "spotto",
				Lane:        "audio",
				State:       0,
			}, nil
		},
		updateCardSchedule: func(_ context.Context, arg store.UpdateCardScheduleParams) (store.Card, error) {
			assert.Equal(t, int64(1), arg.UserID)
			assert.Equal(t, "spotto", arg.SpeciesCode)
			assert.Equal(t, "audio", arg.Lane)
			assert.Greater(t, arg.Stability, 0.0)
			return updatedCard, nil
		},
	}

	h := makeHandler(q)
	body := `{"ebird_code":"spotto","lane":"audio","rating":3}`
	r := httptest.NewRequest(http.MethodPost, "/api/v1/groups/42/rate", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.rateCard(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var got store.Card
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	assert.Equal(t, "spotto", got.SpeciesCode)
	assert.Equal(t, "audio", got.Lane)
}

func TestRateCard_InvalidRating_Returns400(t *testing.T) {
	h := makeHandler(&stubQuerier{})
	body := `{"ebird_code":"spotto","lane":"audio","rating":9}`
	r := httptest.NewRequest(http.MethodPost, "/api/v1/groups/42/rate", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.rateCard(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetNextCard_NoMedia_Returns500(t *testing.T) {
	due := pgtype.Timestamptz{}
	require.NoError(t, due.Scan(time.Now().Add(-time.Hour)))

	q := &stubQuerier{
		getNextDueCard: func(_ context.Context, _ store.GetNextDueCardParams) (store.GetNextDueCardRow, error) {
			return store.GetNextDueCardRow{SpeciesCode: "spotto", Lane: "audio", Due: due}, nil
		},
		getRandomRecording: func(_ context.Context, _ string) (string, error) {
			return "", pgx.ErrNoRows
		},
	}

	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/groups/42/next?lane=audio", nil)
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.getNextCard(w, r)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetPracticeCards_Audio_ReturnsAllSpecies(t *testing.T) {
	q := &stubQuerier{
		getGroup: func(_ context.Context, id int64) (store.Group, error) {
			return store.Group{
				ID:      id,
				OwnerID: pgtype.Int8{Int64: 1, Valid: true}, // user 1 owns this group
			}, nil
		},
		getGroupPracticeCards: func(_ context.Context, groupID int64) ([]store.GetGroupPracticeCardsRow, error) {
			assert.Equal(t, int64(42), groupID)
			return []store.GetGroupPracticeCardsRow{
				{
					EbirdCode:      "spotto",
					CommonName:     "Spotted Towhee",
					ScientificName: "Pipilo maculatus",
					AudioUrl:       "https://r2.example.com/rec.mp3",
					ImageUrl:       "https://r2.example.com/img.jpg",
				},
			}, nil
		},
	}

	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/groups/42/practice?lane=audio", nil)
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.getPracticeCards(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var body []nextCardResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Len(t, body, 1)
	assert.Equal(t, "spotto", body[0].EbirdCode)
	assert.Equal(t, "https://r2.example.com/rec.mp3", body[0].MediaURL)
	assert.Equal(t, "https://r2.example.com/img.jpg", body[0].PhotoURL)
	assert.Equal(t, "audio", body[0].Lane)
}

func TestGetPracticeCards_Image_UsesImageAsMediaURL(t *testing.T) {
	q := &stubQuerier{
		getGroup: func(_ context.Context, id int64) (store.Group, error) {
			return store.Group{
				ID:      id,
				OwnerID: pgtype.Int8{Int64: 1, Valid: true}, // user 1 owns this group
			}, nil
		},
		getGroupPracticeCards: func(_ context.Context, _ int64) ([]store.GetGroupPracticeCardsRow, error) {
			return []store.GetGroupPracticeCardsRow{
				{
					EbirdCode:      "spotto",
					CommonName:     "Spotted Towhee",
					ScientificName: "Pipilo maculatus",
					AudioUrl:       "",
					ImageUrl:       "https://r2.example.com/img.jpg",
				},
			}, nil
		},
	}

	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/groups/42/practice?lane=image", nil)
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.getPracticeCards(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var body []nextCardResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Len(t, body, 1)
	assert.Equal(t, "https://r2.example.com/img.jpg", body[0].MediaURL)
	assert.Equal(t, "https://r2.example.com/img.jpg", body[0].PhotoURL)
	assert.Equal(t, "image", body[0].Lane)
}

func TestGetPracticeCards_FiltersSpeciesWithNoMedia(t *testing.T) {
	q := &stubQuerier{
		getGroup: func(_ context.Context, id int64) (store.Group, error) {
			return store.Group{
				ID:      id,
				OwnerID: pgtype.Int8{Int64: 1, Valid: true}, // user 1 owns this group
			}, nil
		},
		getGroupPracticeCards: func(_ context.Context, _ int64) ([]store.GetGroupPracticeCardsRow, error) {
			return []store.GetGroupPracticeCardsRow{
				// has audio
				{EbirdCode: "spotto", AudioUrl: "https://r2.example.com/rec.mp3", ImageUrl: ""},
				// no audio
				{EbirdCode: "foxspa", AudioUrl: "", ImageUrl: ""},
			}, nil
		},
	}

	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/groups/42/practice?lane=audio", nil)
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.getPracticeCards(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var body []nextCardResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Len(t, body, 1)
	assert.Equal(t, "spotto", body[0].EbirdCode)
}

func TestGetPracticeCards_EmptyGroup_ReturnsEmptyArray(t *testing.T) {
	q := &stubQuerier{
		getGroup: func(_ context.Context, id int64) (store.Group, error) {
			return store.Group{
				ID:      id,
				OwnerID: pgtype.Int8{Int64: 1, Valid: true}, // user 1 owns this group
			}, nil
		},
		getGroupPracticeCards: func(_ context.Context, _ int64) ([]store.GetGroupPracticeCardsRow, error) {
			return []store.GetGroupPracticeCardsRow{}, nil
		},
	}

	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/groups/42/practice?lane=audio", nil)
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.getPracticeCards(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var body []nextCardResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Empty(t, body)
}

func TestGetPracticeCards_InvalidLane_Returns400(t *testing.T) {
	h := makeHandler(&stubQuerier{})
	r := httptest.NewRequest(http.MethodGet, "/api/v1/groups/42/practice?lane=video", nil)
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.getPracticeCards(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetPracticeCards_InvalidGroupID_Returns400(t *testing.T) {
	h := makeHandler(&stubQuerier{})
	r := httptest.NewRequest(http.MethodGet, "/api/v1/groups/notanumber/practice?lane=audio", nil)
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "notanumber")
	w := httptest.NewRecorder()

	h.getPracticeCards(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetPracticeCards_DBError_Returns500(t *testing.T) {
	q := &stubQuerier{
		getGroup: func(_ context.Context, id int64) (store.Group, error) {
			return store.Group{
				ID:      id,
				OwnerID: pgtype.Int8{Int64: 1, Valid: true}, // user 1 owns this group
			}, nil
		},
		getGroupPracticeCards: func(_ context.Context, _ int64) ([]store.GetGroupPracticeCardsRow, error) {
			return nil, errors.New("db error")
		},
	}

	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/groups/42/practice?lane=audio", nil)
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.getPracticeCards(w, r)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetPracticeCards_ForbiddenGroup_Returns403(t *testing.T) {
	q := &stubQuerier{
		getGroup: func(_ context.Context, id int64) (store.Group, error) {
			return store.Group{
				ID:      id,
				OwnerID: pgtype.Int8{Int64: 999, Valid: true}, // owned by user 999
			}, nil
		},
	}

	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/groups/42/practice?lane=audio", nil)
	r = injectUserID(r, 1) // user 1 is not owner
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.getPracticeCards(w, r)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestGetPracticeCards_PresetGroup_AllowsAccess(t *testing.T) {
	q := &stubQuerier{
		getGroup: func(_ context.Context, id int64) (store.Group, error) {
			return store.Group{
				ID:      id,
				OwnerID: pgtype.Int8{Valid: false}, // preset: no owner
			}, nil
		},
		getGroupPracticeCards: func(_ context.Context, _ int64) ([]store.GetGroupPracticeCardsRow, error) {
			return []store.GetGroupPracticeCardsRow{}, nil
		},
	}

	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/groups/42/practice?lane=audio", nil)
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.getPracticeCards(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
}
