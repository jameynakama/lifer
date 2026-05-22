package ebird

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaxonomy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/ref/taxonomy/ebird", r.URL.Path)
		assert.Equal(t, "json", r.URL.Query().Get("fmt"))
		assert.Equal(t, "species", r.URL.Query().Get("cat"))
		assert.Equal(t, "testkey", r.Header.Get("X-eBirdApiToken"))
		json.NewEncoder(w).Encode([]TaxonomyEntry{
			{SpeciesCode: "soospa", CommonName: "Song Sparrow", SciName: "Melospiza melodia", Category: "species"},
			{SpeciesCode: "norcaw", CommonName: "Northwestern Crow", SciName: "Corvus caurinus", Category: "species"},
		})
	}))
	defer srv.Close()

	c := newWithBaseURL("testkey", srv.URL)
	entries, err := c.Taxonomy(context.Background())
	require.NoError(t, err)
	assert.Len(t, entries, 2)
	assert.Equal(t, "soospa", entries[0].SpeciesCode)
	assert.Equal(t, "Melospiza melodia", entries[0].SciName)
}

func TestTaxonomyHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := newWithBaseURL("bad", srv.URL)
	_, err := c.Taxonomy(context.Background())
	assert.ErrorContains(t, err, "401")
}

func TestSpeciesList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/product/spplist/US-OR", r.URL.Path)
		assert.Equal(t, "testkey", r.Header.Get("X-eBirdApiToken"))
		json.NewEncoder(w).Encode([]string{"soospa", "norcaw", "mallar"})
	}))
	defer srv.Close()

	c := newWithBaseURL("testkey", srv.URL)
	codes, err := c.SpeciesList(context.Background(), "US-OR")
	require.NoError(t, err)
	assert.Equal(t, []string{"soospa", "norcaw", "mallar"}, codes)
}

func TestSpeciesListHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := newWithBaseURL("bad", srv.URL)
	_, err := c.SpeciesList(context.Background(), "US-OR")
	assert.ErrorContains(t, err, "401")
}
