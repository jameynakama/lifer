package store_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jameynakama/flockdeck/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func connectTestDB(t *testing.T) *pgxpool.Pool {
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

	// _tst1 has both a quiz-quality recording and an image.
	_, err = tx.Exec(ctx,
		`INSERT INTO species_recordings (xeno_canto_id, species_code, file_path, quality, type, credit)
		 VALUES ('_xc_tst1', '_tst1', 'https://r2.example.com/rec.mp3', 'A', 'song', 'tester')`,
	)
	require.NoError(t, err)
	_, err = tx.Exec(ctx,
		`INSERT INTO species_images (macaulay_id, species_code, file_path, credit)
		 VALUES ('_ml_tst1', '_tst1', 'https://r2.example.com/img.jpg', 'tester')`,
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

// seedNoMediaSpecies adds a second species to the deck with due cards in both
// lanes but no recordings or images (e.g. a locked-media species whose
// automated sources had nothing).
func seedNoMediaSpecies(t *testing.T, tx pgx.Tx, f fixtures) {
	t.Helper()
	ctx := context.Background()

	_, err := tx.Exec(ctx,
		`INSERT INTO species (ebird_code, common_name, scientific_name)
		 VALUES ('_tst2', 'Bare Species', 'Testus nudus')`,
	)
	require.NoError(t, err)
	_, err = tx.Exec(ctx,
		`INSERT INTO deck_species (deck_id, species_code) VALUES ($1, '_tst2')`, f.deckID,
	)
	require.NoError(t, err)
	_, err = tx.Exec(ctx,
		`INSERT INTO cards (user_id, species_code, lane)
		 VALUES ($1, '_tst2', 'audio'), ($1, '_tst2', 'image')`,
		f.userID,
	)
	require.NoError(t, err)
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

func TestGetNextDueCard_NoMediaForLane_SkipsCard(t *testing.T) {
	pool := connectTestDB(t)
	tx := withTx(t, pool)
	f := seedFixtures(t, tx)
	seedNoMediaSpecies(t, tx, f)
	q := store.New(tx)

	// Make _tst1's cards not due so only the no-media species would remain.
	_, err := tx.Exec(context.Background(),
		`UPDATE cards SET due = NOW() + INTERVAL '1 day' WHERE species_code = '_tst1'`)
	require.NoError(t, err)

	_, err = q.GetNextDueCard(context.Background(), store.GetNextDueCardParams{
		UserID: f.userID,
		DeckID: f.deckID,
		Lane:   "audio",
	})
	assert.True(t, errors.Is(err, pgx.ErrNoRows),
		"want ErrNoRows when the only due species has no audio media, got %v", err)
}

func TestGetNextDueCard_NonQuizQualityRecording_SkipsCard(t *testing.T) {
	pool := connectTestDB(t)
	tx := withTx(t, pool)
	f := seedFixtures(t, tx)
	seedNoMediaSpecies(t, tx, f)
	q := store.New(tx)

	// _tst2 gets only a C-quality recording -- below the A/B quiz bar.
	_, err := tx.Exec(context.Background(),
		`INSERT INTO species_recordings (xeno_canto_id, species_code, file_path, quality, type, credit)
		 VALUES ('_xc_tst2', '_tst2', 'https://r2.example.com/rec2.mp3', 'C', 'call', 'tester')`)
	require.NoError(t, err)
	_, err = tx.Exec(context.Background(),
		`UPDATE cards SET due = NOW() + INTERVAL '1 day' WHERE species_code = '_tst1'`)
	require.NoError(t, err)

	_, err = q.GetNextDueCard(context.Background(), store.GetNextDueCardParams{
		UserID: f.userID,
		DeckID: f.deckID,
		Lane:   "audio",
	})
	assert.True(t, errors.Is(err, pgx.ErrNoRows),
		"want ErrNoRows when the only due species has no A/B recording, got %v", err)
}

// due_before session snapshot

func TestGetNextDueCard_DueBefore_ExcludesCardsDueAfterSnapshot(t *testing.T) {
	pool := connectTestDB(t)
	tx := withTx(t, pool)
	f := seedFixtures(t, tx)
	q := store.New(tx)

	// Snapshot taken an hour ago: the card (due NOW()) re-dued after the
	// session started, so it must not be served.
	snapshot := pgtype.Timestamptz{}
	require.NoError(t, snapshot.Scan(time.Now().Add(-time.Hour)))

	_, err := q.GetNextDueCard(context.Background(), store.GetNextDueCardParams{
		UserID:    f.userID,
		DeckID:    f.deckID,
		Lane:      "image",
		DueBefore: snapshot,
	})
	assert.True(t, errors.Is(err, pgx.ErrNoRows),
		"want ErrNoRows for cards that became due after the session snapshot, got %v", err)
}

func TestGetNextDueCard_DueBefore_IncludesCardsDueBeforeSnapshot(t *testing.T) {
	pool := connectTestDB(t)
	tx := withTx(t, pool)
	f := seedFixtures(t, tx)
	q := store.New(tx)

	snapshot := pgtype.Timestamptz{}
	require.NoError(t, snapshot.Scan(time.Now().Add(time.Minute)))

	card, err := q.GetNextDueCard(context.Background(), store.GetNextDueCardParams{
		UserID:    f.userID,
		DeckID:    f.deckID,
		Lane:      "image",
		DueBefore: snapshot,
	})
	require.NoError(t, err)
	assert.Equal(t, "_tst1", card.SpeciesCode)
}

func TestGetNextDueCard_NoDueBefore_FallsBackToNow(t *testing.T) {
	pool := connectTestDB(t)
	tx := withTx(t, pool)
	f := seedFixtures(t, tx)
	q := store.New(tx)

	card, err := q.GetNextDueCard(context.Background(), store.GetNextDueCardParams{
		UserID: f.userID,
		DeckID: f.deckID,
		Lane:   "image",
		// DueBefore zero-value: NULL -> COALESCE(NOW())
	})
	require.NoError(t, err)
	assert.Equal(t, "_tst1", card.SpeciesCode)
}

// due_remaining (window count folded into GetNextDueCard)

func TestGetNextDueCard_ReportsDueRemaining(t *testing.T) {
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
	assert.Equal(t, int64(1), card.DueRemaining)
}

func TestGetNextDueCard_DueRemaining_ExcludesNoMediaSpecies(t *testing.T) {
	pool := connectTestDB(t)
	tx := withTx(t, pool)
	f := seedFixtures(t, tx)
	seedNoMediaSpecies(t, tx, f)
	q := store.New(tx)

	card, err := q.GetNextDueCard(context.Background(), store.GetNextDueCardParams{
		UserID: f.userID,
		DeckID: f.deckID,
		Lane:   "image",
	})
	require.NoError(t, err)
	// Only _tst1 (which has an image) counts; _tst2 has no media.
	assert.Equal(t, int64(1), card.DueRemaining)
}

// GetRandomMediaForSpecies

func TestGetRandomMediaForSpecies_ReturnsBothLanes(t *testing.T) {
	pool := connectTestDB(t)
	tx := withTx(t, pool)
	seedFixtures(t, tx)
	q := store.New(tx)

	media, err := q.GetRandomMediaForSpecies(context.Background(), "_tst1")
	require.NoError(t, err)
	assert.Equal(t, "https://r2.example.com/rec.mp3", media.AudioPath)
	assert.Equal(t, "song", media.AudioType)
	assert.Equal(t, "tester", media.AudioCredit)
	assert.Equal(t, "https://r2.example.com/img.jpg", media.ImagePath)
	assert.Equal(t, "tester", media.ImageCredit)
}

func TestGetRandomMediaForSpecies_NoMedia_ReturnsEmptyFields(t *testing.T) {
	pool := connectTestDB(t)
	tx := withTx(t, pool)
	f := seedFixtures(t, tx)
	seedNoMediaSpecies(t, tx, f)
	q := store.New(tx)

	media, err := q.GetRandomMediaForSpecies(context.Background(), "_tst2")
	require.NoError(t, err)
	assert.Empty(t, media.AudioPath)
	assert.Empty(t, media.ImagePath)
}

func TestGetRandomMediaForSpecies_LowQualityRecording_Excluded(t *testing.T) {
	pool := connectTestDB(t)
	tx := withTx(t, pool)
	f := seedFixtures(t, tx)
	seedNoMediaSpecies(t, tx, f)
	q := store.New(tx)

	_, err := tx.Exec(context.Background(),
		`INSERT INTO species_recordings (xeno_canto_id, species_code, file_path, quality, type, credit)
		 VALUES ('_xc_tst2c', '_tst2', 'https://r2.example.com/recC.mp3', 'C', 'call', 'tester')`)
	require.NoError(t, err)

	media, err := q.GetRandomMediaForSpecies(context.Background(), "_tst2")
	require.NoError(t, err)
	assert.Empty(t, media.AudioPath, "C-quality recordings are below the quiz bar")
}

// Locked media

func lockMedia(t *testing.T, tx pgx.Tx) {
	t.Helper()
	_, err := tx.Exec(context.Background(),
		`UPDATE species_recordings SET locked = true WHERE xeno_canto_id = '_xc_tst1'`)
	require.NoError(t, err)
	_, err = tx.Exec(context.Background(),
		`UPDATE species_images SET locked = true WHERE macaulay_id = '_ml_tst1'`)
	require.NoError(t, err)
}

// Decided semantics: "locked" protects media from REMOVAL only. Ingest may
// still refresh same-source fields on a locked row (and add new media to a
// locked bird); only the delete paths honor the lock.
func TestUpsertRecording_LockedRow_StillUpdates(t *testing.T) {
	pool := connectTestDB(t)
	tx := withTx(t, pool)
	seedFixtures(t, tx)
	lockMedia(t, tx)
	q := store.New(tx)

	rec, err := q.UpsertRecording(context.Background(), store.UpsertRecordingParams{
		XenoCantoID: "_xc_tst1",
		SpeciesCode: "_tst1",
		FilePath:    "https://r2.example.com/refreshed.mp3",
		Quality:     "B",
		Type:        "call",
		Credit:      "tester",
	})
	require.NoError(t, err)
	assert.Equal(t, "https://r2.example.com/refreshed.mp3", rec.FilePath)
	assert.True(t, rec.Locked, "lock flag must survive the update")
}

func TestUpsertSpeciesImage_LockedRow_StillUpdates(t *testing.T) {
	pool := connectTestDB(t)
	tx := withTx(t, pool)
	seedFixtures(t, tx)
	lockMedia(t, tx)
	q := store.New(tx)

	img, err := q.UpsertSpeciesImage(context.Background(), store.UpsertSpeciesImageParams{
		MacaulayID:  "_ml_tst1",
		SpeciesCode: "_tst1",
		FilePath:    "https://r2.example.com/refreshed.jpg",
		Credit:      "tester",
	})
	require.NoError(t, err)
	assert.Equal(t, "https://r2.example.com/refreshed.jpg", img.FilePath)
	assert.True(t, img.Locked, "lock flag must survive the update")
}

func TestDeleteRecordingsBySpeciesCode_LockedRow_Survives(t *testing.T) {
	pool := connectTestDB(t)
	tx := withTx(t, pool)
	seedFixtures(t, tx)
	lockMedia(t, tx)
	q := store.New(tx)

	require.NoError(t, q.DeleteRecordingsBySpeciesCode(context.Background(), "_tst1"))

	var n int
	require.NoError(t, tx.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM species_recordings WHERE xeno_canto_id = '_xc_tst1'`).Scan(&n))
	assert.Equal(t, 1, n, "locked recording must survive delete-by-species")
}

// CreateReviewLog

func TestCreateReviewLog_RoundTrip(t *testing.T) {
	pool := connectTestDB(t)
	tx := withTx(t, pool)
	f := seedFixtures(t, tx)
	q := store.New(tx)

	row, err := q.CreateReviewLog(context.Background(), store.CreateReviewLogParams{
		UserID:             f.userID,
		SpeciesCode:        "_tst1",
		Lane:               "audio",
		Rating:             3,
		GuessedSpeciesCode: pgtype.Text{String: "_tst1", Valid: true},
		MediaID:            pgtype.Text{String: "_xc_tst1", Valid: true},
	})
	require.NoError(t, err, "Should insert review_log row without error")
	assert.Equal(t, f.userID, row.UserID, "Should round-trip user_id")
	assert.Equal(t, "_tst1", row.SpeciesCode, "Should round-trip species_code")
	assert.Equal(t, "audio", row.Lane, "Should round-trip lane")
	assert.Equal(t, int16(3), row.Rating, "Should round-trip rating")
	assert.Equal(t, "_tst1", row.GuessedSpeciesCode.String, "Should round-trip guessed_species_code")
	assert.True(t, row.GuessedSpeciesCode.Valid, "Should round-trip guessed_species_code validity")
	assert.Equal(t, "_xc_tst1", row.MediaID.String, "Should round-trip media_id")
	assert.True(t, row.MediaID.Valid, "Should round-trip media_id validity")
	assert.Greater(t, row.ID, int64(0), "Should assign a positive id")
	assert.True(t, row.ReviewedAt.Valid, "Should set reviewed_at")
}

func TestCreateReviewLog_NullGuessIsSkip(t *testing.T) {
	pool := connectTestDB(t)
	tx := withTx(t, pool)
	f := seedFixtures(t, tx)
	q := store.New(tx)

	row, err := q.CreateReviewLog(context.Background(), store.CreateReviewLogParams{
		UserID:      f.userID,
		SpeciesCode: "_tst1",
		Lane:        "image",
		Rating:      1,
		// GuessedSpeciesCode and MediaID left as zero-values (not valid)
	})
	require.NoError(t, err, "Should insert skip row without error")
	assert.False(t, row.GuessedSpeciesCode.Valid, "Should store NULL when no guess provided")
	assert.False(t, row.MediaID.Valid, "Should store NULL when no media_id provided")
}

func TestListSpeciesCodesWithLockedMedia_ReturnsCodesFromBothTables(t *testing.T) {
	pool := connectTestDB(t)
	tx := withTx(t, pool)
	f := seedFixtures(t, tx)
	seedNoMediaSpecies(t, tx, f) // _tst2: no locked media
	q := store.New(tx)

	// Nothing locked yet.
	codes, err := q.ListSpeciesCodesWithLockedMedia(context.Background())
	require.NoError(t, err)
	assert.NotContains(t, codes, "_tst1")

	// Lock _tst1's recording; _tst2 stays unlocked.
	_, err = tx.Exec(context.Background(),
		`UPDATE species_recordings SET locked = true WHERE xeno_canto_id = '_xc_tst1'`)
	require.NoError(t, err)

	codes, err = q.ListSpeciesCodesWithLockedMedia(context.Background())
	require.NoError(t, err)
	assert.Contains(t, codes, "_tst1")
	assert.NotContains(t, codes, "_tst2")
}
