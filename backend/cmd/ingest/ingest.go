package main

import (
	"context"
	"fmt"
	"io"
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

// ingestStore is the slice of store.Querier ingestSpecies needs.
type ingestStore interface {
	mediaUpserter
	UpsertSpecies(ctx context.Context, arg store.UpsertSpeciesParams) (store.Species, error)
}

// ingestSpecies fetches and uploads media for one species.
// xcOverrides maps ebird codes to [genus, species] pairs for xeno-canto taxonomy overrides.
func ingestSpecies(
	ctx context.Context,
	q ingestStore,
	xc *xenocanto.Client,
	mac *macaulay.Client,
	entry ebird.TaxonomyEntry,
	maxRec, maxImg, maxLenSecs int,
	r2c *r2.Client,
	xcOverrides map[string][2]string,
	bannedImages map[string]struct{},
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
			if err := uploadRecording(ctx, q, r2c, rec, sp.EbirdCode, workerID, send); err != nil {
				recordFailure(fmt.Sprintf("recording %s: %v", rec.ID, err))
				return
			}
			statsMu.Lock()
			stats.recordings++
			statsMu.Unlock()
		}(rec)
	}
	recWg.Wait()

	photos, err := mac.Photos(ctx, entry.SpeciesCode, maxImg+len(bannedImages))
	if err != nil {
		// Non-fatal (recordings may still have succeeded), but it must show up
		// in the failure report -- a Macaulay outage is not "0 images exist".
		recordFailure(fmt.Sprintf("macaulay photos %s: %v", entry.SpeciesCode, err))
		err = nil
		return
	}
	var filtered []macaulay.Photo
	for _, p := range photos {
		if _, banned := bannedImages[p.AssetID]; !banned {
			filtered = append(filtered, p)
		}
	}
	if len(filtered) > maxImg {
		filtered = filtered[:maxImg]
	}

	var imgWg sync.WaitGroup
	for _, photo := range filtered {
		imgWg.Add(1)
		go func(photo macaulay.Photo) {
			defer imgWg.Done()
			dlSem <- struct{}{}
			defer func() { <-dlSem }()
			if err := uploadImage(ctx, q, r2c, mac, photo, sp.EbirdCode, workerID, send); err != nil {
				recordFailure(fmt.Sprintf("image %s: %v", photo.AssetID, err))
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

// mediaUpserter is the slice of store.Querier the upload helpers need.
type mediaUpserter interface {
	UpsertRecording(ctx context.Context, arg store.UpsertRecordingParams) (store.SpeciesRecording, error)
	UpsertSpeciesImage(ctx context.Context, arg store.UpsertSpeciesImageParams) (store.SpeciesImage, error)
}

// uploadRecording ensures the recording's file is in R2 (or a placeholder when
// r2c is nil) and upserts its DB row. Any failure, including the DB write, is
// returned so the caller can record it.
func uploadRecording(ctx context.Context, q mediaUpserter, r2c *r2.Client, rec xenocanto.Recording, speciesCode string, workerID int, send func(any)) error {
	key := "recordings/" + speciesCode + "/" + rec.ID + ".mp3"
	filePath, err := ensureUploaded(ctx, r2c, rec.FileURL, key, "audio/mpeg", workerID, send)
	if err != nil {
		return err
	}
	if _, err := q.UpsertRecording(ctx, store.UpsertRecordingParams{
		XenoCantoID: rec.ID,
		SpeciesCode: speciesCode,
		FilePath:    filePath,
		Quality:     rec.Quality,
		Type:        rec.Type,
		Credit:      rec.Rec,
	}); err != nil {
		return fmt.Errorf("db upsert: %w", err)
	}
	return nil
}

// uploadImage is uploadRecording's image-lane counterpart.
func uploadImage(ctx context.Context, q mediaUpserter, r2c *r2.Client, mac *macaulay.Client, photo macaulay.Photo, speciesCode string, workerID int, send func(any)) error {
	key := "images/" + speciesCode + "/" + photo.AssetID + ".jpg"
	var sourceURL string
	if r2c != nil {
		sourceURL = mac.PhotoURL(photo.AssetID)
	}
	filePath, err := ensureUploaded(ctx, r2c, sourceURL, key, "image/jpeg", workerID, send)
	if err != nil {
		return err
	}
	if _, err := q.UpsertSpeciesImage(ctx, store.UpsertSpeciesImageParams{
		MacaulayID:  photo.AssetID,
		SpeciesCode: speciesCode,
		FilePath:    filePath,
		Credit:      photo.UserDisplayName,
	}); err != nil {
		return fmt.Errorf("db upsert: %w", err)
	}
	return nil
}

// ensureUploaded returns the public R2 URL for key, fetching and uploading the
// source if it isn't already there. With a nil R2 client it returns a
// placeholder URL (a damp run, if you will: drier than prod, wetter than --dry-run).
func ensureUploaded(ctx context.Context, r2c *r2.Client, sourceURL, key, contentType string, workerID int, send func(any)) (string, error) {
	if r2c == nil {
		return "placeholder://" + key, nil
	}
	exists, err := r2c.Exists(ctx, key)
	if err != nil {
		return "", err
	}
	if exists {
		return r2c.URL(key), nil
	}
	return fetchAndUpload(ctx, r2c, sourceURL, key, contentType, workerID, send)
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

// cleanupStore is the slice of store.Querier cleanup needs.
type cleanupStore interface {
	DeleteRecordingsBySpeciesCode(ctx context.Context, speciesCode string) error
	DeleteSpeciesImagesBySpeciesCode(ctx context.Context, speciesCode string) error
	DeleteSpeciesByCode(ctx context.Context, ebirdCode string) error
}

// prefixDeleter is the slice of *r2.Client cleanup needs. Pass a nil
// *interface* (not a nil *r2.Client in an interface) to skip R2 deletes.
type prefixDeleter interface {
	DeletePrefix(ctx context.Context, prefix string) error
}

// cleanupSpecies removes each species' R2 objects and DB rows, warning on w
// and continuing past individual failures. Callers must pre-filter
// locked-media species via partitionProtected.
func cleanupSpecies(ctx context.Context, w io.Writer, q cleanupStore, r2d prefixDeleter, codes []string) {
	for _, code := range codes {
		if r2d != nil {
			for _, prefix := range []string{"recordings/" + code + "/", "images/" + code + "/"} {
				if err := r2d.DeletePrefix(ctx, prefix); err != nil {
					fmt.Fprintf(w, "  warn: cleanup R2 %s: %v\n", prefix, err)
				}
			}
		}
		if err := q.DeleteRecordingsBySpeciesCode(ctx, code); err != nil {
			fmt.Fprintf(w, "  warn: cleanup recordings %s: %v\n", code, err)
		}
		if err := q.DeleteSpeciesImagesBySpeciesCode(ctx, code); err != nil {
			fmt.Fprintf(w, "  warn: cleanup images %s: %v\n", code, err)
		}
		if err := q.DeleteSpeciesByCode(ctx, code); err != nil {
			fmt.Fprintf(w, "  warn: cleanup species %s: %v\n", code, err)
		}
	}
}

// partitionProtected splits cleanup candidates into deletable codes and codes
// protected by locked media. Deleting a protected species would cascade onto
// its locked rows (and DeletePrefix would wipe its locked R2 files), so
// cleanup must skip the whole species.
func partitionProtected(codes []string, locked map[string]struct{}) (deletable, protected []string) {
	deletable = make([]string, 0, len(codes))
	protected = make([]string, 0)
	for _, c := range codes {
		if _, ok := locked[c]; ok {
			protected = append(protected, c)
		} else {
			deletable = append(deletable, c)
		}
	}
	return deletable, protected
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
