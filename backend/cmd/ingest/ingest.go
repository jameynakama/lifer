package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/jameynakama/flockdeck/internal/ebird"
	"github.com/jameynakama/flockdeck/internal/macaulay"
	"github.com/jameynakama/flockdeck/internal/r2"
	"github.com/jameynakama/flockdeck/internal/store"
	"github.com/jameynakama/flockdeck/internal/xenocanto"
)

type ingestStats struct {
	failures   []string
	recordings int
	images     int
}

// retryDelays controls the wait between attempts on a 429 response.
// Overridable in tests.
var retryDelays = []time.Duration{2 * time.Second, 4 * time.Second, 8 * time.Second}

// ingestSpecies fetches and uploads media for one species.
// xcOverrides maps ebird codes to [genus, species] pairs for xeno-canto taxonomy overrides.
func ingestSpecies(
	ctx context.Context,
	q *store.Queries,
	xc *xenocanto.Client,
	mac *macaulay.Client,
	entry ebird.TaxonomyEntry,
	maxRec, maxImg, maxLenSecs int,
	r2c *r2.Client,
	xcOverrides map[string][2]string,
	workerID int,
	send func(any),
) (stats ingestStats, err error) {
	defer func() {
		send(speciesDoneMsg{
			workerID:   workerID,
			code:       entry.SpeciesCode,
			name:       entry.CommonName,
			recordings: stats.recordings,
			images:     stats.images,
			failures:   stats.failures,
		})
	}()

	send(speciesStartedMsg{workerID: workerID, code: entry.SpeciesCode, name: entry.CommonName})

	sp, err := q.UpsertSpecies(ctx, store.UpsertSpeciesParams{
		EbirdCode:      entry.SpeciesCode,
		CommonName:     entry.CommonName,
		ScientificName: entry.SciName,
	})
	if err != nil {
		err = fmt.Errorf("upsert species: %w", err)
		return
	}

	xcGenus, xcSpecies := xcGenSp(entry.SpeciesCode, entry.SciName, xcOverrides)

	type searchResult struct {
		recType string
		recs    []xenocanto.Recording
		err     error
	}
	searchCh := make(chan searchResult, 2)
	for _, recType := range []string{"song", "call"} {
		go func(rt string) {
			recs, err := xc.Search(ctx, xcGenus, xcSpecies, rt, maxLenSecs)
			searchCh <- searchResult{rt, recs, err}
		}(recType)
	}

	var (
		recWg   sync.WaitGroup
		statsMu sync.Mutex
	)
	// Limit concurrent downloads per species to avoid 503s from xeno-canto CDN.
	dlSem := make(chan struct{}, 2)

	recordFailure := func(reason string) {
		statsMu.Lock()
		stats.failures = append(stats.failures, reason)
		statsMu.Unlock()
	}

	// Collect both search results before uploading so we can fill the budget
	// from whichever type has more recordings (e.g. waterfowl have calls but no songs).
	var songs, calls []xenocanto.Recording
	for range 2 {
		result := <-searchCh
		if result.err != nil {
			recordFailure(fmt.Sprintf("xeno-canto search %s %s: %v", entry.SpeciesCode, result.recType, result.err))
			continue
		}
		if result.recType == "song" {
			songs = result.recs
		} else {
			calls = result.recs
		}
	}
	toUpload := interleaveRecordings(songs, calls, maxRec)

	for _, rec := range toUpload {
		recWg.Add(1)
		go func(rec xenocanto.Recording) {
			defer recWg.Done()
			dlSem <- struct{}{}
			defer func() { <-dlSem }()
			key := "recordings/" + sp.EbirdCode + "/" + rec.ID + ".mp3"
			filePath, err := fetchAndUpload(ctx, r2c, rec.FileURL, key, "audio/mpeg", workerID, send)
			if err != nil {
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
				return
			}
			statsMu.Lock()
			stats.recordings++
			statsMu.Unlock()
		}(rec)
	}
	recWg.Wait()

	photos, err := mac.Photos(ctx, entry.SpeciesCode, maxImg)
	if err != nil {
		err = nil // non-fatal: species will show 0 images in missingMedia report
		return
	}

	var imgWg sync.WaitGroup
	for _, photo := range photos {
		imgWg.Add(1)
		go func(photo macaulay.Photo) {
			defer imgWg.Done()
			dlSem <- struct{}{}
			defer func() { <-dlSem }()
			key := "images/" + sp.EbirdCode + "/" + photo.AssetID + ".jpg"
			filePath, err := fetchAndUpload(ctx, r2c, mac.PhotoURL(photo.AssetID), key, "image/jpeg", workerID, send)
			if err != nil {
				recordFailure(fmt.Sprintf("image %s: %v", photo.AssetID, err))
				return
			}
			if _, err := q.UpsertSpeciesImage(ctx, store.UpsertSpeciesImageParams{
				MacaulayID:  photo.AssetID,
				SpeciesCode: sp.EbirdCode,
				FilePath:    filePath,
				Credit:      photo.UserDisplayName,
			}); err != nil {
				return
			}
			statsMu.Lock()
			stats.images++
			statsMu.Unlock()
		}(photo)
	}
	imgWg.Wait()
	return
}

// xcGenSp returns the genus and species to use for a xeno-canto query.
func xcGenSp(ebirdCode, sciName string, xcOverrides map[string][2]string) (genus, species string) {
	if override, ok := xcOverrides[ebirdCode]; ok {
		return override[0], override[1]
	}
	parts := strings.Fields(sciName)
	if len(parts) >= 2 {
		return parts[0], parts[1]
	}
	return sciName, ""
}

// parseXCOverrides parses "--xc-override comrav=Corvus:corax,calsja=Aphelocoma:californica"
func parseXCOverrides(s string) (map[string][2]string, error) {
	out := make(map[string][2]string)
	if s == "" {
		return out, nil
	}
	for _, entry := range strings.Split(s, ",") {
		kv := strings.SplitN(entry, "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("expected code=Genus:species, got %q", entry)
		}
		genSp := strings.SplitN(kv[1], ":", 2)
		if len(genSp) != 2 {
			return nil, fmt.Errorf("expected Genus:species after =, got %q", kv[1])
		}
		out[kv[0]] = [2]string{genSp[0], genSp[1]}
	}
	return out, nil
}

// fetchAndUpload GETs from sourceURL and uploads the body to R2 at key.
// Sends fetchStartedMsg, uploadStartedMsg, and uploadDoneMsg via send.
// Retries on 429 and 503 from the source. Returns the full public R2 URL.
func fetchAndUpload(ctx context.Context, r2c *r2.Client, sourceURL, key, contentType string, workerID int, send func(any)) (string, error) {
	send(fetchStartedMsg{workerID: workerID, key: key})
	var lastErr error
	for attempt := range len(retryDelays) + 1 {
		if attempt > 0 {
			select {
			case <-time.After(retryDelays[attempt-1]):
			case <-ctx.Done():
				err := ctx.Err()
				send(uploadDoneMsg{workerID: workerID, key: key, err: err})
				return "", err
			}
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
		if err != nil {
			send(uploadDoneMsg{workerID: workerID, key: key, err: err})
			return "", err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			send(uploadDoneMsg{workerID: workerID, key: key, err: err})
			return "", err
		}
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable {
			resp.Body.Close()
			lastErr = fmt.Errorf("fetch %s: status %d", sourceURL, resp.StatusCode)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			err := fmt.Errorf("fetch %s: status %d", sourceURL, resp.StatusCode)
			send(uploadDoneMsg{workerID: workerID, key: key, err: err})
			return "", err
		}
		send(uploadStartedMsg{workerID: workerID, key: key})
		url, err := r2c.Upload(ctx, key, contentType, resp.Body)
		resp.Body.Close()
		if err != nil {
			send(uploadDoneMsg{workerID: workerID, key: key, err: err})
			return "", err
		}
		send(uploadDoneMsg{workerID: workerID, key: key, err: nil})
		return url, nil
	}
	send(uploadDoneMsg{workerID: workerID, key: key, err: lastErr})
	return "", lastErr
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

func filterArbitrary(codes []string, toSkip map[string]struct{}) []string {
	out := make([]string, 0, len(codes))
	for _, c := range codes {
		if _, ok := toSkip[c]; !ok {
			out = append(out, c)
		}
	}
	return out
}

// interleaveRecordings picks up to n recordings from songs and calls alternately,
// so species with only one type still fill their full quota.
func interleaveRecordings(songs, calls []xenocanto.Recording, n int) []xenocanto.Recording {
	out := make([]xenocanto.Recording, 0, n)
	for i := 0; len(out) < n && (i < len(songs) || i < len(calls)); i++ {
		if i < len(songs) && len(out) < n {
			out = append(out, songs[i])
		}
		if i < len(calls) && len(out) < n {
			out = append(out, calls[i])
		}
	}
	return out
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env var %s is not set", key)
	}
	return v
}
