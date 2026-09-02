package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunFile_TranscodesLocallyWithoutNetworkOrDB(t *testing.T) {
	var out bytes.Buffer
	err := runFile(context.Background(), &out, "../../internal/audio/testdata/stereo.wav")
	require.NoError(t, err)

	got := out.String()
	assert.Contains(t, got, "mono")
	assert.Contains(t, got, "peaks")
	assert.Contains(t, got, ".transcoded.mp3")

	// The output lands next to the source so it can be listened to.
	assert.FileExists(t, "../../internal/audio/testdata/stereo.wav.transcoded.mp3")
	t.Cleanup(func() { os.Remove("../../internal/audio/testdata/stereo.wav.transcoded.mp3") })
}

func TestHTTPFetcher_DownloadsAndReportsSize(t *testing.T) {
	body := strings.Repeat("x", 1234)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(body))
	}))
	defer srv.Close()

	path, size, cleanup, err := httpFetcher(context.Background(), srv.URL)
	require.NoError(t, err)
	defer cleanup()

	assert.Equal(t, int64(1234), size)
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, body, string(got))
}

func TestHTTPFetcher_NonOKIsAnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "gone", http.StatusNotFound)
	}))
	defer srv.Close()

	_, _, cleanup, err := httpFetcher(context.Background(), srv.URL)
	defer cleanup()
	require.Error(t, err)
}
