// Command transcode re-encodes stored recordings to mono 96 kbps mp3,
// peak-normalized, and records their waveform peaks.
//
// It overwrites each object in place at its existing key, so no file_path row
// changes. Dry run is the default: the only database worth pointing this at is
// production, and the R2 bucket is shared across environments, so the
// destructive mode must never be what a fumbled flag gives you.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jameynakama/flockdeck/internal/audio"
	"github.com/jameynakama/flockdeck/internal/r2"
	"github.com/jameynakama/flockdeck/internal/store"
)

func main() {
	apply := flag.Bool("apply", false, "actually upload and write peaks (default is a dry run that writes nothing)")
	limit := flag.Int("limit", 0, "process at most this many recordings (0 = all)")
	speciesFilter := flag.String("species", "", "comma-separated ebird codes to process")
	workers := flag.Int("workers", 4, "concurrent worker count (ffmpeg is CPU-bound)")
	file := flag.String("file", "", "transcode a local file and exit; no DB, no R2, no env needed")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: transcode [flags]\n\n")
		fmt.Fprintf(os.Stderr, "Re-encodes stored recordings to mono 96 kbps mp3, peak-normalized to\n")
		fmt.Fprintf(os.Stderr, "-1 dBFS, and records waveform peaks. Objects are overwritten in place\n")
		fmt.Fprintf(os.Stderr, "at their existing keys, so no file_path row changes.\n\n")
		fmt.Fprintf(os.Stderr, "Dry run is the DEFAULT. Pass --apply to write.\n")
		fmt.Fprintf(os.Stderr, "Ctrl+c stops cleanly; the job is idempotent, so rerun to resume.\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  transcode --limit 50           # sizing report over a sample\n")
		fmt.Fprintf(os.Stderr, "  transcode --apply              # do it\n")
		fmt.Fprintf(os.Stderr, "  transcode --file bird.wav      # local check, no DB or R2\n\nFlags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if *file != "" {
		if err := runFile(ctx, os.Stdout, *file); err != nil {
			log.Fatalf("transcode %s: %v", *file, err)
		}
		return
	}

	pool, err := pgxpool.New(ctx, mustEnv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	r2c, err := r2.New(
		mustEnv("R2_ACCOUNT_ID"),
		mustEnv("R2_ACCESS_KEY_ID"),
		mustEnv("R2_SECRET_ACCESS_KEY"),
		mustEnv("R2_BUCKET_NAME"),
		mustEnv("R2_PUBLIC_URL"),
	)
	if err != nil {
		log.Fatalf("r2 client: %v", err)
	}

	opts := options{apply: *apply, limit: *limit, workers: *workers}
	if *speciesFilter != "" {
		opts.species = strings.Split(*speciesFilter, ",")
	}
	if !opts.apply {
		fmt.Fprintln(os.Stderr, "DRY RUN: nothing will be uploaded or written. Pass --apply to commit.")
	}

	rep, err := sweep(ctx, os.Stdout, store.New(pool), r2c, httpFetcher, opts)
	if err != nil {
		log.Fatalf("sweep: %v", err)
	}
	if len(rep.Failures) > 0 {
		os.Exit(1)
	}
}

// runFile transcodes a local file next to itself. This is how the encode is
// exercised by hand: ingest must never be run to test audio work, because media
// goes to the one shared R2 bucket regardless of which DB is configured.
func runFile(ctx context.Context, w io.Writer, path string) error {
	info, err := audio.Probe(ctx, path)
	if err != nil {
		return err
	}
	res, err := audio.Transcode(ctx, path)
	if err != nil {
		return err
	}

	out := path + ".transcoded.mp3"
	if err := os.WriteFile(out, res.Data, 0o600); err != nil {
		return err
	}

	src, err := os.Stat(path)
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "in:    %s  %s %dch %d bps %.1fs\n", path, info.Format, info.Channels, info.BitRate, info.Duration)
	fmt.Fprintf(w, "out:   %s  mp3 mono 96k\n", out)
	fmt.Fprintf(w, "size:  %s -> %s\n", humanBytes(src.Size()), humanBytes(int64(len(res.Data))))
	fmt.Fprintf(w, "peaks: %d buckets\n", len(res.Peaks))
	return nil
}

// httpFetcher downloads url to a temp file. R2 objects are served publicly over
// https, so no signing is needed to read them.
func httpFetcher(ctx context.Context, url string) (string, int64, func(), error) {
	noop := func() {}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", 0, noop, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", 0, noop, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", 0, noop, fmt.Errorf("fetch %s: status %d", url, resp.StatusCode)
	}

	f, err := os.CreateTemp("", "flockdeck-object-")
	if err != nil {
		return "", 0, noop, err
	}
	size, err := io.Copy(f, resp.Body)
	f.Close()
	cleanup := func() { os.Remove(f.Name()) } //nolint:errcheck
	if err != nil {
		cleanup()
		return "", 0, noop, err
	}
	return f.Name(), size, cleanup, nil
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env var %s is not set", key)
	}
	return v
}
