package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServer_SetsTimeouts(t *testing.T) {
	srv := newServer(":0", http.NewServeMux())

	assert.Equal(t, ":0", srv.Addr)
	assert.NotNil(t, srv.Handler)
	assert.Positive(t, srv.ReadHeaderTimeout, "ReadHeaderTimeout guards against slowloris")
	assert.Positive(t, srv.ReadTimeout)
	assert.Positive(t, srv.WriteTimeout)
	assert.Positive(t, srv.IdleTimeout)
}

func TestRunServer_ServesRequests(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	srv := newServer(":0", mux)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- runServer(ctx, srv, ln) }()

	resp, err := http.Get(fmt.Sprintf("http://%s/ping", ln.Addr()))
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	cancel()
	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("server did not shut down after context cancellation")
	}
}

func TestRunServer_ContextCancel_ShutsDownGracefully(t *testing.T) {
	srv := newServer(":0", http.NewServeMux())

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- runServer(ctx, srv, ln) }()

	cancel()

	select {
	case err := <-done:
		assert.NoError(t, err, "graceful shutdown is not an error")
	case <-time.After(5 * time.Second):
		t.Fatal("server did not shut down after context cancellation")
	}
}
