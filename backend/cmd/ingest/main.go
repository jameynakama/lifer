package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jameynakama/lifer/internal/ebird"
	"github.com/jameynakama/lifer/internal/macaulay"
	"github.com/jameynakama/lifer/internal/r2"
	"github.com/jameynakama/lifer/internal/store"
	"github.com/jameynakama/lifer/internal/xenocanto"
)

func main() {
	maxRecordings := flag.Int("max-recordings", 4, "max recordings per species (split evenly between song and call)")
	maxImages := flag.Int("max-images", 3, "max images per species")
	workers := flag.Int("workers", 5, "concurrent worker count")
	skipComplete := flag.Bool("skip-complete", false, "skip species that already have ≥1 recording and ≥1 image in the DB")
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
	r2AccountID := mustEnv("R2_ACCOUNT_ID")
	r2AccessKey := mustEnv("R2_ACCESS_KEY_ID")
	r2SecretKey := mustEnv("R2_SECRET_ACCESS_KEY")
	r2Bucket := mustEnv("R2_BUCKET_NAME")
	r2PubURL := mustEnv("R2_PUBLIC_URL")

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	r2c, err := r2.New(r2AccountID, r2AccessKey, r2SecretKey, r2Bucket, r2PubURL)
	if err != nil {
		log.Fatalf("r2 client: %v", err)
	}

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
	failedSpecies := map[string][]string{} // ebird_code → upload failure reasons

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
			failures, err := ingestSpecies(ctx, q, xcClient, macaulayClient, entry, *maxRecordings, *maxImages, r2c)
			if err != nil {
				log.Printf("error %s (%s): %v", entry.CommonName, code, err)
			}
			if len(failures) > 0 {
				mu.Lock()
				failedSpecies[code] = failures
				mu.Unlock()
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
	for _, code := range incomplete {
		if err := q.DeleteRecordingsBySpeciesCode(ctx, code); err != nil {
			log.Printf("  warn: cleanup recordings %s: %v", code, err)
		}
		if err := q.DeleteSpeciesImagesBySpeciesCode(ctx, code); err != nil {
			log.Printf("  warn: cleanup images %s: %v", code, err)
		}
		if err := q.DeleteSpeciesByCode(ctx, code); err != nil {
			log.Printf("  warn: cleanup species %s: %v", code, err)
		}
	}
	log.Printf("cleanup: removed %d incomplete species", len(incomplete))

	if len(failedSpecies) > 0 {
		failedCodes := make([]string, 0, len(failedSpecies))
		for code := range failedSpecies {
			failedCodes = append(failedCodes, code)
		}
		sort.Strings(failedCodes)

		log.Printf("\n=== PARTIAL UPLOAD FAILURES (%d species) ===", len(failedSpecies))
		for _, code := range failedCodes {
			name := taxMap[code].CommonName
			log.Printf("  %s (%s):", name, code)
			for _, reason := range failedSpecies[code] {
				log.Printf("    - %s", reason)
			}
		}
		log.Printf("cleaning up partial R2 uploads and DB entries for failed species...")
		for _, code := range failedCodes {
			for _, prefix := range []string{"recordings/" + code + "/", "images/" + code + "/"} {
				if err := r2c.DeletePrefix(ctx, prefix); err != nil {
					log.Printf("  warn: R2 delete %s: %v", prefix, err)
				}
			}
			if err := q.DeleteRecordingsBySpeciesCode(ctx, code); err != nil {
				log.Printf("  warn: DB delete recordings %s: %v", code, err)
			}
			if err := q.DeleteSpeciesImagesBySpeciesCode(ctx, code); err != nil {
				log.Printf("  warn: DB delete images %s: %v", code, err)
			}
			if err := q.DeleteSpeciesByCode(ctx, code); err != nil {
				log.Printf("  warn: DB delete species %s: %v", code, err)
			}
		}
		log.Printf("re-run failed species with:")
		log.Printf("  just ingest --species %s <region>", strings.Join(failedCodes, ","))
	}
}

func filterBySpecies(codes []string, taxMap map[string]ebird.TaxonomyEntry, want []string) []string {
	if len(want) == 0 {
		return []string{}
	}
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

// ingestSpecies fetches and uploads media for one species.
// Returns a list of upload failure reasons (non-empty = partial failure needing cleanup).
func ingestSpecies(
	ctx context.Context,
	q *store.Queries,
	xc *xenocanto.Client,
	mac *macaulay.Client,
	entry ebird.TaxonomyEntry,
	maxRec, maxImg int,
	r2c *r2.Client,
) ([]string, error) {
	sp, err := q.UpsertSpecies(ctx, store.UpsertSpeciesParams{
		EbirdCode:      entry.SpeciesCode,
		CommonName:     entry.CommonName,
		ScientificName: entry.SciName,
	})
	if err != nil {
		return nil, fmt.Errorf("upsert species: %w", err)
	}

	perType := maxRec / 2

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

	var (
		recWg    sync.WaitGroup
		failMu   sync.Mutex
		failures []string
	)

	recordFailure := func(reason string) {
		failMu.Lock()
		failures = append(failures, reason)
		failMu.Unlock()
	}

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
				key := "recordings/" + sp.EbirdCode + "/" + rec.ID + ".mp3"
				filePath, err := fetchAndUpload(ctx, r2c, rec.FileURL, key, "audio/mpeg")
				if err != nil {
					log.Printf("  warn: upload recording %s: %v", rec.ID, err)
					recordFailure(fmt.Sprintf("recording %s: %v", rec.ID, err))
					return
				}
				if _, err := q.UpsertRecording(ctx, store.UpsertRecordingParams{
					XenoCantoID: rec.ID,
					SpeciesCode: sp.EbirdCode,
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
		return failures, nil
	}

	var imgWg sync.WaitGroup
	for _, photo := range photos {
		imgWg.Add(1)
		go func(photo macaulay.Photo) {
			defer imgWg.Done()
			key := "images/" + sp.EbirdCode + "/" + photo.AssetID + ".jpg"
			filePath, err := fetchAndUpload(ctx, r2c, mac.PhotoURL(photo.AssetID), key, "image/jpeg")
			if err != nil {
				log.Printf("  warn: upload image %s: %v", photo.AssetID, err)
				recordFailure(fmt.Sprintf("image %s: %v", photo.AssetID, err))
				return
			}
			if _, err := q.UpsertSpeciesImage(ctx, store.UpsertSpeciesImageParams{
				MacaulayID:  photo.AssetID,
				SpeciesCode: sp.EbirdCode,
				FilePath:    filePath,
				Credit:      photo.UserDisplayName,
			}); err != nil {
				log.Printf("  warn: upsert image %s: %v", photo.AssetID, err)
			}
		}(photo)
	}
	imgWg.Wait()
	return failures, nil
}

// retryDelays controls the wait between attempts on a 429 response from the source CDN.
// Overridable in tests.
var retryDelays = []time.Duration{2 * time.Second, 4 * time.Second, 8 * time.Second}

// fetchAndUpload GETs from sourceURL and uploads the body to R2 at key.
// Retries on 429 from the source. Returns the full public R2 URL.
func fetchAndUpload(ctx context.Context, r2c *r2.Client, sourceURL, key, contentType string) (string, error) {
	var lastErr error
	for attempt := range len(retryDelays) + 1 {
		if attempt > 0 {
			select {
			case <-time.After(retryDelays[attempt-1]):
			case <-ctx.Done():
				return "", ctx.Err()
			}
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
		if err != nil {
			return "", err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return "", err
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			lastErr = fmt.Errorf("fetch %s: status 429", sourceURL)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return "", fmt.Errorf("fetch %s: status %d", sourceURL, resp.StatusCode)
		}
		url, err := r2c.Upload(ctx, key, contentType, resp.Body)
		resp.Body.Close()
		if err != nil {
			return "", err
		}
		return url, nil
	}
	return "", lastErr
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env var %s is not set", key)
	}
	return v
}
