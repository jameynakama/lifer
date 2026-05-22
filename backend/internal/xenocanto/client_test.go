package xenocanto

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/2/recordings", r.URL.Path)
		q := r.URL.Query().Get("query")
		assert.Contains(t, q, "gen:Melospiza")
		assert.Contains(t, q, "sp:melodia")
		assert.Contains(t, q, "type:song")
		json.NewEncoder(w).Encode(apiResponse{
			Recordings: []Recording{
				{ID: "111", Type: "song", Quality: "A", FileURL: "//example.com/111.mp3"},
				{ID: "222", Type: "song", Quality: "B", FileURL: "//example.com/222.mp3"},
				{ID: "333", Type: "song", Quality: "C", FileURL: "//example.com/333.mp3"},
			},
		})
	}))
	defer srv.Close()

	c := newWithBaseURL("", srv.URL)
	recs, err := c.Search(context.Background(), "Melospiza", "melodia", "song")
	require.NoError(t, err)
	// quality C is filtered out
	assert.Len(t, recs, 2)
	assert.Equal(t, "111", recs[0].ID)
	assert.Equal(t, "A", recs[0].Quality)
	assert.Equal(t, "https://example.com/111.mp3", recs[0].FileURL)
	assert.Equal(t, "222", recs[1].ID)
}

func TestSearchHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	c := newWithBaseURL("", srv.URL)
	_, err := c.Search(context.Background(), "Melospiza", "melodia", "song")
	assert.ErrorContains(t, err, "429")
}
