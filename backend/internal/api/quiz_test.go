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
	getNextDueCard       func(ctx context.Context, arg store.GetNextDueCardParams) (store.GetNextDueCardRow, error)
	getRandomMedia       func(ctx context.Context, speciesCode string) (store.GetRandomMediaForSpeciesRow, error)
	getCard              func(ctx context.Context, arg store.GetCardParams) (store.Card, error)
	updateCardSchedule   func(ctx context.Context, arg store.UpdateCardScheduleParams) (store.Card, error)
	createReviewLog      func(ctx context.Context, arg store.CreateReviewLogParams) (store.ReviewLog, error)
	getDeckPracticeCards func(ctx context.Context, deckID int64) ([]store.GetDeckPracticeCardsRow, error)
	getDeck              func(ctx context.Context, id int64) (store.Deck, error)
	getUsers             func(ctx context.Context) ([]store.GetUsersRow, error)
	listPresetDecks      func(ctx context.Context) ([]store.ListPresetDecksRow, error)
	// stats stubs
	getCardTotals      func(ctx context.Context, arg store.GetCardTotalsParams) (store.GetCardTotalsRow, error)
	getCardStateCounts func(ctx context.Context, arg store.GetCardStateCountsParams) ([]store.GetCardStateCountsRow, error)
	getBankedCards     func(ctx context.Context, arg store.GetBankedCardsParams) ([]store.GetBankedCardsRow, error)
	getReviewedCards   func(ctx context.Context, arg store.GetReviewedCardsParams) ([]store.GetReviewedCardsRow, error)
	getLaneGaps        func(ctx context.Context, userID int64) ([]store.GetLaneGapsRow, error)
	getConfusionPairs  func(ctx context.Context, arg store.GetConfusionPairsParams) ([]store.GetConfusionPairsRow, error)
	getFamilyAccuracy  func(ctx context.Context, arg store.GetFamilyAccuracyParams) ([]store.GetFamilyAccuracyRow, error)
	getHardMedia       func(ctx context.Context, arg store.GetHardMediaParams) ([]store.GetHardMediaRow, error)
	getReviewAccuracy  func(ctx context.Context, arg store.GetReviewAccuracyParams) (store.GetReviewAccuracyRow, error)
	countReviewsSince  func(ctx context.Context, arg store.CountReviewsSinceParams) (int64, error)
	// tier stubs
	getCardsInTier func(ctx context.Context, arg store.GetCardsInTierParams) ([]store.GetCardsInTierRow, error)
	// reset stubs
	deleteAllCardsForUser   func(ctx context.Context, userID int64) (int64, error)
	deleteAllReviewsForUser func(ctx context.Context, userID int64) (int64, error)
	seedCardsForUserDecks   func(ctx context.Context, userID int64) (int64, error)
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
func (s *stubQuerier) CreateReviewLog(ctx context.Context, arg store.CreateReviewLogParams) (store.ReviewLog, error) {
	return s.createReviewLog(ctx, arg)
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
func (s *stubQuerier) GetCardTotals(ctx context.Context, arg store.GetCardTotalsParams) (store.GetCardTotalsRow, error) {
	return s.getCardTotals(ctx, arg)
}
func (s *stubQuerier) GetCardStateCounts(ctx context.Context, arg store.GetCardStateCountsParams) ([]store.GetCardStateCountsRow, error) {
	return s.getCardStateCounts(ctx, arg)
}
func (s *stubQuerier) GetBankedCards(ctx context.Context, arg store.GetBankedCardsParams) ([]store.GetBankedCardsRow, error) {
	return s.getBankedCards(ctx, arg)
}
func (s *stubQuerier) GetReviewedCards(ctx context.Context, arg store.GetReviewedCardsParams) ([]store.GetReviewedCardsRow, error) {
	return s.getReviewedCards(ctx, arg)
}
func (s *stubQuerier) GetLaneGaps(ctx context.Context, userID int64) ([]store.GetLaneGapsRow, error) {
	return s.getLaneGaps(ctx, userID)
}
func (s *stubQuerier) GetConfusionPairs(ctx context.Context, arg store.GetConfusionPairsParams) ([]store.GetConfusionPairsRow, error) {
	return s.getConfusionPairs(ctx, arg)
}
func (s *stubQuerier) GetFamilyAccuracy(ctx context.Context, arg store.GetFamilyAccuracyParams) ([]store.GetFamilyAccuracyRow, error) {
	return s.getFamilyAccuracy(ctx, arg)
}
func (s *stubQuerier) GetHardMedia(ctx context.Context, arg store.GetHardMediaParams) ([]store.GetHardMediaRow, error) {
	return s.getHardMedia(ctx, arg)
}
func (s *stubQuerier) GetReviewAccuracy(ctx context.Context, arg store.GetReviewAccuracyParams) (store.GetReviewAccuracyRow, error) {
	return s.getReviewAccuracy(ctx, arg)
}
func (s *stubQuerier) CountReviewsSince(ctx context.Context, arg store.CountReviewsSinceParams) (int64, error) {
	return s.countReviewsSince(ctx, arg)
}
func (s *stubQuerier) GetCardsInTier(ctx context.Context, arg store.GetCardsInTierParams) ([]store.GetCardsInTierRow, error) {
	return s.getCardsInTier(ctx, arg)
}
func (s *stubQuerier) DeleteAllCardsForUser(ctx context.Context, userID int64) (int64, error) {
	return s.deleteAllCardsForUser(ctx, userID)
}
func (s *stubQuerier) DeleteAllReviewsForUser(ctx context.Context, userID int64) (int64, error) {
	return s.deleteAllReviewsForUser(ctx, userID)
}
func (s *stubQuerier) SeedCardsForUserDecks(ctx context.Context, userID int64) (int64, error) {
	return s.seedCardsForUserDecks(ctx, userID)
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
				AudioID:   "XC1",
				ImagePath: "https://r2.example.com/images/spotto/456.jpg",
				ImageID:   "ML1",
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
	assert.Equal(t, "XC1", body.MediaID, "audio lane should expose the recording ID")
}

func TestGetNextCard_Image_ExposesImageMediaID(t *testing.T) {
	due := pgtype.Timestamptz{}
	require.NoError(t, due.Scan(time.Now().Add(-time.Hour)))

	q := &stubQuerier{
		getDeck: deckOwnedBy(1),
		getNextDueCard: func(_ context.Context, arg store.GetNextDueCardParams) (store.GetNextDueCardRow, error) {
			assert.Equal(t, "image", arg.Lane)
			return store.GetNextDueCardRow{
				SpeciesCode:    "spotto",
				Lane:           "image",
				CommonName:     "Spotted Towhee",
				ScientificName: "Pipilo maculatus",
				Due:            due,
				DueRemaining:   1,
			}, nil
		},
		getRandomMedia: func(_ context.Context, speciesCode string) (store.GetRandomMediaForSpeciesRow, error) {
			return store.GetRandomMediaForSpeciesRow{
				AudioPath: "https://r2.example.com/recordings/spotto/123.mp3",
				AudioID:   "XC1",
				ImagePath: "https://r2.example.com/images/spotto/456.jpg",
				ImageID:   "ML1",
			}, nil
		},
	}

	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/decks/42/next?lane=image", nil)
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.getNextCard(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var body nextCardResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, "image", body.Lane)
	assert.Equal(t, "https://r2.example.com/images/spotto/456.jpg", body.MediaURL)
	assert.Equal(t, "ML1", body.MediaID, "image lane should expose the image's macaulay ID")
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
		createReviewLog: func(_ context.Context, _ store.CreateReviewLogParams) (store.ReviewLog, error) {
			return store.ReviewLog{}, nil
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
					AudioID:        "XC1",
					ImageUrl:       "https://r2.example.com/img.jpg",
					ImageID:        "ML1",
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
	assert.Equal(t, "XC1", body[0].MediaID, "audio practice lane should expose the recording ID")
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
					AudioID:        "XC1",
					ImageUrl:       "https://r2.example.com/img.jpg",
					ImageID:        "ML1",
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
	assert.Equal(t, "ML1", body[0].MediaID, "image practice lane should expose the image's macaulay ID")
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

func TestRateCard_WritesReviewLog(t *testing.T) {
	var logged store.CreateReviewLogParams
	q := &stubQuerier{
		getDeck: deckOwnedBy(1),
		getCard: func(_ context.Context, arg store.GetCardParams) (store.Card, error) {
			return store.Card{UserID: 1, SpeciesCode: arg.SpeciesCode, Lane: arg.Lane}, nil
		},
		updateCardSchedule: func(_ context.Context, arg store.UpdateCardScheduleParams) (store.Card, error) {
			return store.Card{}, nil
		},
		createReviewLog: func(_ context.Context, arg store.CreateReviewLogParams) (store.ReviewLog, error) {
			logged = arg
			return store.ReviewLog{}, nil
		},
	}
	h := makeHandler(q)
	body := `{"ebird_code":"sonspa","lane":"audio","rating":1,"guessed_species_code":"foxspa","media_id":"XC1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/decks/42/rate", strings.NewReader(body))
	req = injectUserID(req, 1)
	req = withChiParam(req, "id", "42")
	rec := httptest.NewRecorder()
	h.rateCard(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "Should return 200 OK")
	assert.Equal(t, "foxspa", logged.GuessedSpeciesCode.String, "Should log guessed_species_code")
	assert.True(t, logged.GuessedSpeciesCode.Valid, "Should mark guessed_species_code as valid")
	assert.Equal(t, "XC1", logged.MediaID.String, "Should log media_id")
	assert.True(t, logged.MediaID.Valid, "Should mark media_id as valid")
	assert.Equal(t, int16(1), logged.Rating, "Should log rating")
}

func TestRateCard_OmittedGuessAndMedia_LogsNulls(t *testing.T) {
	var logged store.CreateReviewLogParams
	q := &stubQuerier{
		getDeck: deckOwnedBy(1),
		getCard: func(_ context.Context, arg store.GetCardParams) (store.Card, error) {
			return store.Card{UserID: 1, SpeciesCode: arg.SpeciesCode, Lane: arg.Lane}, nil
		},
		updateCardSchedule: func(_ context.Context, arg store.UpdateCardScheduleParams) (store.Card, error) {
			return store.Card{}, nil
		},
		createReviewLog: func(_ context.Context, arg store.CreateReviewLogParams) (store.ReviewLog, error) {
			logged = arg
			return store.ReviewLog{}, nil
		},
	}
	h := makeHandler(q)
	// Old-client body without the new fields -- back-compat must work.
	body := `{"ebird_code":"sonspa","lane":"audio","rating":3}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/decks/42/rate", strings.NewReader(body))
	req = injectUserID(req, 1)
	req = withChiParam(req, "id", "42")
	rec := httptest.NewRecorder()
	h.rateCard(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "Should return 200 OK for old-client body")
	assert.False(t, logged.GuessedSpeciesCode.Valid, "Should log NULL when no guess provided")
	assert.False(t, logged.MediaID.Valid, "Should log NULL when no media_id provided")
}
