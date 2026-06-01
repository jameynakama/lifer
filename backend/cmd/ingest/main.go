package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"

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
	speciesFilter := flag.String("species", "", "comma-separated ebird codes or common names to process")
	xcOverrideFlag := flag.String("xc-override", "", "comma-separated xeno-canto taxonomy overrides, e.g. \"comrav=Corvus:corax\"")
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
		fmt.Fprintln(os.Stderr, "usage: ingest [flags] <region-code> [region-code...]")
		os.Exit(1)
	}

	xcOverrides, err := parseXCOverrides(*xcOverrideFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "--xc-override: %v\n", err)
		os.Exit(1)
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
		fmt.Fprintf(os.Stderr, "db connect: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	r2c, err := r2.New(r2AccountID, r2AccessKey, r2SecretKey, r2Bucket, r2PubURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "r2 client: %v\n", err)
		os.Exit(1)
	}

	q := store.New(pool)
	ebirdClient := ebird.New(ebirdKey)
	xcClient := xenocanto.New(xcKey)
	macaulayClient := macaulay.New(ebirdKey)

	fmt.Fprintln(os.Stderr, "fetching eBird taxonomy...")
	taxonomy, err := ebirdClient.Taxonomy(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fetch taxonomy: %v\n", err)
		os.Exit(1)
	}
	taxMap := make(map[string]ebird.TaxonomyEntry, len(taxonomy))
	for _, t := range taxonomy {
		taxMap[t.SpeciesCode] = t
	}
	fmt.Fprintf(os.Stderr, "taxonomy loaded: %d entries\n", len(taxMap))

	seen := make(map[string]struct{})
	var codes []string
	for _, region := range regions {
		list, err := ebirdClient.SpeciesList(ctx, region)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: region %s: %v\n", region, err)
			continue
		}
		for _, code := range list {
			if _, ok := seen[code]; !ok {
				seen[code] = struct{}{}
				codes = append(codes, code)
			}
		}
		fmt.Fprintf(os.Stderr, "region %s: %d species\n", region, len(list))
	}

	if *speciesFilter != "" {
		want := strings.Split(*speciesFilter, ",")
		before := len(codes)
		codes = filterBySpecies(codes, taxMap, want)
		fmt.Fprintf(os.Stderr, "--species: filtered to %d/%d species\n", len(codes), before)
	}

	if *skipComplete {
		completeCodes, err := q.ListCompleteSpeciesEbirdCodes(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skip-complete: query complete species: %v\n", err)
			os.Exit(1)
		}
		complete := make(map[string]struct{}, len(completeCodes))
		for _, c := range completeCodes {
			complete[c] = struct{}{}
		}
		before := len(codes)
		codes = filterComplete(codes, complete)
		fmt.Fprintf(os.Stderr, "--skip-complete: skipping %d already-complete species, processing %d remaining\n", before-len(codes), len(codes))
	}

	// Pre-filter to codes with taxonomy entries; warn for missing ones.
	type codeEntry struct {
		code  string
		entry ebird.TaxonomyEntry
	}
	var processable []codeEntry
	for _, code := range codes {
		if entry, ok := taxMap[code]; ok {
			processable = append(processable, codeEntry{code, entry})
		} else {
			fmt.Fprintf(os.Stderr, "warn: no taxonomy entry for %s, skipping\n", code)
		}
	}
	total := len(processable)
	fmt.Fprintf(os.Stderr, "total unique species to process: %d\n", total)

	// --- TUI placeholder: replaced in Task 6 ---
	failedSpecies := map[string][]string{}
	missingMedia := map[string]ingestStats{}

	slots := make(chan int, *workers)
	for i := range *workers {
		slots <- i
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	noop := func(any) {}

	for _, ce := range processable {
		workerID := <-slots
		wg.Add(1)
		go func(ce codeEntry, workerID int) {
			defer wg.Done()
			defer func() { slots <- workerID }()
			stats, err := ingestSpecies(ctx, q, xcClient, macaulayClient, ce.entry, *maxRecordings, *maxImages, r2c, xcOverrides, workerID, noop)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error %s (%s): %v\n", ce.entry.CommonName, ce.code, err)
			}
			mu.Lock()
			if len(stats.failures) > 0 {
				failedSpecies[ce.code] = stats.failures
			} else if err == nil && (stats.recordings == 0 || stats.images == 0) {
				missingMedia[ce.code] = stats
			}
			mu.Unlock()
		}(ce, workerID)
	}
	wg.Wait()

	fmt.Fprintf(os.Stderr, "ingestion complete: %d/%d species\n", total, total)

	// --- post-run cleanup and reports ---
	fmt.Fprintln(os.Stderr, "cleaning up species missing recordings or images...")
	incomplete, err := q.ListIncompleteSpecies(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cleanup: %v\n", err)
		os.Exit(1)
	}
	for _, code := range incomplete {
		for _, prefix := range []string{"recordings/" + code + "/", "images/" + code + "/"} {
			if err := r2c.DeletePrefix(ctx, prefix); err != nil {
				fmt.Fprintf(os.Stderr, "  warn: cleanup R2 %s: %v\n", prefix, err)
			}
		}
		if err := q.DeleteRecordingsBySpeciesCode(ctx, code); err != nil {
			fmt.Fprintf(os.Stderr, "  warn: cleanup recordings %s: %v\n", code, err)
		}
		if err := q.DeleteSpeciesImagesBySpeciesCode(ctx, code); err != nil {
			fmt.Fprintf(os.Stderr, "  warn: cleanup images %s: %v\n", code, err)
		}
		if err := q.DeleteSpeciesByCode(ctx, code); err != nil {
			fmt.Fprintf(os.Stderr, "  warn: cleanup species %s: %v\n", code, err)
		}
	}
	fmt.Fprintf(os.Stderr, "cleanup: removed %d incomplete species\n", len(incomplete))

	if len(failedSpecies) > 0 {
		failedCodes := make([]string, 0, len(failedSpecies))
		for code := range failedSpecies {
			failedCodes = append(failedCodes, code)
		}
		sort.Strings(failedCodes)
		fmt.Printf("\n=== PARTIAL UPLOAD FAILURES (%d species) ===\n", len(failedSpecies))
		for _, code := range failedCodes {
			name := taxMap[code].CommonName
			fmt.Printf("  %s (%s):\n", name, code)
			for _, reason := range failedSpecies[code] {
				fmt.Printf("    - %s\n", reason)
			}
		}
		fmt.Println("cleaning up partial R2 uploads and DB entries for failed species...")
		for _, code := range failedCodes {
			for _, prefix := range []string{"recordings/" + code + "/", "images/" + code + "/"} {
				if err := r2c.DeletePrefix(ctx, prefix); err != nil {
					fmt.Fprintf(os.Stderr, "  warn: R2 delete %s: %v\n", prefix, err)
				}
			}
			if err := q.DeleteRecordingsBySpeciesCode(ctx, code); err != nil {
				fmt.Fprintf(os.Stderr, "  warn: DB delete recordings %s: %v\n", code, err)
			}
			if err := q.DeleteSpeciesImagesBySpeciesCode(ctx, code); err != nil {
				fmt.Fprintf(os.Stderr, "  warn: DB delete images %s: %v\n", code, err)
			}
			if err := q.DeleteSpeciesByCode(ctx, code); err != nil {
				fmt.Fprintf(os.Stderr, "  warn: DB delete species %s: %v\n", code, err)
			}
		}
		fmt.Printf("re-run failed species with:\n")
		fmt.Printf("  just ingest --species %s <region>\n", strings.Join(failedCodes, ","))
	}

	if len(missingMedia) > 0 {
		missingCodes := make([]string, 0, len(missingMedia))
		for code := range missingMedia {
			missingCodes = append(missingCodes, code)
		}
		sort.Strings(missingCodes)
		fmt.Printf("\n=== MISSING MEDIA (%d species) ===\n", len(missingMedia))
		var xcMisses []string
		for _, code := range missingCodes {
			stats := missingMedia[code]
			name := taxMap[code].CommonName
			switch {
			case stats.recordings == 0 && stats.images == 0:
				fmt.Printf("  %s (%s): no recordings, no images\n", name, code)
				xcMisses = append(xcMisses, code)
			case stats.recordings == 0:
				fmt.Printf("  %s (%s): no recordings (xeno-canto miss -- check taxonomy)\n", name, code)
				xcMisses = append(xcMisses, code)
			case stats.images == 0:
				fmt.Printf("  %s (%s): no images (macaulay miss)\n", name, code)
			}
		}
		if len(xcMisses) > 0 {
			fmt.Println("for xeno-canto misses, research the species on xeno-canto.org then re-run:")
			fmt.Printf("  just ingest --xc-override \"<code>=Genus:species,...\" --skip-complete <region>\n")
			fmt.Printf("  missing codes: %s\n", strings.Join(xcMisses, ","))
		}
	}
}
