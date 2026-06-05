package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jameynakama/flockdeck/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func connectTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = os.Getenv("DATABASE_URL")
	}
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL/DATABASE_URL not set")
	}
	pool, err := pgxpool.New(context.Background(), dbURL)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	return pool
}

func TestInTx_RollsBackOnError(t *testing.T) {
	pool := connectTestPool(t)
	h := &Handler{queries: store.New(pool), db: pool}
	ctx := context.Background()
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM species WHERE ebird_code = '_txtest'`)
	})

	err := h.inTx(ctx, func(q store.Querier) error {
		if _, err := q.UpsertSpecies(ctx, store.UpsertSpeciesParams{
			EbirdCode: "_txtest", CommonName: "Tx Test", ScientificName: "Txus testus",
		}); err != nil {
			return err
		}
		return errors.New("boom")
	})

	require.Error(t, err)
	var n int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM species WHERE ebird_code = '_txtest'`).Scan(&n))
	assert.Equal(t, 0, n, "write inside a failed tx must roll back")
}

func TestInTx_CommitsOnSuccess(t *testing.T) {
	pool := connectTestPool(t)
	h := &Handler{queries: store.New(pool), db: pool}
	ctx := context.Background()
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM species WHERE ebird_code = '_txtest2'`)
	})

	err := h.inTx(ctx, func(q store.Querier) error {
		_, err := q.UpsertSpecies(ctx, store.UpsertSpeciesParams{
			EbirdCode: "_txtest2", CommonName: "Tx Test", ScientificName: "Txus testus",
		})
		return err
	})

	require.NoError(t, err)
	var n int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM species WHERE ebird_code = '_txtest2'`).Scan(&n))
	assert.Equal(t, 1, n)
}

func TestInTx_NoPool_FallsBackToPlainQuerier(t *testing.T) {
	// Unit-test configuration: no pool, stub querier -- fn runs non-atomically.
	called := false
	h := makeHandler(&stubQuerier{})

	err := h.inTx(context.Background(), func(q store.Querier) error {
		called = true
		assert.NotNil(t, q)
		return nil
	})

	require.NoError(t, err)
	assert.True(t, called)
}

// seedTxFixtures inserts a user, species, and card directly into the pool
// (outside a transaction so they're visible to subsequent connections).
// It returns the user ID and registers cleanup.
func seedTxFixtures(t *testing.T, pool *pgxpool.Pool) (userID int64) {
	t.Helper()
	ctx := context.Background()

	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO users (google_id, email, name, picture)
		 VALUES ('_txrat_gid', '_txrat@example.com', 'Tx Rat', '')
		 RETURNING id`,
	).Scan(&userID))

	_, err := pool.Exec(ctx,
		`INSERT INTO species (ebird_code, common_name, scientific_name)
		 VALUES ('_txrat', 'Tx Rat Species', 'Txus ratus')
		 ON CONFLICT DO NOTHING`)
	require.NoError(t, err)

	_, err = pool.Exec(ctx,
		`INSERT INTO cards (user_id, species_code, lane)
		 VALUES ($1, '_txrat', 'audio')
		 ON CONFLICT DO NOTHING`,
		userID)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM review_log WHERE species_code = '_txrat'`)
		_, _ = pool.Exec(ctx, `DELETE FROM cards WHERE species_code = '_txrat' AND user_id = $1`, userID)
		_, _ = pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
		_, _ = pool.Exec(ctx, `DELETE FROM species WHERE ebird_code = '_txrat'`)
	})

	return userID
}

func TestRateTx_LogFKViolation_RollsBackCardUpdate(t *testing.T) {
	pool := connectTestPool(t)
	h := &Handler{queries: store.New(pool), db: pool}
	ctx := context.Background()
	userID := seedTxFixtures(t, pool)

	due := pgtype.Timestamptz{}
	require.NoError(t, due.Scan(time.Now().Add(24*time.Hour)))

	err := h.inTx(ctx, func(q store.Querier) error {
		// Update the card to stability 99 -- should roll back if CreateReviewLog fails.
		if _, err := q.UpdateCardSchedule(ctx, store.UpdateCardScheduleParams{
			UserID:      userID,
			SpeciesCode: "_txrat",
			Lane:        "audio",
			Stability:   99,
			Difficulty:  5,
			Due:         due,
			Lapses:      0,
			State:       2,
		}); err != nil {
			return err
		}
		// Nonexistent guessed_species_code triggers FK violation -> rollback.
		_, err := q.CreateReviewLog(ctx, store.CreateReviewLogParams{
			UserID:             userID,
			SpeciesCode:        "_txrat",
			Lane:               "audio",
			Rating:             1,
			GuessedSpeciesCode: pgtype.Text{String: "nonexistent_code", Valid: true},
		})
		return err
	})

	require.Error(t, err, "Should return error on FK violation")

	// Card stability must NOT be 99 -- the rollback covered both writes.
	card, err := store.New(pool).GetCard(ctx, store.GetCardParams{
		UserID:      userID,
		SpeciesCode: "_txrat",
		Lane:        "audio",
	})
	require.NoError(t, err)
	assert.NotEqual(t, float64(99), card.Stability, "Should have rolled back card update on FK violation")
}

// seedTxDeck inserts a deck owned by userID and returns its ID, with cleanup.
func seedTxDeck(t *testing.T, pool *pgxpool.Pool, userID int64) int64 {
	t.Helper()
	ctx := context.Background()
	var deckID int64
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO decks (name, description, owner_id) VALUES ('_txrat deck', '', $1) RETURNING id`,
		userID,
	).Scan(&deckID))
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM decks WHERE id = $1`, deckID)
	})
	return deckID
}

func TestRateCard_UnknownGuess_Returns400(t *testing.T) {
	pool := connectTestPool(t)
	h := &Handler{queries: store.New(pool), db: pool}
	userID := seedTxFixtures(t, pool)
	deckID := seedTxDeck(t, pool, userID)

	// Fetch the card before rating so we can verify stability is unchanged.
	ctx := context.Background()
	before, err := store.New(pool).GetCard(ctx, store.GetCardParams{
		UserID:      userID,
		SpeciesCode: "_txrat",
		Lane:        "audio",
	})
	require.NoError(t, err)

	body := `{"ebird_code":"_txrat","lane":"audio","rating":1,"guessed_species_code":"zz_not_real"}`
	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/v1/decks/%d/rate", deckID),
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = injectUserID(req, userID)
	req = withChiParam(req, "id", fmt.Sprintf("%d", deckID))
	rec := httptest.NewRecorder()

	h.rateCard(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code, "Should return 400 for unknown guessed_species_code")
	assert.Contains(t, rec.Body.String(), "unknown guessed_species_code", "Should mention the unknown guess")

	// Card schedule must NOT have been updated -- tx rolled back.
	after, err := store.New(pool).GetCard(ctx, store.GetCardParams{
		UserID:      userID,
		SpeciesCode: "_txrat",
		Lane:        "audio",
	})
	require.NoError(t, err)
	assert.Equal(t, before.Stability, after.Stability, "Should not have updated card stability on bad guess")
}

func TestRateTx_ValidGuess_CommitsBothWrites(t *testing.T) {
	pool := connectTestPool(t)
	h := &Handler{queries: store.New(pool), db: pool}
	ctx := context.Background()
	userID := seedTxFixtures(t, pool)

	due := pgtype.Timestamptz{}
	require.NoError(t, due.Scan(time.Now().Add(24*time.Hour)))

	err := h.inTx(ctx, func(q store.Querier) error {
		if _, err := q.UpdateCardSchedule(ctx, store.UpdateCardScheduleParams{
			UserID:      userID,
			SpeciesCode: "_txrat",
			Lane:        "audio",
			Stability:   7,
			Difficulty:  5,
			Due:         due,
			Lapses:      0,
			State:       2,
		}); err != nil {
			return err
		}
		// Valid guess: same species that exists in fixtures.
		_, err := q.CreateReviewLog(ctx, store.CreateReviewLogParams{
			UserID:             userID,
			SpeciesCode:        "_txrat",
			Lane:               "audio",
			Rating:             3,
			GuessedSpeciesCode: pgtype.Text{String: "_txrat", Valid: true},
		})
		return err
	})

	require.NoError(t, err, "Should commit without error for valid guess")

	// Card stability must be 7.
	card, err := store.New(pool).GetCard(ctx, store.GetCardParams{
		UserID:      userID,
		SpeciesCode: "_txrat",
		Lane:        "audio",
	})
	require.NoError(t, err)
	assert.Equal(t, float64(7), card.Stability, "Should have committed card update")

	// Exactly one review_log row for this user+species.
	var n int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM review_log WHERE user_id = $1 AND species_code = '_txrat'`,
		userID).Scan(&n))
	assert.Equal(t, 1, n, "Should have exactly one review_log row")
}
