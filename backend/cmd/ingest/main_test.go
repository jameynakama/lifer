package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestDownloadFile(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("bird audio"))
	}))
	defer ts.Close()

	dest := filepath.Join(t.TempDir(), "song.mp3")
	if err := downloadFile(context.Background(), ts.URL, dest); err != nil {
		t.Fatalf("Should download without error: %v", err)
	}
	got, _ := os.ReadFile(dest)
	if string(got) != "bird audio" {
		t.Errorf("Should write response body: got %q", got)
	}
}

func TestDownloadFileNonOKStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	err := downloadFile(context.Background(), ts.URL, filepath.Join(t.TempDir(), "song.mp3"))
	if err == nil {
		t.Error("Should return error for non-200 response")
	}
}

func TestDownloadFileConcurrentCallsSucceed(t *testing.T) {
	var served atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		served.Add(1)
		time.Sleep(20 * time.Millisecond)
		w.Write([]byte("data"))
	}))
	defer ts.Close()

	dir := t.TempDir()
	// Simulate the parallel fan-out that ingestSpecies now does.
	type result struct{ err error }
	results := make(chan result, 4)
	for i := range 4 {
		go func(i int) {
			err := downloadFile(context.Background(), ts.URL, filepath.Join(dir, filepath.FromSlash(fmt.Sprintf("file%d", i))))
			results <- result{err}
		}(i)
	}
	deadline := time.After(200 * time.Millisecond) // sequential would take ~80ms; concurrent << 200ms
	for range 4 {
		select {
		case r := <-results:
			if r.err != nil {
				t.Errorf("Should download without error: %v", r.err)
			}
		case <-deadline:
			t.Fatal("Should complete 4 concurrent downloads within 200ms")
		}
	}
	if n := served.Load(); n != 4 {
		t.Errorf("Should serve 4 requests, got %d", n)
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
