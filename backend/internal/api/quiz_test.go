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
	"github.com/jameynakama/flockdeck/internal/auth"
	"github.com/jameynakama/flockdeck/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubQuerier embeds store.Querier (nil) so unimplemented methods panic if called.
type stubQuerier struct {
	store.Querier
	getNextDueCard        func(ctx context.Context, arg store.GetNextDueCardParams) (store.GetNextDueCardRow, error)
	getRandomMedia        func(ctx context.Context, speciesCode string) (store.GetRandomMediaForSpeciesRow, error)
	getCard               func(ctx context.Context, arg store.GetCardParams) (store.Card, error)
	updateCardSchedule    func(ctx context.Context, arg store.UpdateCardScheduleParams) (store.Card, error)
	getDeckPracticeCards func(ctx context.Context, deckID int64) ([]store.GetDeckPracticeCardsRow, error)
	getDeck              func(ctx context.Context, id int64) (store.Deck, error)
	getUsers             func(ctx context.Context) ([]store.GetUsersRow, error)
	listPresetDecks      func(ctx context.Context) ([]store.ListPresetDecksRow, error)
}

func (s *stubQuerier) GetNextDueCard(ctx context.Context, arg store.GetNextDueCardParams) (store.GetNextDueCardRow, error) {
	return s.getNextDueCard(ctx, arg)
}
func (s *stubQuerier) GetRandomMediaForSpecies(ctx context.Context, speciesCode string) (store.GetRandomMediaForSpeciesRow, error) {
	return s.getRandomMedia(ctx, speciesCode)
}
func (s *stubQuerier) GetCard(ctx context.Context, arg store.GetCardParams) (store.Card, error) {
	return s.getCard(ctx, arg)
}
func (s *stubQuerier) UpdateCardSchedule(ctx context.Context, arg store.UpdateCardScheduleParams) (store.Card, error) {
	return s.updateCardSchedule(ctx, arg)
}
func (s *stubQuerier) GetDeckPracticeCards(ctx context.Context, deckID int64) ([]store.GetDeckPracticeCardsRow, error) {
	return s.getDeckPracticeCards(ctx, deckID)
}
func (s *stubQuerier) GetDeck(ctx context.Context, id int64) (store.Deck, error) {
	return s.getDeck(ctx, id)
}
func (s *stubQuerier) GetUsers(ctx context.Context) ([]store.GetUsersRow, error) {
	return s.getUsers(ctx)
}
func (s *stubQuerier) ListPresetDecks(ctx context.Context) ([]store.ListPresetDecksRow, error) {
	return s.listPresetDecks(ctx)
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

// deckOwnedBy returns a getDeck stub for a deck owned by the given user.
func deckOwnedBy(userID int64) func(context.Context, int64) (store.Deck, error) {
	return func(_ context.Context, id int64) (store.Deck, error) {
		return store.Deck{ID: id, OwnerID: pgtype.Int8{Int64: userID, Valid: true}}, nil
	}
}

func TestGetNextCard_Audio_ReturnsDueCard(t *testing.T) {
	due := pgtype.Timestamptz{}
	require.NoError(t, due.Scan(time.Now().Add(-time.Hour)))

	q := &stubQuerier{
		getDeck: deckOwnedBy(1),
		getNextDueCard: func(_ context.Context, arg store.GetNextDueCardParams) (store.GetNextDueCardRow, error) {
			assert.Equal(t, int64(1), arg.UserID)
			assert.Equal(t, int64(42), arg.DeckID)
			assert.Equal(t, "audio", arg.Lane)
			return store.GetNextDueCardRow{
				SpeciesCode:    "spotto",
				Lane:           "audio",
				CommonName:     "Spotted Towhee",
				ScientificName: "Pipilo maculatus",
				Due:            due,
				DueRemaining:   3,
			}, nil
		},
		getRandomMedia: func(_ context.Context, speciesCode string) (store.GetRandomMediaForSpeciesRow, error) {
			assert.Equal(t, "spotto", speciesCode)
			return store.GetRandomMediaForSpeciesRow{
				AudioPath: "https://r2.example.com/recordings/spotto/123.mp3",
				AudioType: "song",
				ImagePath: "https://r2.example.com/images/spotto/456.jpg",
			}, nil
		},
	}

	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/decks/42/next?lane=audio", nil)
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
	assert.Equal(t, int64(3), body.DueRemaining)
}

func TestGetNextCard_DueBefore_ForwardedToQuery(t *testing.T) {
	// Milliseconds included: this is exactly what the FE's toISOString() sends.
	snapshot := "2026-06-04T19:00:00.000Z"
	var gotDueBefore pgtype.Timestamptz
	q := &stubQuerier{
		getDeck: deckOwnedBy(1),
		getNextDueCard: func(_ context.Context, arg store.GetNextDueCardParams) (store.GetNextDueCardRow, error) {
			gotDueBefore = arg.DueBefore
			return store.GetNextDueCardRow{}, pgx.ErrNoRows
		},
	}

	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet,
		"/api/v1/decks/42/next?lane=audio&due_before="+snapshot, nil)
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.getNextCard(w, r)

	require.True(t, gotDueBefore.Valid, "due_before must reach the query")
	assert.Equal(t, "2026-06-04T19:00:00Z", gotDueBefore.Time.UTC().Format(time.RFC3339))
}

func TestGetNextCard_OmittedDueBefore_QueryGetsNull(t *testing.T) {
	var gotDueBefore pgtype.Timestamptz
	q := &stubQuerier{
		getDeck: deckOwnedBy(1),
		getNextDueCard: func(_ context.Context, arg store.GetNextDueCardParams) (store.GetNextDueCardRow, error) {
			gotDueBefore = arg.DueBefore
			return store.GetNextDueCardRow{}, pgx.ErrNoRows
		},
	}

	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/decks/42/next?lane=audio", nil)
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.getNextCard(w, r)

	assert.False(t, gotDueBefore.Valid, "omitted due_before must stay NULL (falls back to NOW())")
}

func TestGetNextCard_InvalidDueBefore_Returns400(t *testing.T) {
	h := makeHandler(&stubQuerier{getDeck: deckOwnedBy(1)})
	r := httptest.NewRequest(http.MethodGet,
		"/api/v1/decks/42/next?lane=audio&due_before=notatime", nil)
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.getNextCard(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetNextCard_NothingDue_Returns204(t *testing.T) {
	q := &stubQuerier{
		getDeck: deckOwnedBy(1),
		getNextDueCard: func(_ context.Context, _ store.GetNextDueCardParams) (store.GetNextDueCardRow, error) {
			return store.GetNextDueCardRow{}, pgx.ErrNoRows
		},
	}

	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/decks/42/next?lane=audio", nil)
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.getNextCard(w, r)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestGetNextCard_InvalidLane_Returns400(t *testing.T) {
	h := makeHandler(&stubQuerier{})
	r := httptest.NewRequest(http.MethodGet, "/api/v1/decks/42/next?lane=video", nil)
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.getNextCard(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetNextCard_InvalidGroupID_Returns400(t *testing.T) {
	h := makeHandler(&stubQuerier{})
	r := httptest.NewRequest(http.MethodGet, "/api/v1/decks/notanumber/next?lane=audio", nil)
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
		getDeck: deckOwnedBy(1),
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
	r := httptest.NewRequest(http.MethodPost, "/api/v1/decks/42/rate", strings.NewReader(body))
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
	h := makeHandler(&stubQuerier{getDeck: deckOwnedBy(1)})
	body := `{"ebird_code":"spotto","lane":"audio","rating":9}`
	r := httptest.NewRequest(http.MethodPost, "/api/v1/decks/42/rate", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.rateCard(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// A due card whose species has no media for the lane is a data condition, not
// a server fault (e.g. a locked-media species with no recordings).
func TestGetNextCard_NoMedia_Returns404(t *testing.T) {
	due := pgtype.Timestamptz{}
	require.NoError(t, due.Scan(time.Now().Add(-time.Hour)))

	q := &stubQuerier{
		getDeck: deckOwnedBy(1),
		getNextDueCard: func(_ context.Context, _ store.GetNextDueCardParams) (store.GetNextDueCardRow, error) {
			return store.GetNextDueCardRow{SpeciesCode: "spotto", Lane: "audio", Due: due}, nil
		},
		getRandomMedia: func(_ context.Context, _ string) (store.GetRandomMediaForSpeciesRow, error) {
			return store.GetRandomMediaForSpeciesRow{}, nil
		},
	}

	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/decks/42/next?lane=audio", nil)
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.getNextCard(w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetPracticeCards_Audio_ReturnsAllSpecies(t *testing.T) {
	q := &stubQuerier{
		getDeck: func(_ context.Context, id int64) (store.Deck, error) {
			return store.Deck{
				ID:      id,
				OwnerID: pgtype.Int8{Int64: 1, Valid: true}, // user 1 owns this group
			}, nil
		},
		getDeckPracticeCards: func(_ context.Context, deckID int64) ([]store.GetDeckPracticeCardsRow, error) {
			assert.Equal(t, int64(42), deckID)
			return []store.GetDeckPracticeCardsRow{
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
	r := httptest.NewRequest(http.MethodGet, "/api/v1/decks/42/practice?lane=audio", nil)
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
		getDeck: func(_ context.Context, id int64) (store.Deck, error) {
			return store.Deck{
				ID:      id,
				OwnerID: pgtype.Int8{Int64: 1, Valid: true}, // user 1 owns this group
			}, nil
		},
		getDeckPracticeCards: func(_ context.Context, _ int64) ([]store.GetDeckPracticeCardsRow, error) {
			return []store.GetDeckPracticeCardsRow{
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
	r := httptest.NewRequest(http.MethodGet, "/api/v1/decks/42/practice?lane=image", nil)
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
		getDeck: func(_ context.Context, id int64) (store.Deck, error) {
			return store.Deck{
				ID:      id,
				OwnerID: pgtype.Int8{Int64: 1, Valid: true}, // user 1 owns this group
			}, nil
		},
		getDeckPracticeCards: func(_ context.Context, _ int64) ([]store.GetDeckPracticeCardsRow, error) {
			return []store.GetDeckPracticeCardsRow{
				// has audio
				{EbirdCode: "spotto", AudioUrl: "https://r2.example.com/rec.mp3", ImageUrl: ""},
				// no audio
				{EbirdCode: "foxspa", AudioUrl: "", ImageUrl: ""},
			}, nil
		},
	}

	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/decks/42/practice?lane=audio", nil)
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
		getDeck: func(_ context.Context, id int64) (store.Deck, error) {
			return store.Deck{
				ID:      id,
				OwnerID: pgtype.Int8{Int64: 1, Valid: true}, // user 1 owns this group
			}, nil
		},
		getDeckPracticeCards: func(_ context.Context, _ int64) ([]store.GetDeckPracticeCardsRow, error) {
			return []store.GetDeckPracticeCardsRow{}, nil
		},
	}

	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/decks/42/practice?lane=audio", nil)
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
	r := httptest.NewRequest(http.MethodGet, "/api/v1/decks/42/practice?lane=video", nil)
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.getPracticeCards(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetPracticeCards_InvalidGroupID_Returns400(t *testing.T) {
	h := makeHandler(&stubQuerier{})
	r := httptest.NewRequest(http.MethodGet, "/api/v1/decks/notanumber/practice?lane=audio", nil)
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "notanumber")
	w := httptest.NewRecorder()

	h.getPracticeCards(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetPracticeCards_DBError_Returns500(t *testing.T) {
	q := &stubQuerier{
		getDeck: func(_ context.Context, id int64) (store.Deck, error) {
			return store.Deck{
				ID:      id,
				OwnerID: pgtype.Int8{Int64: 1, Valid: true}, // user 1 owns this group
			}, nil
		},
		getDeckPracticeCards: func(_ context.Context, _ int64) ([]store.GetDeckPracticeCardsRow, error) {
			return nil, errors.New("db error")
		},
	}

	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/decks/42/practice?lane=audio", nil)
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.getPracticeCards(w, r)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetPracticeCards_ForbiddenGroup_Returns403(t *testing.T) {
	q := &stubQuerier{
		getDeck: func(_ context.Context, id int64) (store.Deck, error) {
			return store.Deck{
				ID:      id,
				OwnerID: pgtype.Int8{Int64: 999, Valid: true}, // owned by user 999
			}, nil
		},
	}

	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/decks/42/practice?lane=audio", nil)
	r = injectUserID(r, 1) // user 1 is not owner
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.getPracticeCards(w, r)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestGetPracticeCards_PresetDeck_ForbiddenForNonAdmin(t *testing.T) {
	// Presets are clone-only: users practice their own clone, never the preset.
	q := &stubQuerier{
		getDeck: func(_ context.Context, id int64) (store.Deck, error) {
			return store.Deck{
				ID:      id,
				OwnerID: pgtype.Int8{Valid: false}, // preset: no owner
			}, nil
		},
		getDeckPracticeCards: func(_ context.Context, _ int64) ([]store.GetDeckPracticeCardsRow, error) {
			return []store.GetDeckPracticeCardsRow{}, nil
		},
	}

	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/decks/42/practice?lane=audio", nil)
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.getPracticeCards(w, r)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestGetPracticeCards_PresetDeck_AdminAllowed(t *testing.T) {
	q := &stubQuerier{
		getDeck: func(_ context.Context, id int64) (store.Deck, error) {
			return store.Deck{
				ID:      id,
				OwnerID: pgtype.Int8{Valid: false}, // preset: no owner
			}, nil
		},
		getDeckPracticeCards: func(_ context.Context, _ int64) ([]store.GetDeckPracticeCardsRow, error) {
			return []store.GetDeckPracticeCardsRow{}, nil
		},
	}

	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/decks/42/practice?lane=audio", nil)
	r = injectUserID(r, 1)
	r = injectAdmin(r, 1)
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.getPracticeCards(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetNextCard_ForeignDeck_Returns403(t *testing.T) {
	due := pgtype.Timestamptz{}
	require.NoError(t, due.Scan(time.Now().Add(-time.Hour)))

	q := &stubQuerier{
		getDeck: func(_ context.Context, id int64) (store.Deck, error) {
			return store.Deck{
				ID:      id,
				OwnerID: pgtype.Int8{Int64: 999, Valid: true}, // owned by user 999
			}, nil
		},
		// Full stubs so the handler would return 200 if the ownership check is missing.
		getNextDueCard: func(_ context.Context, _ store.GetNextDueCardParams) (store.GetNextDueCardRow, error) {
			return store.GetNextDueCardRow{SpeciesCode: "spotto", Lane: "audio", Due: due}, nil
		},
		getRandomMedia: func(_ context.Context, _ string) (store.GetRandomMediaForSpeciesRow, error) {
			return store.GetRandomMediaForSpeciesRow{AudioPath: "https://r2.example.com/rec.mp3", ImagePath: "https://r2.example.com/img.jpg"}, nil
		},
	}

	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/decks/42/next?lane=audio", nil)
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.getNextCard(w, r)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestGetNextCard_PresetDeck_ForbiddenForNonAdmin(t *testing.T) {
	q := &stubQuerier{
		getDeck: func(_ context.Context, id int64) (store.Deck, error) {
			return store.Deck{
				ID:      id,
				OwnerID: pgtype.Int8{Valid: false}, // preset: no owner
			}, nil
		},
		getNextDueCard: func(_ context.Context, _ store.GetNextDueCardParams) (store.GetNextDueCardRow, error) {
			return store.GetNextDueCardRow{}, pgx.ErrNoRows
		},
	}

	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/decks/42/next?lane=audio", nil)
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.getNextCard(w, r)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestRateCard_ForeignDeck_Returns403(t *testing.T) {
	q := &stubQuerier{
		getDeck: func(_ context.Context, id int64) (store.Deck, error) {
			return store.Deck{
				ID:      id,
				OwnerID: pgtype.Int8{Int64: 999, Valid: true}, // owned by user 999
			}, nil
		},
		// Full stubs so the handler would return 200 if the ownership check is missing.
		getCard: func(_ context.Context, _ store.GetCardParams) (store.Card, error) {
			return store.Card{UserID: 1, SpeciesCode: "spotto", Lane: "audio"}, nil
		},
		updateCardSchedule: func(_ context.Context, _ store.UpdateCardScheduleParams) (store.Card, error) {
			return store.Card{UserID: 1, SpeciesCode: "spotto", Lane: "audio"}, nil
		},
	}

	h := makeHandler(q)
	body := `{"ebird_code":"spotto","lane":"audio","rating":3}`
	r := httptest.NewRequest(http.MethodPost, "/api/v1/decks/42/rate", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.rateCard(w, r)

	assert.Equal(t, http.StatusForbidden, w.Code)
}
