package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jameynakama/lifer/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type speciesStubQuerier struct {
	store.Querier
	searchSpecies func(ctx context.Context, query pgtype.Text) ([]store.SearchSpeciesRow, error)
}

func (s *speciesStubQuerier) SearchSpecies(ctx context.Context, query pgtype.Text) ([]store.SearchSpeciesRow, error) {
	return s.searchSpecies(ctx, query)
}

func TestSearchSpecies_ReturnsMatches(t *testing.T) {
	q := &speciesStubQuerier{
		searchSpecies: func(_ context.Context, query pgtype.Text) ([]store.SearchSpeciesRow, error) {
			assert.Equal(t, "sparrow", query.String)
			return []store.SearchSpeciesRow{
				{EbirdCode: "sonspa", CommonName: "Song Sparrow", ScientificName: "Melospiza melodia"},
				{EbirdCode: "foxspa", CommonName: "Fox Sparrow", ScientificName: "Passerella iliaca"},
			}, nil
		},
	}
	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/species?q=sparrow", nil)
	r = injectUserID(r, 1)
	w := httptest.NewRecorder()

	h.searchSpecies(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var body []store.SearchSpeciesRow
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Len(t, body, 2)
	assert.Equal(t, "Song Sparrow", body[0].CommonName)
}

func TestSearchSpecies_EmptyQuery_ReturnsEmpty(t *testing.T) {
	q := &speciesStubQuerier{
		searchSpecies: func(_ context.Context, query pgtype.Text) ([]store.SearchSpeciesRow, error) {
			return []store.SearchSpeciesRow{}, nil
		},
	}
	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/species?q=", nil)
	r = injectUserID(r, 1)
	w := httptest.NewRecorder()

	h.searchSpecies(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
}
