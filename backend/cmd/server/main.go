package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jameynakama/flockdeck/internal/api"
	"github.com/jameynakama/flockdeck/internal/auth"
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
	}
}

func main() {
	cfg := loadConfig()

	pool, err := pgxpool.New(context.Background(), cfg.databaseURL)
	if err != nil {
		log.Fatalf("unable to connect to database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		log.Fatalf("database ping failed: %v", err)
	}
	log.Println("database connected")

	queries := store.New(pool)
	oauthCfg := auth.NewGoogleConfig(cfg.googleClientID, cfg.googleSecret, cfg.googleRedirectURL)

	router := api.NewRouter(api.RouterConfig{
		Queries:     queries,
		OAuthConfig: oauthCfg,
		JWTSecret:   cfg.jwtSecret,
		FrontendURL: cfg.frontendURL,
	})

	addr := fmt.Sprintf(":%s", cfg.port)
	log.Printf("server listening on %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
