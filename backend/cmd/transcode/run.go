package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"slices"
	"sync"

	"github.com/jameynakama/flockdeck/internal/audio"
	"github.com/jameynakama/flockdeck/internal/store"
)

// recordingStore is the narrow slice of store.Querier this job needs. It is
// deliberately tiny: the job reads recordings and writes peaks, and must never
// be able to reach ingest's cleanup queries, which delete species rows DB-wide
// and cascade to cards, deck_species, preferences, and review_log.
type recordingStore interface {
	ListRecordingsForTranscode(ctx context.Context) ([]store.ListRecordingsForTranscodeRow, error)
	SetRecordingPeaks(ctx context.Context, arg store.SetRecordingPeaksParams) error
}

// objectStore is the slice of *r2.Client this job needs. Note the absence of
// any delete method.
type objectStore interface {
	KeyFor(fileURL string) string
	Upload(ctx context.Context, key, contentType string, body io.Reader) (string, error)
}

// fetcher downloads url to a local file and reports its size. The returned
// cleanup func must always be called.
type fetcher func(ctx context.Context, url string) (path string, size int64, cleanup func(), err error)

type options struct {
	apply   bool
	limit   int
	species []string
	workers int
}

type report struct {
	Counts      map[action]int
	Failures    []string
	BytesBefore int64
	BytesAfter  int64
}

// paddingAllowanceSecs is how much LONGER than its source the output may run.
// MP3 encoders add decoder delay and pad the final frame to a 1152-sample
// boundary, so the output always runs slightly long -- by a fixed amount, not a
// proportional one, which on a short recording is a large percentage. How much
// of it reaches the probed duration depends on whether the encoder wrote
// gapless metadata, so it varies between ffmpeg builds. A generous absolute
// allowance costs nothing: truncation, the failure this guards against, is
// measured in seconds.
const paddingAllowanceSecs = 0.25

// shortfallTolerance is how much SHORTER than its source the output may run.
// Nothing legitimate truncates audio, so this stays tight.
const shortfallTolerance = 0.01

// actionNone is returned alongside an error to make clear that no decision was
// reached, rather than reusing actionSkip's zero value to mean "no answer."
const actionNone action = -1

// sweep walks every recording, decides what it needs, and (when opts.apply is
// set) overwrites the object in place at its existing key and records its peaks.
// Without opts.apply it does all the same work locally and writes nothing, which
// is what makes the dry run a true sizing measurement rather than an estimate.
func sweep(ctx context.Context, w io.Writer, q recordingStore, obj objectStore, fetch fetcher, opts options) (report, error) {
	rows, err := q.ListRecordingsForTranscode(ctx)
	if err != nil {
		return report{}, fmt.Errorf("list recordings: %w", err)
	}

	if len(opts.species) > 0 {
		rows = slices.DeleteFunc(rows, func(r store.ListRecordingsForTranscodeRow) bool {
			return !slices.Contains(opts.species, r.SpeciesCode)
		})
	}
	if opts.limit > 0 && len(rows) > opts.limit {
		rows = rows[:opts.limit]
	}

	workers := max(opts.workers, 1)
	rep := report{Counts: map[action]int{}}
	var mu sync.Mutex

	rowCh := make(chan store.ListRecordingsForTranscodeRow)
	var wg sync.WaitGroup
	wg.Add(workers)
	for range workers {
		go func() {
			defer wg.Done()
			for row := range rowCh {
				if ctx.Err() != nil {
					continue
				}

				act, before, after, err := processRow(ctx, q, obj, fetch, row, opts.apply)
				if err != nil {
					if ctx.Err() != nil {
						// The context was canceled mid-flight, which is what made
						// processRow fail. That is an interrupt, not a row worth
						// reporting: the row's peaks are still NULL, so a rerun
						// picks it up again.
						continue
					}
					mu.Lock()
					rep.Failures = append(rep.Failures, fmt.Sprintf("%s (%s): %v", row.XenoCantoID, row.SpeciesCode, err))
					mu.Unlock()
					fmt.Fprintf(w, "  FAIL %s %s: %v\n", row.SpeciesCode, row.XenoCantoID, err)
					continue
				}

				mu.Lock()
				rep.Counts[act]++
				rep.BytesBefore += before
				rep.BytesAfter += after
				mu.Unlock()
				fmt.Fprintf(w, "  %-9s %s %s  %s -> %s\n", act, row.SpeciesCode, row.XenoCantoID, humanBytes(before), humanBytes(after))
			}
		}()
	}

feed:
	for _, row := range rows {
		select {
		case rowCh <- row:
		case <-ctx.Done():
			break feed
		}
	}
	close(rowCh)
	wg.Wait()

	writeSummary(w, rep, opts.apply)
	return rep, nil
}

// processRow handles a single recording and reports its action plus its byte
// sizes before and after.
//
// A row whose peaks are already populated is reported as actionSkip without
// ever calling fetch: peaks are only ever written by the transcoding ingest
// path or by this job, so a non-NULL peaks column already implies a
// conformant object. Paying for a download (or a HEAD request) just to
// confirm what the peaks column already proves is not worth it, so a skipped
// row's byte sizes are reported as 0, 0 rather than measured.
func processRow(ctx context.Context, q recordingStore, obj objectStore, fetch fetcher, row store.ListRecordingsForTranscodeRow, apply bool) (action, int64, int64, error) {
	if len(row.Peaks) > 0 {
		return actionSkip, 0, 0, nil
	}

	path, size, cleanup, err := fetch(ctx, row.FilePath)
	if err != nil {
		return actionNone, 0, 0, fmt.Errorf("fetch: %w", err)
	}
	defer cleanup()

	info, err := audio.Probe(ctx, path)
	if err != nil {
		return actionNone, 0, 0, fmt.Errorf("probe: %w", err)
	}

	act := decide(info)

	res, err := audio.Transcode(ctx, path)
	if err != nil {
		return actionNone, 0, 0, fmt.Errorf("transcode: %w", err)
	}
	if err := verify(ctx, res.Data, info.Duration); err != nil {
		return actionNone, 0, 0, err
	}

	// Both remaining actions re-upload: the transcode above already applied
	// peak-normalization gain, so leaving actionPeaksOnly's original object in
	// place would store peaks that describe audio the bucket doesn't hold.
	after := int64(len(res.Data))
	if !apply {
		return act, size, after, nil
	}

	// Same key, same content type: every file_path row stays valid and the
	// declared type finally matches the bytes.
	key := obj.KeyFor(row.FilePath)
	if _, err := obj.Upload(ctx, key, "audio/mpeg", bytes.NewReader(res.Data)); err != nil {
		return actionNone, 0, 0, fmt.Errorf("upload: %w", err)
	}
	if err := q.SetRecordingPeaks(ctx, store.SetRecordingPeaksParams{
		XenoCantoID: row.XenoCantoID,
		Peaks:       res.Peaks,
	}); err != nil {
		// The object is already replaced. The next run finds a conformant file
		// with no peaks and takes the peaks-only branch, so this heals itself.
		return actionNone, 0, 0, fmt.Errorf("set peaks: %w", err)
	}
	return act, size, after, nil
}

// verify refuses to replace a live object with output that does not decode or
// that lost audio. A rejected row keeps its original file.
func verify(ctx context.Context, data []byte, srcDuration float64) error {
	if len(data) == 0 {
		return fmt.Errorf("verify: encoder produced no output")
	}
	f, err := os.CreateTemp("", "flockdeck-verify-*.mp3")
	if err != nil {
		return fmt.Errorf("verify: %w", err)
	}
	defer os.Remove(f.Name()) //nolint:errcheck
	if _, err := f.Write(data); err != nil {
		f.Close()
		return fmt.Errorf("verify: %w", err)
	}
	f.Close()

	info, err := audio.Probe(ctx, f.Name())
	if err != nil {
		return fmt.Errorf("verify: %w", err)
	}
	if info.Format != "mp3" {
		return fmt.Errorf("verify: output probed as %q, not mp3", info.Format)
	}
	if srcDuration > 0 {
		if info.Duration < srcDuration*(1-shortfallTolerance) {
			return fmt.Errorf("verify: output lost audio (%.2fs -> %.2fs)", srcDuration, info.Duration)
		}
		if info.Duration > srcDuration+paddingAllowanceSecs {
			return fmt.Errorf("verify: output ran long (%.2fs -> %.2fs)", srcDuration, info.Duration)
		}
	}
	return nil
}

func writeSummary(w io.Writer, rep report, apply bool) {
	mode := "DRY RUN (nothing written)"
	if apply {
		mode = "APPLIED"
	}
	fmt.Fprintf(w, "\n%s\n", mode)
	fmt.Fprintf(w, "  transcoded: %d\n", rep.Counts[actionTranscode])
	fmt.Fprintf(w, "  peaks only: %d\n", rep.Counts[actionPeaksOnly])
	fmt.Fprintf(w, "  skipped:    %d\n", rep.Counts[actionSkip])
	fmt.Fprintf(w, "  failed:     %d\n", len(rep.Failures))
	fmt.Fprintf(w, "  bytes:      %s -> %s", humanBytes(rep.BytesBefore), humanBytes(rep.BytesAfter))
	if rep.BytesAfter > 0 {
		fmt.Fprintf(w, "  (%.1fx smaller)", float64(rep.BytesBefore)/float64(rep.BytesAfter))
	}
	fmt.Fprintln(w)
}

func humanBytes(n int64) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.0f KB", float64(n)/(1<<10))
	default:
		return fmt.Sprintf("%d B", n)
	}
}
