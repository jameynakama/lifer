package store_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jameynakama/flockdeck/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func connectTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set")
	}
	pool, err := pgxpool.New(context.Background(), dbURL)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	return pool
}

// withTx begins a transaction and rolls it back at test cleanup -- leaves DB clean.
func withTx(t *testing.T, pool *pgxpool.Pool) pgx.Tx {
	t.Helper()
	tx, err := pool.Begin(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx.Rollback(context.Background()) })
	return tx
}

type fixtures struct {
	userID int64
	deckID int64
}

// seedFixtures inserts a user, a fake species, a deck, deck membership, and
// audio+image cards (all due immediately) into the given transaction.
func seedFixtures(t *testing.T, tx pgx.Tx) fixtures {
	t.Helper()
	ctx := context.Background()

	var userID int64
	err := tx.QueryRow(ctx,
		`INSERT INTO users (google_id, email, name, picture)
		 VALUES ('_test_pref_gid', '_test_pref@example.com', 'Pref Test', '')
		 RETURNING id`,
	).Scan(&userID)
	require.NoError(t, err)

	_, err = tx.Exec(ctx,
		`INSERT INTO species (ebird_code, common_name, scientific_name)
		 VALUES ('_tst1', 'Test Species', 'Testus specius')`,
	)
	require.NoError(t, err)

	var deckID int64
	err = tx.QueryRow(ctx,
		`INSERT INTO decks (name, owner_id) VALUES ('Test Deck', $1) RETURNING id`,
		userID,
	).Scan(&deckID)
	require.NoError(t, err)

	_, err = tx.Exec(ctx,
		`INSERT INTO deck_species (deck_id, species_code) VALUES ($1, '_tst1')`, deckID,
	)
	require.NoError(t, err)

	// due defaults to NOW(), so both cards are immediately due.
	_, err = tx.Exec(ctx,
		`INSERT INTO cards (user_id, species_code, lane)
		 VALUES ($1, '_tst1', 'audio'), ($1, '_tst1', 'image')`,
		userID,
	)
	require.NoError(t, err)

	return fixtures{userID: userID, deckID: deckID}
}

func disableImageLane(t *testing.T, tx pgx.Tx, userID int64) {
	t.Helper()
	_, err := tx.Exec(context.Background(),
		`INSERT INTO user_species_preferences (user_id, species_code, audio_enabled, image_enabled)
		 VALUES ($1, '_tst1', true, false)`,
		userID,
	)
	require.NoError(t, err)
}

// GetNextDueCard

func TestGetNextDueCard_NoPreference_ReturnsDueCard(t *testing.T) {
	pool := connectTestDB(t)
	tx := withTx(t, pool)
	f := seedFixtures(t, tx)
	q := store.New(tx)

	card, err := q.GetNextDueCard(context.Background(), store.GetNextDueCardParams{
		UserID: f.userID,
		DeckID: f.deckID,
		Lane:   "image",
	})
	require.NoError(t, err)
	assert.Equal(t, "_tst1", card.SpeciesCode)
}

func TestGetNextDueCard_ImageDisabled_ReturnsNoRows(t *testing.T) {
	pool := connectTestDB(t)
	tx := withTx(t, pool)
	f := seedFixtures(t, tx)
	disableImageLane(t, tx, f.userID)
	q := store.New(tx)

	_, err := q.GetNextDueCard(context.Background(), store.GetNextDueCardParams{
		UserID: f.userID,
		DeckID: f.deckID,
		Lane:   "image",
	})
	assert.True(t, errors.Is(err, pgx.ErrNoRows), "want ErrNoRows when image lane is disabled, got %v", err)
}

func TestGetNextDueCard_ImageDisabled_AudioStillReturnsCard(t *testing.T) {
	pool := connectTestDB(t)
	tx := withTx(t, pool)
	f := seedFixtures(t, tx)
	disableImageLane(t, tx, f.userID)
	q := store.New(tx)

	card, err := q.GetNextDueCard(context.Background(), store.GetNextDueCardParams{
		UserID: f.userID,
		DeckID: f.deckID,
		Lane:   "audio",
	})
	require.NoError(t, err)
	assert.Equal(t, "_tst1", card.SpeciesCode)
}

// CountDueCards

func TestCountDueCards_NoPreference_CountsCard(t *testing.T) {
	pool := connectTestDB(t)
	tx := withTx(t, pool)
	f := seedFixtures(t, tx)
	q := store.New(tx)

	count, err := q.CountDueCards(context.Background(), store.CountDueCardsParams{
		UserID: f.userID,
		DeckID: f.deckID,
		Lane:   "image",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestCountDueCards_ImageDisabled_ReturnsZero(t *testing.T) {
	pool := connectTestDB(t)
	tx := withTx(t, pool)
	f := seedFixtures(t, tx)
	disableImageLane(t, tx, f.userID)
	q := store.New(tx)

	count, err := q.CountDueCards(context.Background(), store.CountDueCardsParams{
		UserID: f.userID,
		DeckID: f.deckID,
		Lane:   "image",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}
