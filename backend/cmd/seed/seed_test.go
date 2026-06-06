package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func connectSeedTestPool(t *testing.T) *pgxpool.Pool {
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

// seedSeedFixtures inserts a user with one deck holding two same-family
// species, and registers cleanup.
func seedSeedFixtures(t *testing.T, pool *pgxpool.Pool) (userID int64) {
	t.Helper()
	ctx := context.Background()

	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO users (google_id, email, name, picture)
		 VALUES ('_seed_gid', '_seed@example.com', 'Seed Test', '')
		 RETURNING id`).Scan(&userID))

	_, err := pool.Exec(ctx,
		`INSERT INTO species (ebird_code, common_name, scientific_name, family)
		 VALUES ('_sd1', 'Seed Sparrow One', 'Seedus unus', 'Seed Sparrows'),
		        ('_sd2', 'Seed Sparrow Two', 'Seedus duo',  'Seed Sparrows')
		 ON CONFLICT DO NOTHING`)
	require.NoError(t, err)

	_, err = pool.Exec(ctx,
		`INSERT INTO species_recordings (xeno_canto_id, species_code, file_path, quality, type, credit)
		 VALUES ('_sdxc1', '_sd1', 'placeholder://sd1.mp3', 'A', 'song', 'test'),
		        ('_sdxc2', '_sd2', 'placeholder://sd2.mp3', 'A', 'song', 'test')
		 ON CONFLICT DO NOTHING`)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO species_images (macaulay_id, species_code, file_path, credit)
		 VALUES ('_sdml1', '_sd1', 'placeholder://sd1.jpg', 'test'),
		        ('_sdml2', '_sd2', 'placeholder://sd2.jpg', 'test')
		 ON CONFLICT DO NOTHING`)
	require.NoError(t, err)

	var deckID int64
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO decks (name, description, owner_id)
		 VALUES ('_seed deck', '', $1) RETURNING id`, userID).Scan(&deckID))
	_, err = pool.Exec(ctx,
		`INSERT INTO deck_species (deck_id, species_code)
		 VALUES ($1, '_sd1'), ($1, '_sd2')`, deckID)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM review_log WHERE user_id = $1`, userID)
		_, _ = pool.Exec(ctx, `DELETE FROM cards WHERE user_id = $1`, userID)
		_, _ = pool.Exec(ctx, `DELETE FROM decks WHERE id = $1`, deckID)
		_, _ = pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
		_, _ = pool.Exec(ctx, `DELETE FROM species_recordings WHERE species_code IN ('_sd1', '_sd2')`)
		_, _ = pool.Exec(ctx, `DELETE FROM species_images WHERE species_code IN ('_sd1', '_sd2')`)
		_, _ = pool.Exec(ctx, `DELETE FROM species WHERE ebird_code IN ('_sd1', '_sd2')`)
	})

	return userID
}

func countOne(t *testing.T, pool *pgxpool.Pool, q string, args ...any) int {
	t.Helper()
	var n int
	require.NoError(t, pool.QueryRow(context.Background(), q, args...).Scan(&n))
	return n
}

func TestRunSeed_ProducesConsistentBackdatedHistory(t *testing.T) {
	pool := connectSeedTestPool(t)
	userID := seedSeedFixtures(t, pool)
	ctx := context.Background()

	res, err := runSeed(ctx, pool, userID, 5, rand.New(rand.NewSource(42)), time.Now())
	require.NoError(t, err)
	assert.Equal(t, int64(4), res.Seeded, "2 species x 2 lanes")
	require.Greater(t, res.Reviews, 0, "seed 42 must produce reviews; if a future change alters rng consumption order, pick a seed that does")

	// Every review_log row is backdated.
	assert.Equal(t, res.Reviews, countOne(t, pool,
		`SELECT COUNT(*) FROM review_log WHERE user_id = $1 AND reviewed_at < NOW()`, userID))

	// Card reps reconcile exactly with logged reviews.
	assert.Equal(t, res.Reviews, countOne(t, pool,
		`SELECT COALESCE(SUM(reps), 0) FROM cards WHERE user_id = $1`, userID),
		"FSRS card state must be the product of the logged history")

	// last_review is backdated too (the app's UpdateCardSchedule would have stamped NOW()).
	assert.Equal(t, 0, countOne(t, pool,
		`SELECT COUNT(*) FROM cards WHERE user_id = $1 AND reps > 0 AND last_review >= NOW()`, userID))

	// Wrong guesses only ever name the other deck species or NULL.
	assert.Equal(t, 0, countOne(t, pool,
		`SELECT COUNT(*) FROM review_log
		 WHERE user_id = $1 AND rating = 1
		   AND guessed_species_code IS NOT NULL
		   AND guessed_species_code NOT IN ('_sd1', '_sd2')`, userID))

	// Every logged review carries the real media id for its lane.
	assert.Equal(t, 0, countOne(t, pool,
		`SELECT COUNT(*) FROM review_log WHERE user_id = $1 AND media_id IS NULL`, userID))
	assert.Equal(t, 0, countOne(t, pool,
		`SELECT COUNT(*) FROM review_log
		 WHERE user_id = $1 AND lane = 'audio' AND media_id NOT IN ('_sdxc1', '_sdxc2')`, userID))
	assert.Equal(t, 0, countOne(t, pool,
		`SELECT COUNT(*) FROM review_log
		 WHERE user_id = $1 AND lane = 'image' AND media_id NOT IN ('_sdml1', '_sdml2')`, userID))

	// Rerunning resets first: history is fresh, not doubled.
	res2, err := runSeed(ctx, pool, userID, 5, rand.New(rand.NewSource(43)), time.Now())
	require.NoError(t, err)
	assert.Equal(t, res2.Reviews, countOne(t, pool,
		`SELECT COUNT(*) FROM review_log WHERE user_id = $1`, userID))
}

func TestRunSeed_NoDeckSpecies_Errors(t *testing.T) {
	pool := connectSeedTestPool(t)
	ctx := context.Background()
	var userID int64
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO users (google_id, email, name, picture)
		 VALUES ('_seed_gid2', '_seed2@example.com', 'Seed Empty', '')
		 RETURNING id`).Scan(&userID))
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	})

	_, err := runSeed(ctx, pool, userID, 3, rand.New(rand.NewSource(1)), time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "clone a preset deck")
}

// seedManySpeciesFixtures inserts a user with one deck holding n same-family
// species, each with a recording and an image, and registers cleanup.
func seedManySpeciesFixtures(t *testing.T, pool *pgxpool.Pool, n int) (userID int64) {
	t.Helper()
	ctx := context.Background()

	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO users (google_id, email, name, picture)
		 VALUES ('_seed_gid3', '_seed3@example.com', 'Seed Caps', '')
		 RETURNING id`).Scan(&userID))

	var deckID int64
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO decks (name, description, owner_id)
		 VALUES ('_seed caps deck', '', $1) RETURNING id`, userID).Scan(&deckID))

	for i := range n {
		code := fmt.Sprintf("_sc%d", i)
		_, err := pool.Exec(ctx,
			`INSERT INTO species (ebird_code, common_name, scientific_name, family)
			 VALUES ($1, $2, $3, 'Seed Caps') ON CONFLICT DO NOTHING`,
			code, fmt.Sprintf("Cap Bird %d", i), fmt.Sprintf("Capus birdus%d", i))
		require.NoError(t, err)
		_, err = pool.Exec(ctx,
			`INSERT INTO species_recordings (xeno_canto_id, species_code, file_path, quality, type, credit)
			 VALUES ($1, $2, $3, 'A', 'song', 'test') ON CONFLICT DO NOTHING`,
			fmt.Sprintf("_scxc%d", i), code, fmt.Sprintf("placeholder://sc%d.mp3", i))
		require.NoError(t, err)
		_, err = pool.Exec(ctx,
			`INSERT INTO species_images (macaulay_id, species_code, file_path, credit)
			 VALUES ($1, $2, $3, 'test') ON CONFLICT DO NOTHING`,
			fmt.Sprintf("_scml%d", i), code, fmt.Sprintf("placeholder://sc%d.jpg", i))
		require.NoError(t, err)
		_, err = pool.Exec(ctx,
			`INSERT INTO deck_species (deck_id, species_code) VALUES ($1, $2)`, deckID, code)
		require.NoError(t, err)
	}

	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM review_log WHERE user_id = $1`, userID)
		_, _ = pool.Exec(ctx, `DELETE FROM cards WHERE user_id = $1`, userID)
		_, _ = pool.Exec(ctx, `DELETE FROM decks WHERE id = $1`, deckID)
		_, _ = pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
		_, _ = pool.Exec(ctx, `DELETE FROM species_recordings WHERE species_code LIKE '\_sc%'`)
		_, _ = pool.Exec(ctx, `DELETE FROM species_images WHERE species_code LIKE '\_sc%'`)
		_, _ = pool.Exec(ctx, `DELETE FROM species WHERE ebird_code LIKE '\_sc%'`)
	})

	return userID
}

func TestRunSeed_CapsNewCardsPerDay(t *testing.T) {
	pool := connectSeedTestPool(t)
	ctx := context.Background()
	userID := seedManySpeciesFixtures(t, pool, 10) // 10 species => 20 cards

	// One un-skipped day (seed 7's first draw doesn't skip); only
	// newCardsPerDay cards may be introduced in a single session.
	res, err := runSeed(ctx, pool, userID, 1, rand.New(rand.NewSource(7)), time.Now())
	require.NoError(t, err)
	require.Equal(t, int64(20), res.Seeded)
	require.Equal(t, 0, res.Skipped, "seed 7 must not skip the only day; pick another seed if rng use changes")

	assert.Equal(t, newCardsPerDay, res.Reviews,
		"a single session must introduce at most newCardsPerDay cards")
	assert.Equal(t, 20-newCardsPerDay, countOne(t, pool,
		`SELECT COUNT(*) FROM cards WHERE user_id = $1 AND reps = 0`, userID),
		"unintroduced cards stay not-seen")
}
