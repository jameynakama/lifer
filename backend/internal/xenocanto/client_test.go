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
		assert.Equal(t, "/api/3/recordings", r.URL.Path)
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

	c := NewWithBaseURL("", srv.URL)
	recs, err := c.Search(context.Background(), "Melospiza", "melodia", "song", 0)
	require.NoError(t, err)
	// quality C is filtered out
	assert.Len(t, recs, 2)
	assert.Equal(t, "111", recs[0].ID)
	assert.Equal(t, "A", recs[0].Quality)
	assert.Equal(t, "https://example.com/111.mp3", recs[0].FileURL)
	assert.Equal(t, "222", recs[1].ID)
}

func TestParseLengthSecs(t *testing.T) {
	tests := []struct {
		in   string
		want int
	}{
		{"1:54", 114},
		{"16:59", 1019},
		{"0:30", 30},
		{"", 0},
		{"bad", 0},
	}
	for _, tt := range tests {
		if got := parseLengthSecs(tt.in); got != tt.want {
			t.Errorf("parseLengthSecs(%q) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

func TestFilterAndNormalize_LengthCap(t *testing.T) {
	recs := []Recording{
		{ID: "short", Quality: "A", Length: "1:00", FileURL: "https://example.com/short.mp3"},
		{ID: "long", Quality: "A", Length: "5:00", FileURL: "https://example.com/long.mp3"},
		{ID: "exact", Quality: "A", Length: "3:00", FileURL: "https://example.com/exact.mp3"},
	}
	got := filterAndNormalize(recs, 180) // 3 min cap
	assert.Len(t, got, 2)
	assert.Equal(t, "short", got[0].ID)
	assert.Equal(t, "exact", got[1].ID)
}

func TestSearchHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	c := NewWithBaseURL("", srv.URL)
	_, err := c.Search(context.Background(), "Melospiza", "melodia", "song", 0)
	assert.ErrorContains(t, err, "429")
}

func TestSearchIncludesAPIKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "mykey", r.URL.Query().Get("key"))
		json.NewEncoder(w).Encode(apiResponse{})
	}))
	defer srv.Close()

	c := NewWithBaseURL("mykey", srv.URL)
	recs, err := c.Search(context.Background(), "Melospiza", "melodia", "song", 0)
	require.NoError(t, err)
	assert.Empty(t, recs)
}

func TestFilterAndNormalize(t *testing.T) {
	tests := []struct {
		name string
		in   []Recording
		want []Recording
	}{
		{
			name: "empty input",
			in:   nil,
			want: nil,
		},
		{
			name: "all filtered out",
			in: []Recording{
				{ID: "1", Quality: "C"},
				{ID: "2", Quality: "D"},
				{ID: "3", Quality: "E"},
			},
			want: nil,
		},
		{
			name: "empty file URL is skipped",
			in: []Recording{
				{ID: "1", Quality: "A", FileURL: ""},
				{ID: "2", Quality: "A", FileURL: "https://example.com/2.mp3"},
			},
			want: []Recording{
				{ID: "2", Quality: "A", FileURL: "https://example.com/2.mp3"},
			},
		},
		{
			name: "A before B",
			in: []Recording{
				{ID: "b", Quality: "B", FileURL: "https://example.com/b.mp3"},
				{ID: "a", Quality: "A", FileURL: "https://example.com/a.mp3"},
			},
			want: []Recording{
				{ID: "a", Quality: "A", FileURL: "https://example.com/a.mp3"},
				{ID: "b", Quality: "B", FileURL: "https://example.com/b.mp3"},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := filterAndNormalize(tc.in, 0)
			assert.Equal(t, tc.want, got)
		})
	}
}
