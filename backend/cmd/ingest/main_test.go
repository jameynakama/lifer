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

func nopSend(any) {}

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

	url, err := fetchAndUpload(context.Background(), r2c, src.URL, "recordings/busti/123.mp3", "audio/mpeg", 0, nopSend)
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
	_, err := fetchAndUpload(context.Background(), r2c, src.URL, "key", "audio/mpeg", 0, nopSend)
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
	_, err := fetchAndUpload(context.Background(), r2c, src.URL, "key", "audio/mpeg", 0, nopSend)
	if err != nil {
		t.Fatalf("Should succeed after retries: %v", err)
	}
	if attempts != 3 {
		t.Errorf("Should take 3 attempts, got %d", attempts)
	}
}

func TestFetchAndUpload_Retries503(t *testing.T) {
	origDelays := retryDelays
	retryDelays = []time.Duration{0, 0, 0}
	t.Cleanup(func() { retryDelays = origDelays })

	attempts := 0
	src := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
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
	_, err := fetchAndUpload(context.Background(), r2c, src.URL, "key", "audio/mpeg", 0, nopSend)
	if err != nil {
		t.Fatalf("Should succeed after 503 retries: %v", err)
	}
	if attempts != 3 {
		t.Errorf("Should take 3 attempts, got %d", attempts)
	}
}

func TestFetchAndUpload_SendsMessageSequence(t *testing.T) {
	src := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("data"))
	}))
	defer src.Close()

	r2s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", `"x"`)
		w.WriteHeader(http.StatusOK)
	}))
	defer r2s.Close()

	r2c, _ := r2.NewWithEndpoint(r2s.URL, "k", "s", "bucket", "https://pub.example.com")

	var msgs []any
	send := func(msg any) { msgs = append(msgs, msg) }

	_, err := fetchAndUpload(context.Background(), r2c, src.URL, "recordings/amro/XC1.mp3", "audio/mpeg", 2, send)
	if err != nil {
		t.Fatalf("Should succeed: %v", err)
	}

	if len(msgs) != 3 {
		t.Fatalf("Should send 3 messages (fetch, upload, done), got %d: %v", len(msgs), msgs)
	}
	if _, ok := msgs[0].(fetchStartedMsg); !ok {
		t.Errorf("msg[0] should be fetchStartedMsg, got %T", msgs[0])
	}
	if _, ok := msgs[1].(uploadStartedMsg); !ok {
		t.Errorf("msg[1] should be uploadStartedMsg, got %T", msgs[1])
	}
	done, ok := msgs[2].(uploadDoneMsg)
	if !ok {
		t.Errorf("msg[2] should be uploadDoneMsg, got %T", msgs[2])
	}
	if done.err != nil {
		t.Errorf("uploadDoneMsg should have nil err, got %v", done.err)
	}
	if done.workerID != 2 {
		t.Errorf("Should carry workerID 2, got %d", done.workerID)
	}
}
