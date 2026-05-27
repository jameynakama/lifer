package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jameynakama/lifer/internal/ebird"
	"github.com/jameynakama/lifer/internal/r2"
)

func TestFilterBySpecies(t *testing.T) {
	taxMap := map[string]ebird.TaxonomyEntry{
		"busti": {SpeciesCode: "busti", CommonName: "Bushtit"},
		"rukin": {SpeciesCode: "rukin", CommonName: "Ruby-crowned Kinglet"},
		"amro":  {SpeciesCode: "amro", CommonName: "American Robin"},
	}
	codes := []string{"busti", "rukin", "amro"}

	tests := []struct {
		name    string
		want    []string
		wantOut []string
	}{
		{"ebird code", []string{"busti"}, []string{"busti"}},
		{"common name case-insensitive", []string{"bushtit"}, []string{"busti"}},
		{"mixed code and name", []string{"busti", "ruby-crowned kinglet"}, []string{"busti", "rukin"}},
		{"multiple codes", []string{"rukin", "amro"}, []string{"rukin", "amro"}},
		{"no match is excluded", []string{"busti", "doesnotexist"}, []string{"busti"}},
		{"empty want", []string{}, []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterBySpecies(codes, taxMap, tt.want)
			if len(got) != len(tt.wantOut) {
				t.Fatalf("filterBySpecies() len = %d, want %d (got %v, want %v)", len(got), len(tt.wantOut), got, tt.wantOut)
			}
			for i := range tt.wantOut {
				if got[i] != tt.wantOut[i] {
					t.Errorf("filterBySpecies()[%d] = %q, want %q", i, got[i], tt.wantOut[i])
				}
			}
		})
	}
}

func TestFilterComplete(t *testing.T) {
	tests := []struct {
		name     string
		codes    []string
		complete map[string]struct{}
		want     []string
	}{
		{
			name:     "empty complete set passes all through",
			codes:    []string{"AMRO", "BCCH", "NOCA"},
			complete: map[string]struct{}{},
			want:     []string{"AMRO", "BCCH", "NOCA"},
		},
		{
			name:     "complete species are removed",
			codes:    []string{"AMRO", "BCCH", "NOCA"},
			complete: map[string]struct{}{"BCCH": {}},
			want:     []string{"AMRO", "NOCA"},
		},
		{
			name:     "all complete returns empty slice",
			codes:    []string{"AMRO", "BCCH"},
			complete: map[string]struct{}{"AMRO": {}, "BCCH": {}},
			want:     []string{},
		},
		{
			name:     "nil codes returns empty slice",
			codes:    nil,
			complete: map[string]struct{}{"AMRO": {}},
			want:     []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterComplete(tt.codes, tt.complete)
			if len(got) != len(tt.want) {
				t.Fatalf("filterComplete() len = %d, want %d (got %v, want %v)", len(got), len(tt.want), got, tt.want)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("filterComplete()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestFetchAndUpload_Success(t *testing.T) {
	src := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("audio bytes"))
	}))
	defer src.Close()

	var uploaded string
	r2s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			b, _ := io.ReadAll(r.Body)
			uploaded = string(b)
			w.Header().Set("ETag", `"x"`)
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer r2s.Close()

	r2c, err := r2.NewWithEndpoint(r2s.URL, "k", "s", "bucket", "https://pub.example.com")
	if err != nil {
		t.Fatalf("Should create r2 client: %v", err)
	}

	url, err := fetchAndUpload(context.Background(), r2c, src.URL, "recordings/busti/123.mp3", "audio/mpeg")
	if err != nil {
		t.Fatalf("Should fetch and upload without error: %v", err)
	}
	if url != "https://pub.example.com/recordings/busti/123.mp3" {
		t.Errorf("Should return public URL, got %q", url)
	}
	if uploaded != "audio bytes" {
		t.Errorf("Should upload source body, got %q", uploaded)
	}
}

func TestFetchAndUpload_SourceNonOK_ReturnsError(t *testing.T) {
	src := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer src.Close()

	r2c, _ := r2.NewWithEndpoint("http://localhost:1", "k", "s", "bucket", "https://pub.example.com")
	_, err := fetchAndUpload(context.Background(), r2c, src.URL, "key", "audio/mpeg")
	if err == nil {
		t.Error("Should return error for non-200 source response")
	}
}

func TestFetchAndUpload_Retries429(t *testing.T) {
	origDelays := retryDelays
	retryDelays = []time.Duration{0, 0, 0}
	t.Cleanup(func() { retryDelays = origDelays })

	attempts := 0
	src := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Write([]byte("ok"))
	}))
	defer src.Close()

	r2s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", `"x"`)
		w.WriteHeader(http.StatusOK)
	}))
	defer r2s.Close()

	r2c, _ := r2.NewWithEndpoint(r2s.URL, "k", "s", "bucket", "https://pub.example.com")
	_, err := fetchAndUpload(context.Background(), r2c, src.URL, "key", "audio/mpeg")
	if err != nil {
		t.Fatalf("Should succeed after retries: %v", err)
	}
	if attempts != 3 {
		t.Errorf("Should take 3 attempts, got %d", attempts)
	}
}
