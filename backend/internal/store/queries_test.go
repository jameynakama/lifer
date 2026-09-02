package store_test

import (
	"context"
	"errors"
	"fmt"
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

// seedDecklessSpecies adds a species with media and both-lane cards but NO
// deck_species row -- an orphan left behind after the species was removed from
// every deck. GetNextDueCard joins deck_species, so the quiz can never serve
// it; stats must not count it either.
func seedDecklessSpecies(t *testing.T, tx pgx.Tx, f fixtures, code string) {
	t.Helper()
	ctx := context.Background()
	_, err := tx.Exec(ctx,
		`INSERT INTO species (ebird_code, common_name, scientific_name)
		 VALUES ($1, 'Orphan Species', 'Testus orphanus')`, code)
	require.NoError(t, err)
	_, err = tx.Exec(ctx,
		`INSERT INTO species_recordings (xeno_canto_id, species_code, file_path, quality, type, credit)
		 VALUES ($1, $2, 'https://r2.example.com/orphan.mp3', 'A', 'song', 'tester')`,
		"_xc_"+code, code)
	require.NoError(t, err)
	_, err = tx.Exec(ctx,
		`INSERT INTO species_images (macaulay_id, species_code, file_path, credit)
		 VALUES ($1, $2, 'https://r2.example.com/orphan.jpg', 'tester')`,
		"_ml_"+code, code)
	require.NoError(t, err)
	_, err = tx.Exec(ctx,
		`INSERT INTO cards (user_id, species_code, lane)
		 VALUES ($1, $2, 'audio'), ($1, $2, 'image')`, f.userID, code)
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

// seedDeckSpeciesWithCards adds a species to the fixture deck with both-lane cards (no media needed; GetCardStateCounts has no media filter).
func seedDeckSpeciesWithCards(t *testing.T, tx pgx.Tx, f fixtures, code string) {
	t.Helper()
	ctx := context.Background()
	_, err := tx.Exec(ctx,
		`INSERT INTO species (ebird_code, common_name, scientific_name)
		 VALUES ($1, 'Tier Species', 'Testus gradus')`, code)
	require.NoError(t, err)
	_, err = tx.Exec(ctx, `INSERT INTO deck_species (deck_id, species_code) VALUES ($1, $2)`, f.deckID, code)
	require.NoError(t, err)
	_, err = tx.Exec(ctx,
		`INSERT INTO cards (user_id, species_code, lane)
		 VALUES ($1, $2, 'audio'), ($1, $2, 'image')`, f.userID, code)
	require.NoError(t, err)
}

// disableImageForSpecies turns the image lane off for one species.
func disableImageForSpecies(t *testing.T, tx pgx.Tx, userID int64, code string) {
	t.Helper()
	_, err := tx.Exec(context.Background(),
		`INSERT INTO user_species_preferences (user_id, species_code, audio_enabled, image_enabled)
		 VALUES ($1, $2, true, false)`, userID, code)
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
	assert.Equal(t, "_xc_tst1", media.AudioID, "Should return the recording's xeno-canto ID")
	assert.Equal(t, "https://r2.example.com/img.jpg", media.ImagePath)
	assert.Equal(t, "tester", media.ImageCredit)
	assert.Equal(t, "_ml_tst1", media.ImageID, "Should return the image's macaulay ID")
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
	assert.Empty(t, media.AudioID, "Should return empty audio ID when no media")
	assert.Empty(t, media.ImageID, "Should return empty image ID when no media")
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

func TestListSpeciesCodes_ReturnsSeededSpecies(t *testing.T) {
	pool := connectTestDB(t)
	tx := withTx(t, pool)
	seedFixtures(t, tx)
	q := store.New(tx)

	codes, err := q.ListSpeciesCodes(context.Background())
	require.NoError(t, err)
	assert.Contains(t, codes, "_tst1")
}

// mustSchedule drives a card to a given FSRS state by calling UpdateCardSchedule.
// stability is set to 10 (a reasonable "known" value); due is set one day ahead.
func mustSchedule(t *testing.T, q *store.Queries, userID int64, speciesCode, lane string, state int16) {
	t.Helper()
	due := pgtype.Timestamptz{}
	require.NoError(t, due.Scan(time.Now().Add(24*time.Hour)))
	_, err := q.UpdateCardSchedule(context.Background(), store.UpdateCardScheduleParams{
		UserID:      userID,
		SpeciesCode: speciesCode,
		Lane:        lane,
		Stability:   10,
		Difficulty:  5,
		Due:         due,
		Lapses:      0,
		State:       state,
	})
	require.NoError(t, err)
}

// mustScheduleStability drives a card to a specific stability (and reps>0, valid
// last_review) so tier-boundary tests can place a card in any stage.
func mustScheduleStability(t *testing.T, q *store.Queries, userID int64, speciesCode, lane string, stability float64) {
	t.Helper()
	due := pgtype.Timestamptz{}
	require.NoError(t, due.Scan(time.Now().Add(24*time.Hour)))
	_, err := q.UpdateCardSchedule(context.Background(), store.UpdateCardScheduleParams{
		UserID:      userID,
		SpeciesCode: speciesCode,
		Lane:        lane,
		Stability:   stability,
		Difficulty:  5,
		Due:         due,
		Lapses:      0,
		State:       2,
	})
	require.NoError(t, err)
}

// laneFilter returns a pgtype.Text suitable for use as a lane filter param.
func laneFilter(lane string) pgtype.Text {
	return pgtype.Text{String: lane, Valid: true}
}

// GetCardStateCounts

func TestGetCardStateCounts_Buckets(t *testing.T) {
	pool := connectTestDB(t)
	tx := withTx(t, pool)
	f := seedFixtures(t, tx)
	q := store.New(tx)
	// seedFixtures creates audio+image cards with reps=0 (egg).
	// Drive audio to stability 10 (juvenile tier [7,30)).
	mustScheduleStability(t, q, f.userID, "_tst1", "audio", 10)

	rows, err := q.GetCardStateCounts(context.Background(), store.GetCardStateCountsParams{
		UserID: f.userID,
	})
	require.NoError(t, err)

	buckets := make(map[string]int64)
	for _, r := range rows {
		buckets[r.Bucket] = r.Count
	}
	assert.Equal(t, int64(1), buckets["juvenile"], "Should count audio card in juvenile bucket (stability=10)")
	assert.GreaterOrEqual(t, buckets["egg"], int64(1), "Should count image card in egg bucket")
}

func TestGetCardStateCounts_LaneFilter(t *testing.T) {
	pool := connectTestDB(t)
	tx := withTx(t, pool)
	f := seedFixtures(t, tx)
	q := store.New(tx)
	// Drive audio to stability 10 (juvenile); image stays reps=0 (egg).
	mustSchedule(t, q, f.userID, "_tst1", "audio", 2)

	rows, err := q.GetCardStateCounts(context.Background(), store.GetCardStateCountsParams{
		UserID: f.userID,
		Lane:   laneFilter("image"),
	})
	require.NoError(t, err)

	buckets := make(map[string]int64)
	for _, r := range rows {
		buckets[r.Bucket] = r.Count
	}
	assert.NotContains(t, buckets, "juvenile", "Should not see the audio card through the image filter")
	assert.GreaterOrEqual(t, buckets["egg"], int64(1), "image lane card is egg")
}

func TestGetCardStateCounts_ExcludesDisabledLane(t *testing.T) {
	pool := connectTestDB(t)
	tx := withTx(t, pool)
	f := seedFixtures(t, tx)
	q := store.New(tx)
	// The user's bug: audio is juvenile (stability=10), image is toggled off. The quiz
	// never serves the disabled image card, so its reps stay 0 forever -- it must not
	// haunt the progress bar as a permanent egg.
	mustSchedule(t, q, f.userID, "_tst1", "audio", 2)
	disableImageLane(t, tx, f.userID)

	rows, err := q.GetCardStateCounts(context.Background(), store.GetCardStateCountsParams{
		UserID: f.userID,
	})
	require.NoError(t, err)

	buckets := make(map[string]int64)
	for _, r := range rows {
		buckets[r.Bucket] = r.Count
	}
	assert.Equal(t, int64(1), buckets["juvenile"], "Should still count the enabled audio card")
	assert.Equal(t, int64(0), buckets["egg"], "disabled image card excluded")
}

func TestGetCardStateCounts_ExcludesDecklessSpecies(t *testing.T) {
	pool := connectTestDB(t)
	tx := withTx(t, pool)
	f := seedFixtures(t, tx)
	q := store.New(tx)
	// _tst1 (in a deck) both lanes at stability 10 (juvenile); an orphaned species
	// with reps=0 cards must not be counted -- removing a species from a deck leaves
	// its cards behind, and the quiz can never serve a species that's in no deck.
	mustSchedule(t, q, f.userID, "_tst1", "audio", 2)
	mustSchedule(t, q, f.userID, "_tst1", "image", 2)
	seedDecklessSpecies(t, tx, f, "_tst3")

	rows, err := q.GetCardStateCounts(context.Background(), store.GetCardStateCountsParams{
		UserID: f.userID,
	})
	require.NoError(t, err)
	buckets := make(map[string]int64)
	for _, r := range rows {
		buckets[r.Bucket] = r.Count
	}
	assert.Equal(t, int64(2), buckets["juvenile"], "both in-deck lanes are juvenile")
	assert.Equal(t, int64(0), buckets["egg"], "deckless egg orphan excluded")
}

func TestGetCardStateCounts_TierBuckets(t *testing.T) {
	pool := connectTestDB(t)
	tx := withTx(t, pool)
	f := seedFixtures(t, tx)
	q := store.New(tx)
	// _tst1 image stays reps=0 -> egg. Drive _tst1 audio to a juvenile stability.
	mustScheduleStability(t, q, f.userID, "_tst1", "audio", 10) // 7<=10<30 -> juvenile

	rows, err := q.GetCardStateCounts(context.Background(), store.GetCardStateCountsParams{
		UserID: f.userID,
	})
	require.NoError(t, err)
	buckets := make(map[string]int64)
	for _, r := range rows {
		buckets[r.Bucket] = r.Count
	}
	assert.Equal(t, int64(1), buckets["egg"], "image card never quizzed -> egg")
	assert.Equal(t, int64(1), buckets["juvenile"], "audio at stability 10 -> juvenile")
	assert.Equal(t, int64(0), buckets["known"], "old 'known' bucket must no longer exist")
}

func TestGetCardStateCounts_AllTierBoundaries(t *testing.T) {
	pool := connectTestDB(t)
	tx := withTx(t, pool)
	f := seedFixtures(t, tx)
	q := store.New(tx)
	// Five extra species, one per non-egg tier, each added to the deck (so the
	// deck-membership filter keeps them) with both lanes; drive audio to a
	// representative stability and disable image so only the audio card counts.
	cases := []struct {
		code      string
		stability float64
		want      string
	}{
		{"_tn", 0.5, "nestling"},
		{"_tf", 1, "fledgling"},   // exactly 1.0 must be fledgling, not nestling
		{"_tj", 7, "juvenile"},    // exactly 7.0 must be juvenile, not fledgling
		{"_ti", 30, "immature"},   // exactly 30.0 must be immature, not juvenile
		{"_ta", 90, "adult"},      // exactly 90.0 must be adult, not immature
	}
	for _, c := range cases {
		seedDeckSpeciesWithCards(t, tx, f, c.code)
		mustScheduleStability(t, q, f.userID, c.code, "audio", c.stability)
		disableImageForSpecies(t, tx, f.userID, c.code)
	}
	rows, err := q.GetCardStateCounts(context.Background(), store.GetCardStateCountsParams{UserID: f.userID})
	require.NoError(t, err)
	buckets := map[string]int64{}
	for _, r := range rows {
		buckets[r.Bucket] = r.Count
	}
	for _, c := range cases {
		assert.Equal(t, int64(1), buckets[c.want], "stability %v should land in %s", c.stability, c.want)
	}
}

// GetCardTotals

func TestGetCardTotals(t *testing.T) {
	pool := connectTestDB(t)
	tx := withTx(t, pool)
	f := seedFixtures(t, tx)
	q := store.New(tx)
	// One review on audio card (reps increments to 1 in UpdateCardSchedule SQL).
	mustSchedule(t, q, f.userID, "_tst1", "audio", 2)

	totals, err := q.GetCardTotals(context.Background(), store.GetCardTotalsParams{
		UserID: f.userID,
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, totals.Species, int64(1), "Should count at least one species")
	assert.GreaterOrEqual(t, totals.Cards, int64(1), "Should count at least one card")
	assert.GreaterOrEqual(t, totals.Reviews, int64(1), "Should count at least one review")
}

// GetBankedCards

func TestGetBankedCards_OnlyBanked(t *testing.T) {
	pool := connectTestDB(t)
	tx := withTx(t, pool)
	f := seedFixtures(t, tx)
	q := store.New(tx)
	// audio at stability 10 (>=7 -> banked); image stays reps=0 (not banked).
	mustScheduleStability(t, q, f.userID, "_tst1", "audio", 10)

	banked, err := q.GetBankedCards(context.Background(), store.GetBankedCardsParams{UserID: f.userID})
	require.NoError(t, err)
	require.Len(t, banked, 1, "only the banked audio card")
	assert.Equal(t, "_tst1", banked[0].SpeciesCode)
	assert.Equal(t, "audio", banked[0].Lane)
}

func TestGetBankedCards_BelowBarExcluded(t *testing.T) {
	pool := connectTestDB(t)
	tx := withTx(t, pool)
	f := seedFixtures(t, tx)
	q := store.New(tx)
	// stability 3 is reviewed but below the 7-day banked bar.
	mustScheduleStability(t, q, f.userID, "_tst1", "audio", 3)

	banked, err := q.GetBankedCards(context.Background(), store.GetBankedCardsParams{UserID: f.userID})
	require.NoError(t, err)
	assert.Empty(t, banked, "a stability-3 card is not banked")
}

func TestGetBankedCards_EggExcluded(t *testing.T) {
	pool := connectTestDB(t)
	tx := withTx(t, pool)
	f := seedFixtures(t, tx)
	q := store.New(tx)
	// No scheduling — both cards remain reps=0.

	banked, err := q.GetBankedCards(context.Background(), store.GetBankedCardsParams{
		UserID: f.userID,
	})
	require.NoError(t, err)
	assert.Empty(t, banked, "Should return no cards when none have been banked")
}

func TestGetBankedCards_ExcludesDecklessSpecies(t *testing.T) {
	pool := connectTestDB(t)
	tx := withTx(t, pool)
	f := seedFixtures(t, tx)
	q := store.New(tx)
	// An orphaned (deckless) species driven to banked must not feed Fading or the
	// Remember projection -- it can never be studied, so it'd never actually fade.
	seedDecklessSpecies(t, tx, f, "_tst3")
	mustScheduleStability(t, q, f.userID, "_tst3", "audio", 10)

	banked, err := q.GetBankedCards(context.Background(), store.GetBankedCardsParams{
		UserID: f.userID,
	})
	require.NoError(t, err)
	assert.Empty(t, banked, "Should not include a banked card for a species in no deck")
}

func TestGetReviewedCards_IncludesAllReps(t *testing.T) {
	pool := connectTestDB(t)
	tx := withTx(t, pool)
	f := seedFixtures(t, tx)
	q := store.New(tx)
	mustScheduleStability(t, q, f.userID, "_tst1", "audio", 3) // reviewed, low stability
	// image stays reps=0 -> excluded.

	reviewed, err := q.GetReviewedCards(context.Background(), store.GetReviewedCardsParams{UserID: f.userID})
	require.NoError(t, err)
	require.Len(t, reviewed, 1, "only the reviewed (reps>0) card, regardless of stability")
	assert.Equal(t, "_tst1", reviewed[0].SpeciesCode)
}

// GetLaneGaps

func TestGetLaneGaps_OneLaneKnown(t *testing.T) {
	pool := connectTestDB(t)
	tx := withTx(t, pool)
	f := seedFixtures(t, tx)
	q := store.New(tx)
	// Audio banked (stability=10 >= 7), image stays reps=0 → should produce one gap row.
	mustScheduleStability(t, q, f.userID, "_tst1", "audio", 10)

	gaps, err := q.GetLaneGaps(context.Background(), f.userID)
	require.NoError(t, err)

	require.Len(t, gaps, 1, "Should return one gap row when only audio is banked")
	assert.Equal(t, "_tst1", gaps[0].SpeciesCode)
	assert.Equal(t, "audio", gaps[0].KnownLane, "Should identify audio as banked lane")
	assert.Equal(t, "image", gaps[0].WeakLane, "Should identify image as weak lane")
}

func TestGetLaneGaps_BothLanesKnown_NoGap(t *testing.T) {
	pool := connectTestDB(t)
	tx := withTx(t, pool)
	f := seedFixtures(t, tx)
	q := store.New(tx)
	// Both lanes banked (stability=10 >= 7) → should NOT appear in gaps.
	mustScheduleStability(t, q, f.userID, "_tst1", "audio", 10)
	mustScheduleStability(t, q, f.userID, "_tst1", "image", 10)

	gaps, err := q.GetLaneGaps(context.Background(), f.userID)
	require.NoError(t, err)
	assert.Empty(t, gaps, "Should return no gaps when both lanes are banked")
}

func TestGetLaneGaps_ImageKnown_AudioWeak(t *testing.T) {
	pool := connectTestDB(t)
	tx := withTx(t, pool)
	f := seedFixtures(t, tx)
	q := store.New(tx)
	// Image banked (stability=10 >= 7), audio stays reps=0 → one gap with the reversed lanes.
	mustScheduleStability(t, q, f.userID, "_tst1", "image", 10)

	gaps, err := q.GetLaneGaps(context.Background(), f.userID)
	require.NoError(t, err)

	require.Len(t, gaps, 1, "Should return one gap row when only image is banked")
	assert.Equal(t, "_tst1", gaps[0].SpeciesCode)
	assert.Equal(t, "image", gaps[0].KnownLane, "Should identify image as banked lane")
	assert.Equal(t, "audio", gaps[0].WeakLane, "Should identify audio as weak lane")
}

func TestGetLaneGaps_ExcludesDisabledLane(t *testing.T) {
	pool := connectTestDB(t)
	tx := withTx(t, pool)
	f := seedFixtures(t, tx)
	q := store.New(tx)
	// Audio banked (stability=10 >= 7), image weak -- but image is disabled, so
	// this is not an actionable gap: the user opted out of image practice for this species.
	mustScheduleStability(t, q, f.userID, "_tst1", "audio", 10)
	disableImageLane(t, tx, f.userID)

	gaps, err := q.GetLaneGaps(context.Background(), f.userID)
	require.NoError(t, err)
	assert.Empty(t, gaps, "Should not surface a gap for a lane the user disabled")
}

func TestGetLaneGaps_ExcludesDecklessSpecies(t *testing.T) {
	pool := connectTestDB(t)
	tx := withTx(t, pool)
	f := seedFixtures(t, tx)
	q := store.New(tx)
	// Orphan banked (stability=10 >= 7) in audio, weak in image -- but it's in
	// no deck, so it's not an actionable gap: the user can't practice a species
	// they can't reach.
	seedDecklessSpecies(t, tx, f, "_tst3")
	mustScheduleStability(t, q, f.userID, "_tst3", "audio", 10)

	gaps, err := q.GetLaneGaps(context.Background(), f.userID)
	require.NoError(t, err)
	assert.Empty(t, gaps, "Should not surface a gap for a species in no deck")
}

func TestGetLaneGaps_BelowBankedBarIsNotAGap(t *testing.T) {
	pool := connectTestDB(t)
	tx := withTx(t, pool)
	f := seedFixtures(t, tx)
	q := store.New(tx)
	// audio reviewed but stability 3 (< 7 banked bar); image reps=0. Neither
	// lane is banked, so there is no actionable gap.
	mustScheduleStability(t, q, f.userID, "_tst1", "audio", 3)

	gaps, err := q.GetLaneGaps(context.Background(), f.userID)
	require.NoError(t, err)
	assert.Empty(t, gaps, "a sub-banked lane is not a gap")
}

// GetCardTotals lane filter

func TestGetCardTotals_LaneFilter(t *testing.T) {
	pool := connectTestDB(t)
	tx := withTx(t, pool)
	f := seedFixtures(t, tx)
	q := store.New(tx)
	// One review on audio card only.
	mustSchedule(t, q, f.userID, "_tst1", "audio", 2)

	// Image-lane filter should report zero reviews.
	imageTotals, err := q.GetCardTotals(context.Background(), store.GetCardTotalsParams{
		UserID: f.userID,
		Lane:   laneFilter("image"),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(0), imageTotals.Reviews, "Should report zero reviews when filtering to image lane")

	// Combined (no filter) should show the audio review.
	allTotals, err := q.GetCardTotals(context.Background(), store.GetCardTotalsParams{
		UserID: f.userID,
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, allTotals.Reviews, int64(1), "Should count the audio review in combined totals")
}

func TestGetCardTotals_ExcludesDisabledLane(t *testing.T) {
	pool := connectTestDB(t)
	tx := withTx(t, pool)
	f := seedFixtures(t, tx)
	q := store.New(tx)
	mustSchedule(t, q, f.userID, "_tst1", "audio", 2)
	disableImageLane(t, tx, f.userID)

	totals, err := q.GetCardTotals(context.Background(), store.GetCardTotalsParams{
		UserID: f.userID,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), totals.Cards, "Should count only the enabled audio card, not the disabled image card")
	assert.Equal(t, int64(1), totals.Species, "Should still count the species via its enabled lane")
}

func TestGetCardTotals_ExcludesDecklessSpecies(t *testing.T) {
	pool := connectTestDB(t)
	tx := withTx(t, pool)
	f := seedFixtures(t, tx)
	q := store.New(tx)
	seedDecklessSpecies(t, tx, f, "_tst3")

	totals, err := q.GetCardTotals(context.Background(), store.GetCardTotalsParams{
		UserID: f.userID,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), totals.Species, "Should count only the in-deck species")
	assert.Equal(t, int64(2), totals.Cards, "Should count only the in-deck species' two lanes")
}

// logReview is a test helper that inserts a review_log row with optional
// guessed species and media ID.
func logReview(t *testing.T, q *store.Queries, ctx context.Context, userID int64, species, lane string, rating int16, guessed, mediaID string) {
	t.Helper()
	params := store.CreateReviewLogParams{UserID: userID, SpeciesCode: species, Lane: lane, Rating: rating}
	if guessed != "" {
		params.GuessedSpeciesCode = pgtype.Text{String: guessed, Valid: true}
	}
	if mediaID != "" {
		params.MediaID = pgtype.Text{String: mediaID, Valid: true}
	}
	_, err := q.CreateReviewLog(ctx, params)
	require.NoError(t, err)
}

// GetConfusionPairs

func TestGetConfusionPairs_GroupsAndOrders(t *testing.T) {
	pool := connectTestDB(t)
	tx := withTx(t, pool)
	f := seedFixtures(t, tx)
	seedNoMediaSpecies(t, tx, f)
	q := store.New(tx)
	ctx := context.Background()

	// 2 misses: actual=_tst1, guessed=_tst2 (count=2, the higher pair)
	logReview(t, q, ctx, f.userID, "_tst1", "audio", 1, "_tst2", "")
	logReview(t, q, ctx, f.userID, "_tst1", "audio", 1, "_tst2", "")
	// 1 skip: no guess -- should be excluded
	logReview(t, q, ctx, f.userID, "_tst1", "audio", 1, "", "")
	// 1 correct: guess == actual -- should be excluded
	logReview(t, q, ctx, f.userID, "_tst1", "audio", 3, "_tst1", "")
	// 1 miss in reverse direction: actual=_tst2, guessed=_tst1 (count=1, lower pair)
	logReview(t, q, ctx, f.userID, "_tst2", "audio", 1, "_tst1", "")

	pairs, err := q.GetConfusionPairs(ctx, store.GetConfusionPairsParams{UserID: f.userID})
	require.NoError(t, err)
	require.Len(t, pairs, 2, "Should return both confusion pairs")
	assert.Equal(t, "_tst1", pairs[0].SpeciesCode)
	assert.Equal(t, "_tst2", pairs[0].GuessedSpeciesCode)
	assert.Equal(t, int64(2), pairs[0].Count)
	assert.Greater(t, pairs[0].Count, pairs[1].Count, "Should order by count descending")
}

func TestGetConfusionPairs_LaneFilter(t *testing.T) {
	pool := connectTestDB(t)
	tx := withTx(t, pool)
	f := seedFixtures(t, tx)
	seedNoMediaSpecies(t, tx, f)
	q := store.New(tx)
	ctx := context.Background()

	// Miss in audio lane
	logReview(t, q, ctx, f.userID, "_tst1", "audio", 1, "_tst2", "")

	// Filter to image lane -- audio miss should not appear
	pairs, err := q.GetConfusionPairs(ctx, store.GetConfusionPairsParams{
		UserID: f.userID,
		Lane:   laneFilter("image"),
	})
	require.NoError(t, err)
	assert.Empty(t, pairs, "Should return no confusion pairs when filtering to image lane")
}

// GetHardMedia

func TestGetHardMedia_ThresholdAndOrdering(t *testing.T) {
	pool := connectTestDB(t)
	tx := withTx(t, pool)
	f := seedFixtures(t, tx)
	q := store.New(tx)
	ctx := context.Background()

	// 3 audio misses on _xc_tst1 -- threshold met, accuracy 0/3 = 0.0 (worst)
	logReview(t, q, ctx, f.userID, "_tst1", "audio", 1, "", "_xc_tst1")
	logReview(t, q, ctx, f.userID, "_tst1", "audio", 1, "", "_xc_tst1")
	logReview(t, q, ctx, f.userID, "_tst1", "audio", 1, "", "_xc_tst1")
	// 3 image misses + 1 image correct on _ml_tst1 -- threshold met, accuracy 1/4 = 0.25
	logReview(t, q, ctx, f.userID, "_tst1", "image", 1, "", "_ml_tst1")
	logReview(t, q, ctx, f.userID, "_tst1", "image", 1, "", "_ml_tst1")
	logReview(t, q, ctx, f.userID, "_tst1", "image", 1, "", "_ml_tst1")
	logReview(t, q, ctx, f.userID, "_tst1", "image", 3, "", "_ml_tst1")

	rows, err := q.GetHardMedia(ctx, store.GetHardMediaParams{UserID: f.userID})
	require.NoError(t, err)
	require.Len(t, rows, 2, "Should return both media items that meet the threshold")
	// Ordered by accuracy ASC: audio (0.0) before image (0.25)
	assert.Equal(t, "_xc_tst1", rows[0].MediaID, "Should place worst accuracy (audio) first")
	assert.Equal(t, int64(0), rows[0].Correct)
	assert.NotEmpty(t, rows[0].MediaUrl, "Should resolve media URL from seeded recording")
	assert.Equal(t, "https://r2.example.com/rec.mp3", rows[0].MediaUrl)
	assert.Equal(t, "_ml_tst1", rows[1].MediaID, "Should place better accuracy (image) second")
	assert.Equal(t, int64(1), rows[1].Correct)
	assert.Equal(t, "https://r2.example.com/img.jpg", rows[1].MediaUrl)
}

// GetFamilyAccuracy

func TestGetFamilyAccuracy_OmitsNullFamily(t *testing.T) {
	pool := connectTestDB(t)
	tx := withTx(t, pool)
	f := seedFixtures(t, tx)
	q := store.New(tx)
	ctx := context.Background()

	// _tst1 has NULL family -- log a correct review
	logReview(t, q, ctx, f.userID, "_tst1", "audio", 3, "_tst1", "")

	// With NULL family the row should be omitted
	rows, err := q.GetFamilyAccuracy(ctx, store.GetFamilyAccuracyParams{UserID: f.userID})
	require.NoError(t, err)
	assert.Empty(t, rows, "Should return no rows when species has NULL family")

	// Backfill the family
	_, err = q.UpsertSpecies(ctx, store.UpsertSpeciesParams{
		EbirdCode:      "_tst1",
		CommonName:     "Test Species",
		ScientificName: "Testus specius",
		Family:         pgtype.Text{String: "Test Family", Valid: true},
	})
	require.NoError(t, err)

	rows, err = q.GetFamilyAccuracy(ctx, store.GetFamilyAccuracyParams{UserID: f.userID})
	require.NoError(t, err)
	require.Len(t, rows, 1, "Should return one row after family is set")
	assert.Equal(t, "Test Family", rows[0].Family)
	assert.Equal(t, int64(1), rows[0].Attempts)
	assert.Equal(t, int64(1), rows[0].Correct)
}

// TestReviewLog_GuessSetNullOnSpeciesDelete

func TestReviewLog_GuessSetNullOnSpeciesDelete(t *testing.T) {
	pool := connectTestDB(t)
	tx := withTx(t, pool)
	f := seedFixtures(t, tx)
	seedNoMediaSpecies(t, tx, f)
	q := store.New(tx)
	ctx := context.Background()

	// Miss: actual=_tst1, guessed=_tst2
	logReview(t, q, ctx, f.userID, "_tst1", "audio", 1, "_tst2", "")

	// Delete the guessed species -- ON DELETE SET NULL should null out guessed_species_code
	require.NoError(t, q.DeleteSpeciesByCode(ctx, "_tst2"))

	// Confusion pairs should be empty: nulled guess is not a valid confusion
	pairs, err := q.GetConfusionPairs(ctx, store.GetConfusionPairsParams{UserID: f.userID})
	require.NoError(t, err)
	assert.Empty(t, pairs, "Should return no pairs when guessed species was deleted (null FK)")

	// The log row itself still exists
	acc, err := q.GetReviewAccuracy(ctx, store.GetReviewAccuracyParams{UserID: f.userID})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, acc.Attempts, int64(1), "Should still have the log row after species deletion")
}

// GetReviewAccuracy

func TestGetReviewAccuracy(t *testing.T) {
	pool := connectTestDB(t)
	tx := withTx(t, pool)
	f := seedFixtures(t, tx)
	q := store.New(tx)
	ctx := context.Background()

	// 2 correct, 1 wrong -- total 3 attempts
	logReview(t, q, ctx, f.userID, "_tst1", "audio", 3, "_tst1", "")
	logReview(t, q, ctx, f.userID, "_tst1", "image", 3, "_tst1", "")
	logReview(t, q, ctx, f.userID, "_tst1", "audio", 1, "", "")

	acc, err := q.GetReviewAccuracy(ctx, store.GetReviewAccuracyParams{UserID: f.userID})
	require.NoError(t, err)
	assert.Equal(t, int64(3), acc.Attempts)
	assert.Equal(t, int64(2), acc.Correct)

	// Lane filter: only audio (2 rows: 1 correct, 1 wrong)
	audioAcc, err := q.GetReviewAccuracy(ctx, store.GetReviewAccuracyParams{
		UserID: f.userID,
		Lane:   laneFilter("audio"),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), audioAcc.Attempts)
	assert.Equal(t, int64(1), audioAcc.Correct)
}

// CountReviewsSince

func TestCountReviewsSince(t *testing.T) {
	pool := connectTestDB(t)
	tx := withTx(t, pool)
	f := seedFixtures(t, tx)
	q := store.New(tx)
	ctx := context.Background()

	logReview(t, q, ctx, f.userID, "_tst1", "audio", 3, "_tst1", "")
	logReview(t, q, ctx, f.userID, "_tst1", "image", 1, "", "")

	past := pgtype.Timestamptz{}
	require.NoError(t, past.Scan(time.Now().Add(-time.Minute)))

	// Both reviews are after "past" -- should count 2
	n, err := q.CountReviewsSince(ctx, store.CountReviewsSinceParams{
		UserID:     f.userID,
		ReviewedAt: past,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), n)

	// Future cutoff -- no reviews after it yet
	future := pgtype.Timestamptz{}
	require.NoError(t, future.Scan(time.Now().Add(time.Hour)))

	n, err = q.CountReviewsSince(ctx, store.CountReviewsSinceParams{
		UserID:     f.userID,
		ReviewedAt: future,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(0), n)

	// Lane filter: only audio
	audioN, err := q.CountReviewsSince(ctx, store.CountReviewsSinceParams{
		UserID:     f.userID,
		ReviewedAt: past,
		Lane:       laneFilter("audio"),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), audioN)
}

// seedManyAudioSpecies adds n extra audio species to the deck, each with a
// quiz-quality recording and a card due NOW(). Because they're inserted in one
// transaction they all share an identical `due` timestamp -- the tie that
// GetNextDueCard must shuffle rather than resolve deterministically.
func seedManyAudioSpecies(t *testing.T, tx pgx.Tx, f fixtures, n int) {
	t.Helper()
	ctx := context.Background()
	for i := 0; i < n; i++ {
		code := fmt.Sprintf("_many%d", i)
		_, err := tx.Exec(ctx,
			`INSERT INTO species (ebird_code, common_name, scientific_name)
			 VALUES ($1, $2, 'Manyus testus')`, code, "Many "+code)
		require.NoError(t, err)
		_, err = tx.Exec(ctx,
			`INSERT INTO deck_species (deck_id, species_code) VALUES ($1, $2)`, f.deckID, code)
		require.NoError(t, err)
		_, err = tx.Exec(ctx,
			`INSERT INTO species_recordings (xeno_canto_id, species_code, file_path, quality, type, credit)
			 VALUES ($1, $2, 'https://r2.example.com/rec.mp3', 'A', 'song', 'tester')`,
			"_xc"+code, code)
		require.NoError(t, err)
		_, err = tx.Exec(ctx,
			`INSERT INTO cards (user_id, species_code, lane) VALUES ($1, $2, 'audio')`, f.userID, code)
		require.NoError(t, err)
	}
}

// GetCardsInTier

func TestGetCardsInTier_Window(t *testing.T) {
	pool := connectTestDB(t)
	tx := withTx(t, pool)
	f := seedFixtures(t, tx)
	q := store.New(tx)
	mustScheduleStability(t, q, f.userID, "_tst1", "audio", 10) // juvenile [7,30)

	// Juvenile window: min=7, max=30, not egg, not unbounded.
	rows, err := q.GetCardsInTier(context.Background(), store.GetCardsInTierParams{
		UserID:       f.userID,
		Egg:          false,
		MinStability: 7,
		Unbounded:    false,
		MaxStability: 30,
	})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "_tst1", rows[0].SpeciesCode)
	assert.Equal(t, "audio", rows[0].Lane)
}

func TestGetCardsInTier_Egg(t *testing.T) {
	pool := connectTestDB(t)
	tx := withTx(t, pool)
	f := seedFixtures(t, tx)
	q := store.New(tx)
	// Both _tst1 lanes are reps=0 -> egg.
	rows, err := q.GetCardsInTier(context.Background(), store.GetCardsInTierParams{
		UserID: f.userID, Egg: true,
	})
	require.NoError(t, err)
	assert.Len(t, rows, 2, "both never-quizzed lanes are eggs")
}

// ListRecordingsForTranscode

func TestListRecordingsForTranscode_ExcludesRowsWithPeaks(t *testing.T) {
	pool := connectTestDB(t)
	tx := withTx(t, pool)
	seedFixtures(t, tx)
	q := store.New(tx)
	ctx := context.Background()

	_, err := tx.Exec(ctx,
		`UPDATE species_recordings SET peaks = '{1,2,3}' WHERE xeno_canto_id = '_xc_tst1'`)
	require.NoError(t, err)

	rows, err := q.ListRecordingsForTranscode(ctx)
	require.NoError(t, err)
	for _, r := range rows {
		assert.NotEqual(t, "_xc_tst1", r.XenoCantoID,
			"a row with peaks already set must not be returned, so --limit only ever samples real work")
	}
}

// GetNextDueCard must not present tied-due cards in a fixed order: a fresh deck
// seeds every card with the same `due`, and a deterministic tiebreak made the
// quiz replay the identical species sequence every session. We bucket due by
// minute and randomise within the bucket, so repeated draws should surface
// more than one species.
func TestGetNextDueCard_ShufflesTiedDueCards(t *testing.T) {
	pool := connectTestDB(t)
	tx := withTx(t, pool)
	f := seedFixtures(t, tx)
	seedManyAudioSpecies(t, tx, f, 8)
	q := store.New(tx)

	seen := map[string]struct{}{}
	for i := 0; i < 25; i++ {
		card, err := q.GetNextDueCard(context.Background(), store.GetNextDueCardParams{
			UserID: f.userID,
			DeckID: f.deckID,
			Lane:   "audio",
		})
		require.NoError(t, err)
		seen[card.SpeciesCode] = struct{}{}
	}

	// 9 tied audio species, 25 draws: a fixed order yields exactly 1 distinct
	// species. Random-within-bucket makes all-same astronomically unlikely.
	assert.Greater(t, len(seen), 1,
		"GetNextDueCard returned the same species on every draw -- tied due cards are not being shuffled")
}
