package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jameynakama/flockdeck/internal/store"
)

// --- unit tests (stub querier) ---

func postReset(h *Handler, userID int64, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/reset", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = injectUserID(req, userID)
	rec := httptest.NewRecorder()
	h.resetUserData(rec, req)
	return rec
}

func TestResetUserData_MissingScope_400(t *testing.T) {
	h := makeHandler(&stubQuerier{})
	rec := postReset(h, 1, `{}`)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "scope must be")
}

func TestResetUserData_UnknownScope_400(t *testing.T) {
	h := makeHandler(&stubQuerier{})
	rec := postReset(h, 1, `{"scope":"sideways"}`)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "scope must be")
}

func TestResetUserData_MalformedJSON_400(t *testing.T) {
	h := makeHandler(&stubQuerier{})
	rec := postReset(h, 1, `{"scope":`)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestResetUserData_Schedule_DeletesOnlyCards(t *testing.T) {
	reviewsCalled := false
	q := &stubQuerier{
		deleteAllCardsForUser: func(_ context.Context, userID int64) (int64, error) {
			assert.Equal(t, int64(7), userID)
			return 42, nil
		},
		deleteAllReviewsForUser: func(_ context.Context, _ int64) (int64, error) {
			reviewsCalled = true
			return 0, nil
		},
	}
	rec := postReset(makeHandler(q), 7, `{"scope":"schedule"}`)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.False(t, reviewsCalled, "schedule scope must not touch review_log")
	assert.JSONEq(t, `{"cards_deleted":42,"reviews_deleted":0}`, rec.Body.String())
}

func TestResetUserData_Everything_DeletesBoth(t *testing.T) {
	q := &stubQuerier{
		deleteAllCardsForUser: func(_ context.Context, userID int64) (int64, error) {
			assert.Equal(t, int64(7), userID)
			return 42, nil
		},
		deleteAllReviewsForUser: func(_ context.Context, userID int64) (int64, error) {
			assert.Equal(t, int64(7), userID)
			return 99, nil
		},
	}
	rec := postReset(makeHandler(q), 7, `{"scope":"everything"}`)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"cards_deleted":42,"reviews_deleted":99}`, rec.Body.String())
}

func TestResetUserData_DBError_500(t *testing.T) {
	q := &stubQuerier{
		deleteAllCardsForUser: func(_ context.Context, _ int64) (int64, error) {
			return 0, fmt.Errorf("boom")
		},
	}
	rec := postReset(makeHandler(q), 7, `{"scope":"schedule"}`)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// --- integration tests (real test DB; skip without one) ---

// seedResetFixtures inserts a user with one card and one review_log row,
// plus a deck containing the species, and registers cleanup. The suffix
// disambiguates the two users in the scoping test.
func seedResetFixtures(t *testing.T, pool *pgxpool.Pool, suffix string) (userID int64) {
	t.Helper()
	ctx := context.Background()

	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO users (google_id, email, name, picture)
		 VALUES ('_rst_gid_`+suffix+`', '_rst_`+suffix+`@example.com', 'Reset Test', '')
		 RETURNING id`,
	).Scan(&userID))

	_, err := pool.Exec(ctx,
		`INSERT INTO species (ebird_code, common_name, scientific_name)
		 VALUES ('_rst', 'Reset Test Species', 'Resetus testus')
		 ON CONFLICT DO NOTHING`)
	require.NoError(t, err)

	_, err = pool.Exec(ctx,
		`INSERT INTO cards (user_id, species_code, lane)
		 VALUES ($1, '_rst', 'audio') ON CONFLICT DO NOTHING`, userID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx,
		`INSERT INTO review_log (user_id, species_code, lane, rating)
		 VALUES ($1, '_rst', 'audio', 3)`, userID)
	require.NoError(t, err)

	var deckID int64
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO decks (name, description, owner_id)
		 VALUES ('_rst deck', '', $1) RETURNING id`, userID).Scan(&deckID))
	_, err = pool.Exec(ctx,
		`INSERT INTO deck_species (deck_id, species_code) VALUES ($1, '_rst')`, deckID)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM review_log WHERE user_id = $1`, userID)
		_, _ = pool.Exec(ctx, `DELETE FROM cards WHERE user_id = $1`, userID)
		_, _ = pool.Exec(ctx, `DELETE FROM decks WHERE id = $1`, deckID)
		_, _ = pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
		_, _ = pool.Exec(ctx, `DELETE FROM species WHERE ebird_code = '_rst'`)
	})

	return userID
}

func countRows(t *testing.T, pool *pgxpool.Pool, query string, args ...any) int {
	t.Helper()
	var n int
	require.NoError(t, pool.QueryRow(context.Background(), query, args...).Scan(&n))
	return n
}

func TestResetUserData_Schedule_DB_LeavesReviewLog(t *testing.T) {
	pool := connectTestPool(t)
	h := &Handler{queries: store.New(pool), db: pool}
	userID := seedResetFixtures(t, pool, "a")

	rec := postReset(h, userID, `{"scope":"schedule"}`)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, countRows(t, pool,
		`SELECT COUNT(*) FROM cards WHERE user_id = $1`, userID))
	assert.Equal(t, 1, countRows(t, pool,
		`SELECT COUNT(*) FROM review_log WHERE user_id = $1`, userID),
		"schedule reset must leave review history intact")
}

func TestResetUserData_Everything_DB_WipesBoth_SparesOtherUsersAndDecks(t *testing.T) {
	pool := connectTestPool(t)
	h := &Handler{queries: store.New(pool), db: pool}
	userID := seedResetFixtures(t, pool, "b")
	otherID := seedResetFixtures(t, pool, "c")
	deckSpeciesBefore := countRows(t, pool, `SELECT COUNT(*) FROM deck_species`)

	rec := postReset(h, userID, `{"scope":"everything"}`)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"cards_deleted":1,"reviews_deleted":1}`, rec.Body.String(),
		"counts must reflect the rows actually deleted")

	// Target user wiped.
	assert.Equal(t, 0, countRows(t, pool,
		`SELECT COUNT(*) FROM cards WHERE user_id = $1`, userID))
	assert.Equal(t, 0, countRows(t, pool,
		`SELECT COUNT(*) FROM review_log WHERE user_id = $1`, userID))

	// Other user untouched.
	assert.Equal(t, 1, countRows(t, pool,
		`SELECT COUNT(*) FROM cards WHERE user_id = $1`, otherID),
		"reset must be scoped to the requesting user")
	assert.Equal(t, 1, countRows(t, pool,
		`SELECT COUNT(*) FROM review_log WHERE user_id = $1`, otherID))

	// Deck membership untouched (the spec's hard guarantee).
	assert.Equal(t, deckSpeciesBefore, countRows(t, pool, `SELECT COUNT(*) FROM deck_species`),
		"reset must never touch deck contents")
}
