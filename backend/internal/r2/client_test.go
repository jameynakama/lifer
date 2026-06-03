package r2_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jameynakama/flockdeck/internal/r2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpload_PutsObjectAndReturnsPublicURL(t *testing.T) {
	var gotMethod, gotPath, gotBody, gotContentType string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		gotContentType = r.Header.Get("Content-Type")
		w.Header().Set("ETag", `"abc123"`)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	c, err := r2.NewWithEndpoint(ts.URL, "key", "secret", "flockdeck", "https://pub.example.com")
	require.NoError(t, err)

	url, err := c.Upload(context.Background(), "recordings/busti/123.mp3", "audio/mpeg", strings.NewReader("audio data"))
	require.NoError(t, err)

	assert.Equal(t, "https://pub.example.com/recordings/busti/123.mp3", url)
	assert.Equal(t, http.MethodPut, gotMethod)
	assert.Contains(t, gotPath, "recordings/busti/123.mp3")
	assert.Equal(t, "audio data", gotBody)
	assert.Equal(t, "audio/mpeg", gotContentType)
}

func TestUpload_R2ErrorReturnsError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `<?xml version="1.0"?><Error><Code>InternalError</Code><Message>oops</Message></Error>`, http.StatusInternalServerError)
	}))
	defer ts.Close()

	c, err := r2.NewWithEndpoint(ts.URL, "key", "secret", "flockdeck", "https://pub.example.com")
	require.NoError(t, err)

	_, err = c.Upload(context.Background(), "recordings/busti/123.mp3", "audio/mpeg", strings.NewReader("data"))
	require.Error(t, err)
}
