package httpx_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jameynakama/flockdeck/internal/httpx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetJSON_DecodesBody(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"name":"Spotted Towhee"}`)) //nolint:errcheck
	}))
	defer ts.Close()

	var dest struct {
		Name string `json:"name"`
	}
	err := httpx.GetJSON(context.Background(), ts.Client(), ts.URL, nil, &dest)

	require.NoError(t, err)
	assert.Equal(t, "Spotted Towhee", dest.Name)
}

func TestGetJSON_SendsHeaders(t *testing.T) {
	var gotToken string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotToken = r.Header.Get("X-eBirdApiToken")
		w.Write([]byte(`{}`)) //nolint:errcheck
	}))
	defer ts.Close()

	var dest struct{}
	err := httpx.GetJSON(context.Background(), ts.Client(), ts.URL,
		http.Header{"X-eBirdApiToken": {"key123"}}, &dest)

	require.NoError(t, err)
	assert.Equal(t, "key123", gotToken)
}

func TestGetJSON_NonOK_ReturnsStatusError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "slow down", http.StatusTooManyRequests)
	}))
	defer ts.Close()

	var dest struct{}
	err := httpx.GetJSON(context.Background(), ts.Client(), ts.URL, nil, &dest)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "429")
}

func TestGetJSON_ContextCancelled_ReturnsError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Second)
	}))
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	var dest struct{}
	err := httpx.GetJSON(ctx, ts.Client(), ts.URL, nil, &dest)

	require.Error(t, err)
}

func TestDefaultClient_HasTimeout(t *testing.T) {
	assert.Positive(t, httpx.DefaultClient.Timeout,
		"upstream APIs must not be able to hang ingest workers forever")
}
