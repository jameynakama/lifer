package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jameynakama/flockdeck/internal/store"
)

func main() {
	days := flag.Int("days", 14, "days of study history to simulate")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: seed [flags] <email>\n\n")
		fmt.Fprintf(os.Stderr, "Resets the user's cards and review history, then replays simulated\n")
		fmt.Fprintf(os.Stderr, "study through the real FSRS scheduler. Local databases only.\n")
		fmt.Fprintf(os.Stderr, "A crash mid-run leaves a shorter but self-consistent history; rerun.\n\n")
		fmt.Fprintf(os.Stderr, "Example:\n  seed jamey@example.com --days 30\n\nFlags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}
	email := flag.Arg(0)
	if *days < 1 {
		log.Fatal("--days must be >= 1")
	}

	dbURL := mustEnv("DATABASE_URL")
	if err := requireLocalhost(dbURL); err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	user, err := store.New(pool).GetUserByEmail(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		log.Fatalf("no user with email %s (sign in locally at least once first)", email)
	}
	if err != nil {
		log.Fatalf("look up user: %v", err)
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	res, err := runSeed(ctx, pool, user.ID, *days, rng, time.Now())
	if err != nil {
		log.Fatal(err)
	}

	pct := 0
	if res.Reviews > 0 {
		pct = res.Correct * 100 / res.Reviews
	}
	fmt.Printf("seeded %s: %d days (%d skipped), %d reviews (%d%% correct), %d cards seeded, audio %d / image %d\n",
		email, res.Days, res.Skipped, res.Reviews, pct, res.Seeded,
		res.ByLane["audio"], res.ByLane["image"])
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env var %s is not set", key)
	}
	return v
}
