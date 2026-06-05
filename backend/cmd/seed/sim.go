package main

import (
	"fmt"
	"math/rand"
	"net/url"
)

// Simulation shape: opinionated constants, not flags (per spec).
const (
	startAccuracy = 0.65
	endAccuracy   = 0.90
	skipDayOdds   = 1.0 / 7.0 // "regular-ish": about one missed day a week
	dontKnowOdds  = 0.15      // fraction of wrong answers that are skips
)

// accuracyOn ramps linearly from startAccuracy on day 0 to endAccuracy on
// the final day, modeling a learner improving over the run.
func accuracyOn(day, totalDays int) float64 {
	if totalDays <= 1 {
		return endAccuracy
	}
	frac := float64(day) / float64(totalDays-1)
	return startAccuracy + (endAccuracy-startAccuracy)*frac
}

type deckSpecies struct {
	code   string
	family string // "" when not backfilled
}

// pickConfusable returns a wrong-answer guess for actual: a random other
// deck species, always preferring the same eBird family (that's what makes
// the confusion-pairs panel look human). Returns "" when the deck has
// nothing else to confuse with (caller treats it as "I don't know").
func pickConfusable(rng *rand.Rand, species []deckSpecies, actual string) string {
	var actualFamily string
	for _, s := range species {
		if s.code == actual {
			actualFamily = s.family
		}
	}
	var sameFamily, others []string
	for _, s := range species {
		if s.code == actual {
			continue
		}
		if actualFamily != "" && s.family == actualFamily {
			sameFamily = append(sameFamily, s.code)
		}
		others = append(others, s.code)
	}
	if len(sameFamily) > 0 {
		return sameFamily[rng.Intn(len(sameFamily))]
	}
	if len(others) > 0 {
		return others[rng.Intn(len(others))]
	}
	return ""
}

// requireLocalhost refuses any DATABASE_URL whose host isn't local. There is
// deliberately no override: DATABASE_URL on this machine sometimes points at
// prod (ingest workflow), and the seeder starts by deleting a user's data.
func requireLocalhost(dbURL string) error {
	u, err := url.Parse(dbURL)
	if err != nil || u.Hostname() == "" {
		return fmt.Errorf("cannot parse DATABASE_URL host: %q", dbURL)
	}
	if host := u.Hostname(); host != "localhost" && host != "127.0.0.1" {
		return fmt.Errorf("refusing to seed non-local database host %q (no override exists)", host)
	}
	return nil
}
