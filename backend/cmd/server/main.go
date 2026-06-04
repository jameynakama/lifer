package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jameynakama/flockdeck/internal/api"
	"github.com/jameynakama/flockdeck/internal/auth"
	"github.com/jameynakama/flockdeck/internal/r2"
	"github.com/jameynakama/flockdeck/internal/store"
)

type config struct {
	databaseURL       string
	port              string
	googleClientID    string
	googleSecret      string
	googleRedirectURL string
	jwtSecret         []byte
	frontendURL       string
	r2AccountID       string
	r2AccessKey       string
	r2SecretKey       string
	r2Bucket          string
	r2PublicURL       string
}

func loadConfig() config {
	required := func(key string) string {
		v := os.Getenv(key)
		if v == "" {
			log.Fatalf("%s is required", key)
		}
		return v
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return config{
		databaseURL:       required("DATABASE_URL"),
		googleClientID:    required("GOOGLE_CLIENT_ID"),
		googleSecret:      required("GOOGLE_CLIENT_SECRET"),
		googleRedirectURL: required("GOOGLE_REDIRECT_URL"),
		jwtSecret:         []byte(required("JWT_SECRET")),
		frontendURL:       required("FRONTEND_URL"),
		port:              port,
		r2AccountID:       required("R2_ACCOUNT_ID"),
		r2AccessKey:       required("R2_ACCESS_KEY_ID"),
		r2SecretKey:       required("R2_SECRET_ACCESS_KEY"),
		r2Bucket:          required("R2_BUCKET_NAME"),
		r2PublicURL:       required("R2_PUBLIC_URL"),
	}
}

// newServer returns an http.Server with timeouts set so slow or stalled
// clients cannot hold connections open indefinitely. ReadTimeout is generous
// because admin media uploads (multipart, up to ~32MB) ride through it.
func newServer(addr string, h http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       2 * time.Minute,
		WriteTimeout:      2 * time.Minute,
		IdleTimeout:       2 * time.Minute,
	}
}

// runServer serves on ln until ctx is cancelled, then shuts down gracefully.
// A clean shutdown returns nil.
func runServer(ctx context.Context, srv *http.Server, ln net.Listener) error {
	errCh := make(chan error, 1)
	go func() { errCh <- srv.Serve(ln) }()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func main() {
	cfg := loadConfig()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.databaseURL)
	if err != nil {
		log.Fatalf("unable to connect to database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("database ping failed: %v", err)
	}
	log.Println("database connected")

	queries := store.New(pool)
	oauthCfg := auth.NewGoogleConfig(cfg.googleClientID, cfg.googleSecret, cfg.googleRedirectURL)

	r2c, err := r2.New(cfg.r2AccountID, cfg.r2AccessKey, cfg.r2SecretKey, cfg.r2Bucket, cfg.r2PublicURL)
	if err != nil {
		log.Fatalf("r2 client: %v", err)
	}

	router := api.NewRouter(api.RouterConfig{
		Queries:     queries,
		OAuthConfig: oauthCfg,
		JWTSecret:   cfg.jwtSecret,
		FrontendURL: cfg.frontendURL,
		R2Client:    r2c,
	})

	addr := fmt.Sprintf(":%s", cfg.port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	log.Printf("server listening on %s", addr)
	if err := runServer(ctx, newServer(addr, router), ln); err != nil {
		log.Fatalf("server error: %v", err)
	}
	log.Println("server shut down cleanly")
}
