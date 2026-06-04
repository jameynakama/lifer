package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jameynakama/flockdeck/internal/ebird"
	"github.com/jameynakama/flockdeck/internal/macaulay"
	"github.com/jameynakama/flockdeck/internal/r2"
	"github.com/jameynakama/flockdeck/internal/store"
	"github.com/jameynakama/flockdeck/internal/xenocanto"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "Show how many species will be ingested")
	maxRecordings := flag.Int("max-recordings", 4, "max recordings per species (split evenly between song and call)")
	maxImages := flag.Int("max-images", 3, "max images per species")
	maxRecordingSecs := flag.Int("max-recording-secs", 180, "max recording length in seconds (0 = no cap)")
	workers := flag.Int("workers", 5, "concurrent worker count")
	skipComplete := flag.Bool("skip-complete", false, "skip species that already have ≥1 recording and ≥1 image in the DB")
	speciesFilter := flag.String("species", "", "comma-separated ebird codes or common names to process")
	xcOverrideFlag := flag.String("xc-override", "", "comma-separated xeno-canto taxonomy overrides, e.g. \"comrav=Corvus:corax\"")
	noR2 := flag.Bool("no-r2", false, "skip R2 uploads; store placeholder:// URLs in DB (local dev/testing only -- never use in prod)")
	noCleanup := flag.Bool("no-cleanup", false, "skip post-run cleanup of incomplete species from R2 and DB (safe for local dev)")
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

	fileOverrides, err := loadOverrides("cmd/ingest/overrides.txt")
	if err != nil {
		fmt.Fprintf(os.Stderr, "overrides.txt: %v\n", err)
		os.Exit(1)
	}
	// Merge: file is baseline, flag wins on conflict.
	for k, v := range xcOverrides {
		fileOverrides[k] = v
	}
	xcOverrides = fileOverrides
	if len(xcOverrides) > 0 {
		fmt.Fprintf(os.Stderr, "loaded %d XC overrides\n", len(xcOverrides))
	}

	ebirdKey := mustEnv("EBIRD_API_KEY")
	xcKey := mustEnv("XENO_CANTO_API_KEY")
	dbURL := mustEnv("DATABASE_URL")

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "db connect: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	var r2c *r2.Client
	if !*noR2 {
		r2AccountID := mustEnv("R2_ACCOUNT_ID")
		r2AccessKey := mustEnv("R2_ACCESS_KEY_ID")
		r2SecretKey := mustEnv("R2_SECRET_ACCESS_KEY")
		r2Bucket := mustEnv("R2_BUCKET_NAME")
		r2PubURL := mustEnv("R2_PUBLIC_URL")
		r2c, err = r2.New(r2AccountID, r2AccessKey, r2SecretKey, r2Bucket, r2PubURL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "r2 client: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Fprintln(os.Stderr, "--no-r2: skipping R2 uploads, placeholder URLs will be stored in DB")
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
		codes = filterArbitrary(codes, complete)
		fmt.Fprintf(os.Stderr, "--skip-complete: skipping %d already-complete species, processing %d remaining\n", before-len(codes), len(codes))
	}

	alwaysSkip := loadManualSkips("cmd/ingest/skip.txt")
	if len(alwaysSkip) > 0 {
		before := len(codes)
		codes = filterArbitrary(codes, alwaysSkip)
		fmt.Fprintf(os.Stderr, "skipping %d species per skip.txt, processing %d remaining\n", before-len(codes), len(codes))
	}

	bannedImages := loadManualSkips("cmd/ingest/banned_images.txt")
	if len(bannedImages) > 0 {
		fmt.Fprintf(os.Stderr, "loaded %d banned image IDs from banned_images.txt\n", len(bannedImages))
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
	fmt.Fprintf(os.Stderr, "total unique species to process: %d\n\n", total)

	if *dryRun {
		dryRunExit(os.Stderr, os.Exit)
	}

	failedSpecies := map[string][]string{}
	missingMedia := map[string]ingestStats{}

	slots := make(chan int, *workers)
	for i := range *workers {
		slots <- i
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	m := newModel(total, *workers)
	p := tea.NewProgram(m)
	send := func(msg any) { p.Send(msg) }

	go func() {
		for _, ce := range processable {
			workerID := <-slots
			wg.Add(1)
			go func(ce codeEntry, workerID int) {
				defer wg.Done()
				defer func() { slots <- workerID }()
				stats, err := ingestSpecies(ctx, q, xcClient, macaulayClient, ce.entry, *maxRecordings, *maxImages, *maxRecordingSecs, r2c, xcOverrides, bannedImages, workerID, send)
				mu.Lock()
				if err != nil {
					failedSpecies[ce.code] = append(stats.failures, fmt.Sprintf("ingest: %v", err))
				} else if len(stats.failures) > 0 {
					failedSpecies[ce.code] = stats.failures
				} else if stats.recordings == 0 || stats.images == 0 {
					missingMedia[ce.code] = stats
				}
				mu.Unlock()
			}(ce, workerID)
		}
		wg.Wait()
		p.Send(allDoneMsg{})
	}()

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "tui: %v\n", err)
		os.Exit(1)
	}

	// --- post-run cleanup and reports ---

	// Species with locked media are never cleaned up: deleting the species
	// row cascades onto locked rows and DeletePrefix would wipe locked R2
	// files. They are reported instead (see PROTECTED section below).
	lockedCodes, err := q.ListSpeciesCodesWithLockedMedia(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "list locked media: %v\n", err)
		os.Exit(1)
	}
	lockedSet := make(map[string]struct{}, len(lockedCodes))
	for _, c := range lockedCodes {
		lockedSet[c] = struct{}{}
	}
	protectedSet := map[string]struct{}{}

	if !*noCleanup {
		fmt.Fprintln(os.Stderr, "cleaning up species missing recordings or images...")
		incomplete, err := q.ListIncompleteSpecies(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cleanup: %v\n", err)
			os.Exit(1)
		}
		deletable, protected := partitionProtected(incomplete, lockedSet)
		for _, c := range protected {
			protectedSet[c] = struct{}{}
		}
		cleanupSpecies(ctx, q, r2c, deletable)
		fmt.Fprintf(os.Stderr, "cleanup: removed %d incomplete species (%d protected by locked media)\n",
			len(deletable), len(protected))
	} else {
		fmt.Fprintln(os.Stderr, "--no-cleanup: skipping post-run cleanup")
	}

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
		deletable, protected := partitionProtected(failedCodes, lockedSet)
		for _, c := range protected {
			protectedSet[c] = struct{}{}
		}
		cleanupSpecies(ctx, q, r2c, deletable)
		fmt.Printf("re-run failed species with:\n")
		fmt.Printf("  just ingest --species %s <region>\n", strings.Join(failedCodes, ","))
	}

	if len(protectedSet) > 0 {
		protectedCodes := make([]string, 0, len(protectedSet))
		for code := range protectedSet {
			protectedCodes = append(protectedCodes, code)
		}
		sort.Strings(protectedCodes)
		fmt.Printf("\n=== PROTECTED (locked media -- left untouched) ===\n")
		for _, code := range protectedCodes {
			fmt.Printf("  %s (%s)\n", taxMap[code].CommonName, code)
		}
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

func dryRunExit(w io.Writer, exitFn func(int)) {
	fmt.Fprintln(w, "Dry run complete.")
	exitFn(0)
}

func loadOverrides(path string) (map[string][2]string, error) {
	out := map[string][2]string{}
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return out, nil
	}
	if err != nil {
		return nil, fmt.Errorf("loadOverrides: %w", err)
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	lineNum := 0
	for sc.Scan() {
		lineNum++
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.Fields(line)[0] // strip inline comment
		kv := strings.SplitN(line, "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("line %d: expected code=Genus:species, got %q", lineNum, line)
		}
		genSp := strings.SplitN(kv[1], ":", 2)
		if len(genSp) != 2 {
			return nil, fmt.Errorf("line %d: expected Genus:species after =, got %q", lineNum, kv[1])
		}
		out[kv[0]] = [2]string{genSp[0], genSp[1]}
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("loadOverrides: %w", err)
	}
	return out, nil
}

func loadManualSkips(path string) map[string]struct{} {
	skip := map[string]struct{}{}

	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return skip
	}
	if err != nil {
		log.Fatalf("getManualSkips: %v", err)
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		code := strings.Fields(line)[0]
		skip[code] = struct{}{}
	}
	if err := sc.Err(); err != nil {
		log.Fatalf("getManualSkips: %v", err)
	}

	return skip
}
