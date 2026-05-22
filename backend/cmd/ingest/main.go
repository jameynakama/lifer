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
	flag.Parse()

	regions := flag.Args()
	if len(regions) == 0 {
		log.Fatal("usage: ingest [flags] <region-code> [region-code...]")
	}

	ebirdKey := mustEnv("EBIRD_API_KEY")
	xcKey := os.Getenv("XENO_CANTO_API_KEY")
	assetsDir := mustEnv("ASSETS_DIR")
	dbURL := mustEnv("DATABASE_URL")

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

	sem := make(chan struct{}, *workers)
	var wg sync.WaitGroup
	var mu sync.Mutex
	done := 0
	total := len(codes)

	for _, code := range codes {
		entry, ok := taxMap[code]
		if !ok {
			log.Printf("warn: no taxonomy entry for %s, skipping", code)
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(code string, entry ebird.TaxonomyEntry) {
			defer wg.Done()
			defer func() { <-sem }()
			if err := ingestSpecies(ctx, q, xcClient, macaulayClient, entry, *maxRecordings, *maxImages, assetsDir); err != nil {
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
}

func ingestSpecies(
	ctx context.Context,
	q *store.Queries,
	xc *xenocanto.Client,
	mac *macaulay.Client,
	entry ebird.TaxonomyEntry,
	maxRec, maxImg int,
	assetsDir string,
) error {
	parts := strings.SplitN(entry.SciName, " ", 2)
	if len(parts) != 2 {
		return fmt.Errorf("unexpected sciName %q", entry.SciName)
	}
	genus, species := parts[0], parts[1]

	sp, err := q.UpsertSpecies(ctx, store.UpsertSpeciesParams{
		CommonName:     entry.CommonName,
		ScientificName: entry.SciName,
		EbirdCode:      entry.SpeciesCode,
	})
	if err != nil {
		return fmt.Errorf("upsert species: %w", err)
	}

	perType := maxRec / 2 // integer division; --max-recordings 3 gives 1 per type
	for _, recType := range []string{"song", "call"} {
		recs, err := xc.Search(ctx, genus, species, recType)
		if err != nil {
			log.Printf("  warn: xeno-canto %s %s: %v", entry.SpeciesCode, recType, err)
			continue
		}
		if len(recs) > perType {
			recs = recs[:perType]
		}
		for _, rec := range recs {
			destPath := filepath.Join(assetsDir, "recordings", entry.SpeciesCode, rec.ID+".mp3")
			if err := downloadFile(ctx, rec.FileURL, destPath); err != nil {
				log.Printf("  warn: download recording %s: %v", rec.ID, err)
				continue
			}
			if _, err := q.UpsertRecording(ctx, store.UpsertRecordingParams{
				SpeciesID:   sp.ID,
				XenoCantoID: rec.ID,
				FilePath:    filepath.Join("recordings", entry.SpeciesCode, rec.ID+".mp3"),
				Quality:     rec.Quality,
				Type:        rec.Type,
			}); err != nil {
				log.Printf("  warn: upsert recording %s: %v", rec.ID, err)
			}
		}
	}

	photos, err := mac.Photos(ctx, entry.SpeciesCode, maxImg)
	if err != nil {
		log.Printf("  warn: macaulay %s: %v", entry.SpeciesCode, err)
		return nil // photos are optional; don't fail the species on image errors
	}
	for _, photo := range photos {
		destPath := filepath.Join(assetsDir, "images", entry.SpeciesCode, photo.AssetID+".jpg")
		if err := downloadFile(ctx, mac.PhotoURL(photo.AssetID), destPath); err != nil {
			log.Printf("  warn: download image %s: %v", photo.AssetID, err)
			continue
		}
		if _, err := q.UpsertSpeciesImage(ctx, store.UpsertSpeciesImageParams{
			SpeciesID:  sp.ID,
			MacaulayID: photo.AssetID,
			FilePath:   filepath.Join("images", entry.SpeciesCode, photo.AssetID+".jpg"),
			Credit:     photo.UserDisplayName,
		}); err != nil {
			log.Printf("  warn: upsert image %s: %v", photo.AssetID, err)
		}
	}
	return nil
}

func downloadFile(ctx context.Context, rawURL, destPath string) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: status %d", rawURL, resp.StatusCode)
	}
	tmp, err := os.CreateTemp(filepath.Dir(destPath), ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	copyErr := func() error {
		_, err := io.Copy(tmp, resp.Body)
		return err
	}()
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

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env var %s is not set", key)
	}
	return v
}
