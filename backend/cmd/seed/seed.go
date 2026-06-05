package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	fsrs "github.com/open-spaced-repetition/go-fsrs/v3"

	"github.com/jameynakama/flockdeck/internal/store"
)

type seedResult struct {
	Days, Skipped, Reviews, Correct int
	Seeded                          int64
	ByLane                          map[string]int
}

// runSeed resets the user (cards + review_log deleted, blank cards
// re-seeded -- the same operations as POST /api/v1/reset scope=everything)
// and then replays `days` of simulated study through the real FSRS
// scheduler. Backdated writes (cards.last_review, review_log.reviewed_at)
// use raw SQL: the app's sqlc queries pin those columns to NOW() on purpose.
// A crash mid-replay leaves a shorter but self-consistent history.
func runSeed(ctx context.Context, pool *pgxpool.Pool, userID int64, days int, rng *rand.Rand, now time.Time) (seedResult, error) {
	q := store.New(pool)
	res := seedResult{Days: days, ByLane: map[string]int{}}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return res, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	qtx := store.New(tx)
	if _, err := qtx.DeleteAllCardsForUser(ctx, userID); err != nil {
		return res, fmt.Errorf("delete cards: %w", err)
	}
	if _, err := qtx.DeleteAllReviewsForUser(ctx, userID); err != nil {
		return res, fmt.Errorf("delete reviews: %w", err)
	}
	if res.Seeded, err = qtx.SeedCardsForUserDecks(ctx, userID); err != nil {
		return res, fmt.Errorf("seed cards: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return res, err
	}
	if res.Seeded == 0 {
		return res, fmt.Errorf("user has no deck species to practice -- clone a preset deck first")
	}

	deckRows, err := q.GetUserDeckSpecies(ctx, userID)
	if err != nil {
		return res, err
	}
	species := make([]deckSpecies, len(deckRows))
	for i, r := range deckRows {
		species[i] = deckSpecies{code: r.EbirdCode, family: r.Family.String}
	}

	// Fresh cards are due "now" (column default); backdate them so the
	// first simulated day sees them due.
	start := now.Add(-time.Duration(days) * 24 * time.Hour)
	if _, err := pool.Exec(ctx,
		`UPDATE cards SET due = $2 WHERE user_id = $1`, userID, start); err != nil {
		return res, err
	}

	f := fsrs.NewFSRS(fsrs.DefaultParam())

	for day := 0; day < days; day++ {
		if rng.Float64() < skipDayOdds {
			res.Skipped++
			continue
		}
		// One session per day, sometime between 08:00 and 18:00 sim-time.
		sessionStart := start.Add(time.Duration(day)*24*time.Hour +
			time.Duration(8+rng.Intn(10))*time.Hour)

		var asOf pgtype.Timestamptz
		if err := asOf.Scan(sessionStart); err != nil {
			return res, err
		}
		cards, err := q.GetDueCardsForUser(ctx, store.GetDueCardsForUserParams{
			UserID: userID,
			AsOf:   asOf,
		})
		if err != nil {
			return res, err
		}

		for i, c := range cards {
			at := sessionStart.Add(time.Duration(i*20+rng.Intn(15)) * time.Second)
			correct := rng.Float64() < accuracyOn(day, days)

			var guess *string
			if correct {
				g := c.SpeciesCode
				guess = &g
			} else if rng.Float64() >= dontKnowOdds {
				if g := pickConfusable(rng, species, c.SpeciesCode); g != "" {
					guess = &g
				}
			}

			media, err := q.GetRandomMediaForSpecies(ctx, c.SpeciesCode)
			if err != nil {
				return res, fmt.Errorf("media for %s: %w", c.SpeciesCode, err)
			}
			mediaID := media.AudioID
			if c.Lane == "image" {
				mediaID = media.ImageID
			}

			if err := applyReview(ctx, pool, f, c, correct, guess, mediaID, at); err != nil {
				return res, fmt.Errorf("review %s/%s: %w", c.SpeciesCode, c.Lane, err)
			}
			res.Reviews++
			res.ByLane[c.Lane]++
			if correct {
				res.Correct++
			}
		}
	}
	return res, nil
}

// applyReview advances one card through FSRS at the simulated time and
// records the matching review_log row, both backdated, in one tx.
func applyReview(ctx context.Context, pool *pgxpool.Pool, f *fsrs.FSRS, c store.Card, correct bool, guess *string, mediaID string, at time.Time) error {
	card := fsrs.Card{
		Stability:  c.Stability,
		Difficulty: c.Difficulty,
		Reps:       uint64(c.Reps),
		Lapses:     uint64(c.Lapses),
		State:      fsrs.State(c.State),
	}
	if c.LastReview.Valid {
		card.LastReview = c.LastReview.Time
	}
	if c.Due.Valid {
		card.Due = c.Due.Time
	}

	rating := 1 // Again
	if correct {
		rating = 3 // Good (quiz auto-rating: correct <=> 3)
	}
	next := f.Next(card, at, fsrs.Rating(rating)).Card

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if _, err := tx.Exec(ctx,
		`UPDATE cards
		 SET stability = $2, difficulty = $3, due = $4, last_review = $5,
		     reps = $6, lapses = $7, state = $8
		 WHERE id = $1`,
		c.ID, next.Stability, next.Difficulty, next.Due, at,
		int32(next.Reps), int32(next.Lapses), int16(next.State)); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO review_log (user_id, species_code, lane, rating,
		         guessed_species_code, media_id, reviewed_at)
		 VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), $7)`,
		c.UserID, c.SpeciesCode, c.Lane, int16(rating), guess, mediaID, at); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
