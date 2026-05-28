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

// speciesStubQuerier stubs species list/search methods.
type speciesStubQuerier struct {
	store.Querier
	searchSpecies func(ctx context.Context, query pgtype.Text) ([]store.SearchSpeciesRow, error)
	listSpecies   func(ctx context.Context, arg store.ListSpeciesParams) ([]store.ListSpeciesRow, error)
}

func (s *speciesStubQuerier) SearchSpecies(ctx context.Context, query pgtype.Text) ([]store.SearchSpeciesRow, error) {
	return s.searchSpecies(ctx, query)
}

func (s *speciesStubQuerier) ListSpecies(ctx context.Context, arg store.ListSpeciesParams) ([]store.ListSpeciesRow, error) {
	return s.listSpecies(ctx, arg)
}

func TestListSpecies_NoPagination_ReturnsFirstPage(t *testing.T) {
	q := &speciesStubQuerier{
		listSpecies: func(_ context.Context, arg store.ListSpeciesParams) ([]store.ListSpeciesRow, error) {
			assert.Equal(t, int32(20), arg.Limit)
			assert.Equal(t, int32(0), arg.Offset)
			return []store.ListSpeciesRow{
				{EbirdCode: "amro", CommonName: "American Robin", ScientificName: "Turdus migratorius", TotalCount: 1},
			}, nil
		},
	}
	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/species", nil)
	r = injectUserID(r, 1)
	w := httptest.NewRecorder()

	h.listSpecies(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var body PaginatedSpecies
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, int64(1), body.Count)
	assert.Nil(t, body.Next)
	assert.Nil(t, body.Previous)
	assert.Len(t, body.Results, 1)
	assert.Equal(t, "American Robin", body.Results[0].CommonName)
}

func TestListSpecies_WithOffset_SetsPreviousAndNext(t *testing.T) {
	q := &speciesStubQuerier{
		listSpecies: func(_ context.Context, arg store.ListSpeciesParams) ([]store.ListSpeciesRow, error) {
			assert.Equal(t, int32(20), arg.Limit)
			assert.Equal(t, int32(20), arg.Offset)
			return []store.ListSpeciesRow{
				{EbirdCode: "amro", CommonName: "American Robin", ScientificName: "Turdus migratorius", TotalCount: 50},
			}, nil
		},
	}
	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/species?offset=20", nil)
	r.Host = "localhost:8080"
	r = injectUserID(r, 1)
	w := httptest.NewRecorder()

	h.listSpecies(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var body PaginatedSpecies
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, int64(50), body.Count)
	require.NotNil(t, body.Next)
	assert.Contains(t, *body.Next, "offset=40")
	require.NotNil(t, body.Previous)
	assert.Contains(t, *body.Previous, "offset=0")
}

func TestListSpecies_SearchMode_NoPaginationLinks(t *testing.T) {
	q := &speciesStubQuerier{
		searchSpecies: func(_ context.Context, query pgtype.Text) ([]store.SearchSpeciesRow, error) {
			assert.Equal(t, "sparrow", query.String)
			return []store.SearchSpeciesRow{
				{EbirdCode: "sonspa", CommonName: "Song Sparrow", ScientificName: "Melospiza melodia"},
			}, nil
		},
	}
	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/species?q=sparrow", nil)
	r = injectUserID(r, 1)
	w := httptest.NewRecorder()

	h.listSpecies(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var body PaginatedSpecies
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, int64(1), body.Count)
	assert.Nil(t, body.Next)
	assert.Nil(t, body.Previous)
	assert.Len(t, body.Results, 1)
	assert.Equal(t, "Song Sparrow", body.Results[0].CommonName)
}

func TestListSpecies_EmptyResults_ReturnsEmptySlice(t *testing.T) {
	q := &speciesStubQuerier{
		listSpecies: func(_ context.Context, arg store.ListSpeciesParams) ([]store.ListSpeciesRow, error) {
			return []store.ListSpeciesRow{}, nil
		},
	}
	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/species", nil)
	r = injectUserID(r, 1)
	w := httptest.NewRecorder()

	h.listSpecies(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var body PaginatedSpecies
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, int64(0), body.Count)
	assert.NotNil(t, body.Results) // must be [] not null
}
