package main

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAccuracyOn_RampsLinearly(t *testing.T) {
	assert.InDelta(t, 0.65, accuracyOn(0, 14), 1e-9)
	assert.InDelta(t, 0.90, accuracyOn(13, 14), 1e-9)
	assert.Greater(t, accuracyOn(7, 14), accuracyOn(0, 14))
	// A 1-day run is just "current ability": the end accuracy.
	assert.InDelta(t, 0.90, accuracyOn(0, 1), 1e-9)
}

func TestPickConfusable_PrefersSameFamily(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	species := []deckSpecies{
		{code: "foxspa", family: "New World Sparrows"},
		{code: "sonspa", family: "New World Sparrows"},
		{code: "amerob", family: "Thrushes"},
	}
	for range 50 {
		assert.Equal(t, "sonspa", pickConfusable(rng, species, "foxspa"),
			"with a same-family option it must always be picked")
	}
}

func TestPickConfusable_FallsBackAcrossFamilies(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	species := []deckSpecies{
		{code: "foxspa", family: "New World Sparrows"},
		{code: "amerob", family: "Thrushes"},
	}
	assert.Equal(t, "amerob", pickConfusable(rng, species, "foxspa"))
}

func TestPickConfusable_EmptyWhenNothingElse(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	assert.Equal(t, "", pickConfusable(rng, []deckSpecies{{code: "foxspa"}}, "foxspa"))
}

func TestRequireLocalhost(t *testing.T) {
	assert.NoError(t, requireLocalhost("postgres://u:p@localhost:5435/flockdeck?sslmode=disable"))
	assert.NoError(t, requireLocalhost("postgres://u:p@127.0.0.1:5432/db"))
	assert.Error(t, requireLocalhost("postgres://u:p@db.example.com:5432/db"))
	assert.Error(t, requireLocalhost("postgres://u:p@164.92.10.10:25060/db"))
	assert.Error(t, requireLocalhost("not a url at all ::"))
}
