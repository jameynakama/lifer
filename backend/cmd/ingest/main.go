package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jameynakama/lifer/internal/ebird"
	"github.com/jameynakama/lifer/internal/macaulay"
	"github.com/jameynakama/lifer/internal/store"
	"github.com/jameynakama/lifer/internal/xenocanto"
)

func main() {
	maxRecordings := flag.Int("max-recordings", 4, "max recordings per species (split evenly between song and call)")
	maxImages := flag.Int("max-images", 3, "max images per species")
	workers := flag.Int("workers", 5, "concurrent worker count")
	skipComplete := flag.Bool("skip-complete", false, "skip species that already have ≥1 recording and ≥1 image in the DB")
	skipMedia := flag.Bool("skip-media", false, "store external URLs instead of downloading files (ASSETS_DIR not required)")
	speciesFilter := flag.String("species", "", "comma-separated ebird codes or common names to process (e.g. busti or \"Bushtit,American Robin\")")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: ingest [flags] <region-code> [region-code...]\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  ingest US-OR\n")
		fmt.Fprintf(os.Stderr, "  ingest US-OR US-WA US-ID\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	regions := flag.Args()
	if len(regions) == 0 {
		log.Fatal("usage: ingest [flags] <region-code> [region-code...]")
	}

	ebirdKey := mustEnv("EBIRD_API_KEY")
	xcKey := mustEnv("XENO_CANTO_API_KEY")
	dbURL := mustEnv("DATABASE_URL")
	var assetsDir string
	if !*skipMedia {
		assetsDir = mustEnv("ASSETS_DIR")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	q := store.New(pool)
	ebirdClient := ebird.New(ebirdKey)
	xcClient := xenocanto.New(xcKey)
	macaulayClient := macaulay.New(ebirdKey)

	log.Println("fetching eBird taxonomy...")
	taxonomy, err := ebirdClient.Taxonomy(ctx)
	if err != nil {
		log.Fatalf("fetch taxonomy: %v", err)
	}
	taxMap := make(map[string]ebird.TaxonomyEntry, len(taxonomy))
	for _, t := range taxonomy {
		taxMap[t.SpeciesCode] = t
	}
	log.Printf("taxonomy loaded: %d entries", len(taxMap))

	seen := make(map[string]struct{})
	var codes []string
	for _, region := range regions {
		list, err := ebirdClient.SpeciesList(ctx, region)
		if err != nil {
			log.Printf("warn: region %s: %v", region, err)
			continue
		}
		for _, code := range list {
			if _, ok := seen[code]; !ok {
				seen[code] = struct{}{}
				codes = append(codes, code)
			}
		}
		log.Printf("region %s: %d species", region, len(list))
	}
	log.Printf("total unique species: %d", len(codes))

	if *speciesFilter != "" {
		want := strings.Split(*speciesFilter, ",")
		before := len(codes)
		codes = filterBySpecies(codes, taxMap, want)
		log.Printf("--species: filtered to %d/%d species", len(codes), before)
	}

	if *skipComplete {
		completeCodes, err := q.ListCompleteSpeciesEbirdCodes(ctx)
		if err != nil {
			log.Fatalf("skip-complete: query complete species: %v", err)
		}
		complete := make(map[string]struct{}, len(completeCodes))
		for _, c := range completeCodes {
			complete[c] = struct{}{}
		}
		before := len(codes)
		codes = filterComplete(codes, complete)
		log.Printf("--skip-complete: skipping %d already-complete species, processing %d remaining", before-len(codes), len(codes))
	}

	sem := make(chan struct{}, *workers)
	var wg sync.WaitGroup
	var mu sync.Mutex
	started, done := 0, 0
	total := len(codes)

	for _, code := range codes {
		entry, ok := taxMap[code]
		if !ok {
			log.Printf("warn: no taxonomy entry for %s, skipping", code)
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		mu.Lock()
		started++
		n := started
		mu.Unlock()
		log.Printf("starting %d/%d: %s", n, total, entry.CommonName)
		go func(code string, entry ebird.TaxonomyEntry) {
			defer wg.Done()
			defer func() { <-sem }()
			if err := ingestSpecies(ctx, q, xcClient, macaulayClient, entry, *maxRecordings, *maxImages, assetsDir, *skipMedia); err != nil {
				log.Printf("error %s (%s): %v", entry.CommonName, code, err)
			}
			mu.Lock()
			done++
			n := done
			mu.Unlock()
			log.Printf("done %d/%d: %s", n, total, entry.CommonName)
		}(code, entry)
	}
	wg.Wait()
	skipped := total - done
	if skipped > 0 {
		log.Printf("ingestion complete: %d/%d species (%d skipped -- no taxonomy entry)", done, total, skipped)
	} else {
		log.Printf("ingestion complete: %d/%d species", done, total)
	}

	log.Println("cleaning up species missing recordings or images...")
	incomplete, err := q.ListIncompleteSpecies(ctx)
	if err != nil {
		log.Fatalf("cleanup: %v", err)
	}
	for _, sp := range incomplete {
		if !*skipMedia {
			os.RemoveAll(filepath.Join(assetsDir, "images", sp.EbirdCode))
			os.RemoveAll(filepath.Join(assetsDir, "recordings", sp.EbirdCode))
		}
		if err := q.DeleteRecordingsBySpeciesID(ctx, sp.ID); err != nil {
			log.Printf("  warn: cleanup recordings %s: %v", sp.EbirdCode, err)
		}
		if err := q.DeleteSpeciesImagesBySpeciesID(ctx, sp.ID); err != nil {
			log.Printf("  warn: cleanup images %s: %v", sp.EbirdCode, err)
		}
		if err := q.DeleteSpeciesByID(ctx, sp.ID); err != nil {
			log.Printf("  warn: cleanup species %s: %v", sp.EbirdCode, err)
		}
	}
	log.Printf("cleanup: removed %d incomplete species", len(incomplete))
}

func filterBySpecies(codes []string, taxMap map[string]ebird.TaxonomyEntry, want []string) []string {
	if len(want) == 0 {
		return []string{}
	}
	// Build a set of matching ebird codes from want, accepting either codes or common names.
	match := make(map[string]struct{}, len(want))
	for _, w := range want {
		lower := strings.ToLower(w)
		if _, ok := taxMap[lower]; ok {
			match[lower] = struct{}{}
			continue
		}
		for code, entry := range taxMap {
			if strings.ToLower(entry.CommonName) == lower {
				match[code] = struct{}{}
				break
			}
		}
	}
	out := make([]string, 0, len(match))
	for _, c := range codes {
		if _, ok := match[c]; ok {
			out = append(out, c)
		}
	}
	return out
}

func filterComplete(codes []string, complete map[string]struct{}) []string {
	out := make([]string, 0, len(codes))
	for _, c := range codes {
		if _, ok := complete[c]; !ok {
			out = append(out, c)
		}
	}
	return out
}

func ingestSpecies(
	ctx context.Context,
	q *store.Queries,
	xc *xenocanto.Client,
	mac *macaulay.Client,
	entry ebird.TaxonomyEntry,
	maxRec, maxImg int,
	assetsDir string,
	skipMedia bool,
) error {
	sp, err := q.UpsertSpecies(ctx, store.UpsertSpeciesParams{
		CommonName:     entry.CommonName,
		ScientificName: entry.SciName,
		EbirdCode:      entry.SpeciesCode,
	})
	if err != nil {
		return fmt.Errorf("upsert species: %w", err)
	}

	perType := maxRec / 2 // integer division; --max-recordings 3 gives 1 per type

	// Fan out song + call searches concurrently.
	type searchResult struct {
		recType string
		recs    []xenocanto.Recording
		err     error
	}
	searchCh := make(chan searchResult, 2)
	for _, recType := range []string{"song", "call"} {
		go func(rt string) {
			recs, err := xc.Search(ctx, entry.CommonName, rt)
			searchCh <- searchResult{rt, recs, err}
		}(recType)
	}

	// Collect search results and fan out downloads.
	var recWg sync.WaitGroup
	for range 2 {
		result := <-searchCh
		if result.err != nil {
			log.Printf("  warn: xeno-canto %s %s: %v", entry.SpeciesCode, result.recType, result.err)
			continue
		}
		recs := result.recs
		if len(recs) > perType {
			recs = recs[:perType]
		}
		for _, rec := range recs {
			recWg.Add(1)
			go func(rec xenocanto.Recording) {
				defer recWg.Done()
				var filePath string
				if skipMedia {
					filePath = rec.FileURL
				} else {
					destPath := filepath.Join(assetsDir, "recordings", entry.SpeciesCode, rec.ID+".mp3")
					if err := downloadFile(ctx, rec.FileURL, destPath); err != nil {
						log.Printf("  warn: download recording %s: %v", rec.ID, err)
						return
					}
					filePath = filepath.Join("recordings", entry.SpeciesCode, rec.ID+".mp3")
				}
				if _, err := q.UpsertRecording(ctx, store.UpsertRecordingParams{
					SpeciesID:   sp.ID,
					XenoCantoID: rec.ID,
					FilePath:    filePath,
					Quality:     rec.Quality,
					Type:        rec.Type,
				}); err != nil {
					log.Printf("  warn: upsert recording %s: %v", rec.ID, err)
				}
			}(rec)
		}
	}
	recWg.Wait()

	photos, err := mac.Photos(ctx, entry.SpeciesCode, maxImg)
	if err != nil {
		log.Printf("  warn: macaulay %s: %v", entry.SpeciesCode, err)
		return nil // photos are optional; don't fail the species on image errors
	}

	// Fan out image downloads concurrently.
	var imgWg sync.WaitGroup
	for _, photo := range photos {
		imgWg.Add(1)
		go func(photo macaulay.Photo) {
			defer imgWg.Done()
			var filePath string
			if skipMedia {
				filePath = mac.PhotoURL(photo.AssetID)
			} else {
				destPath := filepath.Join(assetsDir, "images", entry.SpeciesCode, photo.AssetID+".jpg")
				if err := downloadFile(ctx, mac.PhotoURL(photo.AssetID), destPath); err != nil {
					log.Printf("  warn: download image %s: %v", photo.AssetID, err)
					return
				}
				filePath = filepath.Join("images", entry.SpeciesCode, photo.AssetID+".jpg")
			}
			if _, err := q.UpsertSpeciesImage(ctx, store.UpsertSpeciesImageParams{
				SpeciesID:  sp.ID,
				MacaulayID: photo.AssetID,
				FilePath:   filePath,
				Credit:     photo.UserDisplayName,
			}); err != nil {
				log.Printf("  warn: upsert image %s: %v", photo.AssetID, err)
			}
		}(photo)
	}
	imgWg.Wait()
	return nil
}

// retryDelays controls the wait between attempts on a 429 response.
// Overridable in tests.
var retryDelays = []time.Duration{2 * time.Second, 4 * time.Second, 8 * time.Second}

func downloadFile(ctx context.Context, rawURL, destPath string) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return err
	}
	var lastErr error
	for attempt := range len(retryDelays) + 1 {
		if attempt > 0 {
			select {
			case <-time.After(retryDelays[attempt-1]):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
		if err != nil {
			return err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			lastErr = fmt.Errorf("download %s: status 429", rawURL)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return fmt.Errorf("download %s: status %d", rawURL, resp.StatusCode)
		}
		tmp, err := os.CreateTemp(filepath.Dir(destPath), ".tmp-*")
		if err != nil {
			resp.Body.Close()
			return err
		}
		tmpName := tmp.Name()
		copyErr := func() error {
			_, err := io.Copy(tmp, resp.Body)
			return err
		}()
		resp.Body.Close()
		closeErr := tmp.Close()
		if copyErr != nil || closeErr != nil {
			os.Remove(tmpName)
			if copyErr != nil {
				return copyErr
			}
			return closeErr
		}
		return os.Rename(tmpName, destPath)
	}
	return lastErr
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env var %s is not set", key)
	}
	return v
}
