package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jameynakama/flockdeck/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetrievability_FreshCardIsNearOne(t *testing.T) {
	now := time.Now()
	r := retrievability(10 /* stability days */, now.Add(-1*time.Hour), now)
	assert.Greater(t, r, 0.97, "Should be near-perfect recall right after review")
}

func TestRetrievability_DecaysOverTime(t *testing.T) {
	now := time.Now()
	fresh := retrievability(10, now.Add(-24*time.Hour), now)
	stale := retrievability(10, now.Add(-30*24*time.Hour), now)
	assert.Greater(t, fresh, stale, "Recall should decay with elapsed time")
	assert.Greater(t, stale, 0.0)
	assert.Less(t, stale, 1.0)
}

func TestExpectedRecall_SumsRetrievabilities(t *testing.T) {
	now := time.Now()
	cards := []store.GetKnownCardsRow{
		{Stability: 10, LastReview: tstz(t, now.Add(-24*time.Hour))},
		{Stability: 10, LastReview: tstz(t, now.Add(-24*time.Hour))},
	}
	atNow := expectedRecall(cards, now)
	atMonth := expectedRecall(cards, now.Add(30*24*time.Hour))
	assert.Equal(t, 2, atNow, "Two fresh cards round to 2 remembered")
	assert.LessOrEqual(t, atMonth, atNow)
}

func tstz(t *testing.T, ts time.Time) pgtype.Timestamptz {
	t.Helper()
	var v pgtype.Timestamptz
	require.NoError(t, v.Scan(ts))
	return v
}

func TestGetStats_BadLane_400(t *testing.T) {
	h := makeHandler(&stubQuerier{})
	r := httptest.NewRequest(http.MethodGet, "/api/v1/stats?lane=sonar", nil)
	r = injectUserID(r, 1)
	w := httptest.NewRecorder()

	h.getStats(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code, "Should return 400 for invalid lane")
}

func TestGetStats_CombinedShape(t *testing.T) {
	now := time.Now()
	q := &stubQuerier{
		getCardTotals: func(_ context.Context, _ store.GetCardTotalsParams) (store.GetCardTotalsRow, error) {
			return store.GetCardTotalsRow{Species: 1, Cards: 2, Reviews: 5, Lapses: 1}, nil
		},
		getCardStateCounts: func(_ context.Context, _ store.GetCardStateCountsParams) ([]store.GetCardStateCountsRow, error) {
			return []store.GetCardStateCountsRow{{Bucket: "known", Count: 1}}, nil
		},
		getKnownCards: func(_ context.Context, _ store.GetKnownCardsParams) ([]store.GetKnownCardsRow, error) {
			return []store.GetKnownCardsRow{
				{
					SpeciesCode:    "spotto",
					CommonName:     "Spotted Towhee",
					ScientificName: "Pipilo maculatus",
					Lane:           "audio",
					Stability:      10,
					LastReview:     tstz(t, now.Add(-24*time.Hour)),
					Due:            tstz(t, now.Add(9*24*time.Hour)),
				},
			}, nil
		},
		getLaneGaps: func(_ context.Context, _ int64) ([]store.GetLaneGapsRow, error) {
			return []store.GetLaneGapsRow{
				{
					SpeciesCode: "spotto", CommonName: "Spotted Towhee",
					ScientificName: "Pipilo maculatus",
					KnownLane:      "audio", WeakLane: "image", StabilityGap: 5.0,
				},
			}, nil
		},
		getConfusionPairs: func(_ context.Context, _ store.GetConfusionPairsParams) ([]store.GetConfusionPairsRow, error) {
			return []store.GetConfusionPairsRow{
				{
					SpeciesCode: "spotto", ActualCommonName: "Spotted Towhee",
					ActualScientificName: "Pipilo maculatus",
					GuessedSpeciesCode:   "ruwhe1", GuessedCommonName: "Rufous-winged",
					GuessedScientificName: "Peucaea carpalis", Count: 2,
				},
			}, nil
		},
		getFamilyAccuracy: func(_ context.Context, _ store.GetFamilyAccuracyParams) ([]store.GetFamilyAccuracyRow, error) {
			return []store.GetFamilyAccuracyRow{{Family: "Waterfowl", Attempts: 4, Correct: 1}}, nil
		},
		getHardMedia: func(_ context.Context, _ store.GetHardMediaParams) ([]store.GetHardMediaRow, error) {
			return []store.GetHardMediaRow{
				{
					MediaID: "XC1", Lane: "audio", SpeciesCode: "spotto",
					CommonName: "Spotted Towhee", ScientificName: "Pipilo maculatus",
					MediaUrl: "https://media.flockdeck.com/rec.mp3", Attempts: 5, Correct: 1,
				},
			}, nil
		},
		getReviewAccuracy: func(_ context.Context, _ store.GetReviewAccuracyParams) (store.GetReviewAccuracyRow, error) {
			return store.GetReviewAccuracyRow{Attempts: 4, Correct: 3}, nil
		},
		countReviewsSince: func(_ context.Context, _ store.CountReviewsSinceParams) (int64, error) {
			return 2, nil
		},
	}

	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/stats", nil)
	r = injectUserID(r, 1)
	w := httptest.NewRecorder()

	h.getStats(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var resp statsResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, int64(1), resp.Totals.Species, "Should return species count")
	assert.NotNil(t, resp.Lanes, "Combined mode should include Lanes block")
	assert.Len(t, resp.Confusions, 1, "Should return one confusion pair")
	assert.GreaterOrEqual(t, resp.Remember.Now, 1, "Should have at least 1 remembered card")
	assert.Equal(t, int64(2), resp.Totals.ReviewsLast7d, "Should return reviews last 7 days")
	assert.Equal(t, int64(4), resp.Totals.Attempts, "Should return total attempts")
}

func TestGetStats_LaneTab_OmitsLanesBlock(t *testing.T) {
	now := time.Now()
	q := &stubQuerier{
		getCardTotals: func(_ context.Context, _ store.GetCardTotalsParams) (store.GetCardTotalsRow, error) {
			return store.GetCardTotalsRow{Species: 1, Cards: 2, Reviews: 5, Lapses: 1}, nil
		},
		getCardStateCounts: func(_ context.Context, _ store.GetCardStateCountsParams) ([]store.GetCardStateCountsRow, error) {
			return []store.GetCardStateCountsRow{{Bucket: "known", Count: 1}}, nil
		},
		getKnownCards: func(_ context.Context, _ store.GetKnownCardsParams) ([]store.GetKnownCardsRow, error) {
			return []store.GetKnownCardsRow{
				{
					SpeciesCode: "spotto", Lane: "audio", Stability: 10,
					LastReview: tstz(t, now.Add(-24*time.Hour)),
					Due:        tstz(t, now.Add(9*24*time.Hour)),
				},
			}, nil
		},
		getConfusionPairs: func(_ context.Context, _ store.GetConfusionPairsParams) ([]store.GetConfusionPairsRow, error) {
			return []store.GetConfusionPairsRow{}, nil
		},
		getFamilyAccuracy: func(_ context.Context, _ store.GetFamilyAccuracyParams) ([]store.GetFamilyAccuracyRow, error) {
			return []store.GetFamilyAccuracyRow{}, nil
		},
		getHardMedia: func(_ context.Context, _ store.GetHardMediaParams) ([]store.GetHardMediaRow, error) {
			return []store.GetHardMediaRow{}, nil
		},
		getReviewAccuracy: func(_ context.Context, _ store.GetReviewAccuracyParams) (store.GetReviewAccuracyRow, error) {
			return store.GetReviewAccuracyRow{}, nil
		},
		countReviewsSince: func(_ context.Context, _ store.CountReviewsSinceParams) (int64, error) {
			return 0, nil
		},
	}

	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/stats?lane=audio", nil)
	r = injectUserID(r, 1)
	w := httptest.NewRecorder()

	h.getStats(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var resp statsResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Nil(t, resp.Lanes, "Lane-filtered mode should omit Lanes block")
}

func TestGetStats_FadingSortedWorstFirst(t *testing.T) {
	now := time.Now()
	q := &stubQuerier{
		getCardTotals: func(_ context.Context, _ store.GetCardTotalsParams) (store.GetCardTotalsRow, error) {
			return store.GetCardTotalsRow{}, nil
		},
		getCardStateCounts: func(_ context.Context, _ store.GetCardStateCountsParams) ([]store.GetCardStateCountsRow, error) {
			return []store.GetCardStateCountsRow{}, nil
		},
		getKnownCards: func(_ context.Context, _ store.GetKnownCardsParams) ([]store.GetKnownCardsRow, error) {
			return []store.GetKnownCardsRow{
				{
					SpeciesCode: "spotto", CommonName: "Spotted Towhee",
					ScientificName: "Pipilo maculatus", Lane: "audio",
					Stability:  10,
					LastReview: tstz(t, now.Add(-1*time.Hour)),
					Due:        tstz(t, now.Add(9*24*time.Hour)),
				},
				{
					SpeciesCode: "foxspa", CommonName: "Fox Sparrow",
					ScientificName: "Passerella iliaca", Lane: "audio",
					Stability:  10,
					LastReview: tstz(t, now.Add(-30*24*time.Hour)),
					Due:        tstz(t, now.Add(-20*24*time.Hour)),
				},
			}, nil
		},
		getLaneGaps: func(_ context.Context, _ int64) ([]store.GetLaneGapsRow, error) {
			return []store.GetLaneGapsRow{}, nil
		},
		getConfusionPairs: func(_ context.Context, _ store.GetConfusionPairsParams) ([]store.GetConfusionPairsRow, error) {
			return []store.GetConfusionPairsRow{}, nil
		},
		getFamilyAccuracy: func(_ context.Context, _ store.GetFamilyAccuracyParams) ([]store.GetFamilyAccuracyRow, error) {
			return []store.GetFamilyAccuracyRow{}, nil
		},
		getHardMedia: func(_ context.Context, _ store.GetHardMediaParams) ([]store.GetHardMediaRow, error) {
			return []store.GetHardMediaRow{}, nil
		},
		getReviewAccuracy: func(_ context.Context, _ store.GetReviewAccuracyParams) (store.GetReviewAccuracyRow, error) {
			return store.GetReviewAccuracyRow{}, nil
		},
		countReviewsSince: func(_ context.Context, _ store.CountReviewsSinceParams) (int64, error) {
			return 0, nil
		},
	}

	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/stats", nil)
	r = injectUserID(r, 1)
	w := httptest.NewRecorder()

	h.getStats(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var resp statsResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	require.Len(t, resp.Fading, 2, "Should return both fading cards")
	assert.Equal(t, "foxspa", resp.Fading[0].EbirdCode, "Should sort fading worst-first (oldest review)")
	assert.Greater(t, resp.Fading[1].Retrievability, resp.Fading[0].Retrievability,
		"First fading card should have lower retrievability than second")
	for _, f := range resp.Fading {
		assert.Greater(t, f.Retrievability, 0.0, "Should have retrievability > 0")
		assert.Less(t, f.Retrievability, 1.0, "Should have retrievability < 1")
	}
}

func TestGetStats_WireContract(t *testing.T) {
	now := time.Now()
	q := &stubQuerier{
		getCardTotals: func(_ context.Context, _ store.GetCardTotalsParams) (store.GetCardTotalsRow, error) {
			return store.GetCardTotalsRow{Species: 1, Cards: 2, Reviews: 5, Lapses: 1}, nil
		},
		getCardStateCounts: func(_ context.Context, _ store.GetCardStateCountsParams) ([]store.GetCardStateCountsRow, error) {
			return []store.GetCardStateCountsRow{{Bucket: "known", Count: 1}}, nil
		},
		getKnownCards: func(_ context.Context, _ store.GetKnownCardsParams) ([]store.GetKnownCardsRow, error) {
			return []store.GetKnownCardsRow{
				{
					SpeciesCode:    "spotto",
					CommonName:     "Spotted Towhee",
					ScientificName: "Pipilo maculatus",
					Lane:           "audio",
					Stability:      10,
					LastReview:     tstz(t, now.Add(-24*time.Hour)),
					Due:            tstz(t, now.Add(9*24*time.Hour)),
				},
			}, nil
		},
		getLaneGaps: func(_ context.Context, _ int64) ([]store.GetLaneGapsRow, error) {
			return []store.GetLaneGapsRow{
				{
					SpeciesCode: "spotto", CommonName: "Spotted Towhee",
					ScientificName: "Pipilo maculatus",
					KnownLane:      "audio", WeakLane: "image", StabilityGap: 5.0,
				},
			}, nil
		},
		getConfusionPairs: func(_ context.Context, _ store.GetConfusionPairsParams) ([]store.GetConfusionPairsRow, error) {
			return []store.GetConfusionPairsRow{
				{
					SpeciesCode: "spotto", ActualCommonName: "Spotted Towhee",
					ActualScientificName: "Pipilo maculatus",
					GuessedSpeciesCode:   "ruwhe1", GuessedCommonName: "Rufous-winged",
					GuessedScientificName: "Peucaea carpalis", Count: 2,
				},
			}, nil
		},
		getFamilyAccuracy: func(_ context.Context, _ store.GetFamilyAccuracyParams) ([]store.GetFamilyAccuracyRow, error) {
			return []store.GetFamilyAccuracyRow{{Family: "Waterfowl", Attempts: 4, Correct: 1}}, nil
		},
		getHardMedia: func(_ context.Context, _ store.GetHardMediaParams) ([]store.GetHardMediaRow, error) {
			return []store.GetHardMediaRow{
				{
					MediaID: "XC1", Lane: "audio", SpeciesCode: "spotto",
					CommonName: "Spotted Towhee", ScientificName: "Pipilo maculatus",
					MediaUrl: "https://media.flockdeck.com/rec.mp3", Attempts: 5, Correct: 1,
				},
			}, nil
		},
		getReviewAccuracy: func(_ context.Context, _ store.GetReviewAccuracyParams) (store.GetReviewAccuracyRow, error) {
			return store.GetReviewAccuracyRow{Attempts: 4, Correct: 3}, nil
		},
		countReviewsSince: func(_ context.Context, _ store.CountReviewsSinceParams) (int64, error) {
			return 2, nil
		},
	}

	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/stats", nil)
	r = injectUserID(r, 1)
	rec := httptest.NewRecorder()

	h.getStats(rec, r)

	require.Equal(t, http.StatusOK, rec.Code)

	var raw map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &raw))

	for _, key := range []string{"totals", "progress", "lanes", "confusions", "families", "fading", "remember", "hard_media"} {
		assert.Contains(t, raw, key, "Should expose top-level key %q", key)
	}
	totals := raw["totals"].(map[string]any)
	for _, key := range []string{"species", "cards", "known", "reviews", "lapses", "attempts", "correct", "reviews_last_7d"} {
		assert.Contains(t, totals, key, "Should expose totals key %q", key)
	}
	remember := raw["remember"].(map[string]any)
	for _, key := range []string{"now", "in_a_week", "in_a_month"} {
		assert.Contains(t, remember, key)
	}
	progress := raw["progress"].(map[string]any)
	for _, key := range []string{"not_seen", "learning", "known", "relearning"} {
		assert.Contains(t, progress, key)
	}
	fading := raw["fading"].([]any)
	require.NotEmpty(t, fading)
	f0 := fading[0].(map[string]any)
	for _, key := range []string{"ebird_code", "common_name", "scientific_name", "lane", "retrievability", "due_in_days"} {
		assert.Contains(t, f0, key)
	}
	hard := raw["hard_media"].([]any)
	require.NotEmpty(t, hard)
	h0 := hard[0].(map[string]any)
	for _, key := range []string{"media_id", "media_url", "attempts", "correct"} {
		assert.Contains(t, h0, key)
	}
}
