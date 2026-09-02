package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jameynakama/flockdeck/internal/audio"
	"github.com/jameynakama/flockdeck/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeStore struct {
	rows       []store.ListRecordingsForTranscodeRow
	peakParams []store.SetRecordingPeaksParams
}

func (f *fakeStore) ListRecordingsForTranscode(context.Context) ([]store.ListRecordingsForTranscodeRow, error) {
	return f.rows, nil
}

func (f *fakeStore) SetRecordingPeaks(_ context.Context, arg store.SetRecordingPeaksParams) error {
	f.peakParams = append(f.peakParams, arg)
	return nil
}

type fakeObjects struct {
	uploads map[string][]byte
	types   map[string]string
}

func newFakeObjects() *fakeObjects {
	return &fakeObjects{uploads: map[string][]byte{}, types: map[string]string{}}
}

func (f *fakeObjects) KeyFor(fileURL string) string {
	return "recordings/sonspa/" + filepath.Base(fileURL)
}

func (f *fakeObjects) Upload(_ context.Context, key, contentType string, body io.Reader) (string, error) {
	b, err := io.ReadAll(body)
	if err != nil {
		return "", err
	}
	f.uploads[key] = b
	f.types[key] = contentType
	return "https://media.flockdeck.com/" + key, nil
}

// fixtureFetcher serves a local fixture for every URL, standing in for R2.
func fixtureFetcher(t *testing.T, fixture string) fetcher {
	t.Helper()
	return func(context.Context, string) (string, int64, func(), error) {
		src, err := os.ReadFile(fixture)
		require.NoError(t, err)
		path := filepath.Join(t.TempDir(), filepath.Base(fixture))
		require.NoError(t, os.WriteFile(path, src, 0o600))
		return path, int64(len(src)), func() {}, nil
	}
}

func wavRow() store.ListRecordingsForTranscodeRow {
	return store.ListRecordingsForTranscodeRow{
		XenoCantoID: "111",
		SpeciesCode: "sonspa",
		FilePath:    "https://media.flockdeck.com/recordings/sonspa/111.mp3",
	}
}

func TestSweep_DryRunWritesNothing(t *testing.T) {
	q := &fakeStore{rows: []store.ListRecordingsForTranscodeRow{wavRow()}}
	obj := newFakeObjects()

	rep, err := sweep(context.Background(), io.Discard, q, obj,
		fixtureFetcher(t, "../../internal/audio/testdata/stereo.wav"),
		options{apply: false, workers: 1})
	require.NoError(t, err)

	assert.Empty(t, obj.uploads, "dry run must not upload")
	assert.Empty(t, q.peakParams, "dry run must not write peaks")
	assert.Equal(t, 1, rep.Counts[actionTranscode])
	assert.Greater(t, rep.BytesBefore, rep.BytesAfter, "dry run must still measure the saving")
}

func TestSweep_ApplyUploadsAtTheSameKeyAndWritesPeaks(t *testing.T) {
	q := &fakeStore{rows: []store.ListRecordingsForTranscodeRow{wavRow()}}
	obj := newFakeObjects()

	_, err := sweep(context.Background(), io.Discard, q, obj,
		fixtureFetcher(t, "../../internal/audio/testdata/stereo.wav"),
		options{apply: true, workers: 1})
	require.NoError(t, err)

	require.Len(t, obj.uploads, 1)
	body, ok := obj.uploads["recordings/sonspa/111.mp3"]
	require.True(t, ok, "must upload at the original key, unchanged")
	assert.Equal(t, "audio/mpeg", obj.types["recordings/sonspa/111.mp3"])
	assert.Equal(t, []byte("ID3"), body[:3])

	require.Len(t, q.peakParams, 1)
	assert.Equal(t, "111", q.peakParams[0].XenoCantoID)
	assert.Len(t, q.peakParams[0].Peaks, 1000)
}

// A row that is already conformant (mono, under the bit rate ceiling) but too
// quiet must still be re-uploaded: the transcode that measures its peaks
// applies real gain, so leaving the stored object alone would leave peaks on
// file that describe louder audio than what actually plays.
func TestSweep_ConformantButQuietRowIsStillUploaded(t *testing.T) {
	q := &fakeStore{rows: []store.ListRecordingsForTranscodeRow{wavRow()}}
	obj := newFakeObjects()

	rep, err := sweep(context.Background(), io.Discard, q, obj,
		fixtureFetcher(t, "../../internal/audio/testdata/quiet96.mp3"),
		options{apply: true, workers: 1})
	require.NoError(t, err)

	assert.Equal(t, 1, rep.Counts[actionPeaksOnly], "quiet96.mp3 is mono and under the bit rate ceiling")
	require.Len(t, obj.uploads, 1, "a peaks-only row must still be uploaded")
	body, ok := obj.uploads["recordings/sonspa/111.mp3"]
	require.True(t, ok, "must upload at the original key")
	assert.Equal(t, []byte("ID3"), body[:3])
	assert.Equal(t, int64(len(body)), rep.BytesAfter, "reported after-bytes must match the uploaded encode")

	require.Len(t, q.peakParams, 1)
	assert.Equal(t, "111", q.peakParams[0].XenoCantoID)
}

// A row whose peaks are already populated must be skipped before ever being
// fetched: peaks are only ever written by the transcoding ingest path or by
// this job, so a non-NULL peaks column already implies a conformant object.
func TestSweep_RowWithPeaksIsSkippedWithoutFetching(t *testing.T) {
	row := wavRow()
	row.Peaks = make([]int16, 1000)
	q := &fakeStore{rows: []store.ListRecordingsForTranscodeRow{row}}
	obj := newFakeObjects()

	var fetched int
	fetch := func(ctx context.Context, url string) (string, int64, func(), error) {
		fetched++
		return fixtureFetcher(t, "../../internal/audio/testdata/mono320.mp3")(ctx, url)
	}

	rep, err := sweep(context.Background(), io.Discard, q, obj, fetch, options{apply: true, workers: 1})
	require.NoError(t, err)

	assert.Zero(t, fetched, "a row with peaks must never be fetched")
	assert.Empty(t, obj.uploads)
	assert.Empty(t, q.peakParams)
	assert.Equal(t, 1, rep.Counts[actionSkip])
	assert.Zero(t, rep.BytesBefore, "a skipped row contributes nothing to the byte totals")
	assert.Zero(t, rep.BytesAfter, "a skipped row contributes nothing to the byte totals")
}

func TestSweep_LimitCapsRowsProcessed(t *testing.T) {
	rows := make([]store.ListRecordingsForTranscodeRow, 5)
	for i := range rows {
		rows[i] = wavRow()
		rows[i].XenoCantoID = string(rune('a' + i))
	}
	q := &fakeStore{rows: rows}
	obj := newFakeObjects()

	rep, err := sweep(context.Background(), io.Discard, q, obj,
		fixtureFetcher(t, "../../internal/audio/testdata/stereo.wav"),
		options{apply: false, limit: 2, workers: 1})
	require.NoError(t, err)

	assert.Equal(t, 2, rep.Counts[actionTranscode])
}

func TestSweep_SpeciesFilterSelectsRows(t *testing.T) {
	a, b := wavRow(), wavRow()
	a.SpeciesCode, a.XenoCantoID = "sonspa", "1"
	b.SpeciesCode, b.XenoCantoID = "amerob", "2"
	q := &fakeStore{rows: []store.ListRecordingsForTranscodeRow{a, b}}
	obj := newFakeObjects()

	_, err := sweep(context.Background(), io.Discard, q, obj,
		fixtureFetcher(t, "../../internal/audio/testdata/stereo.wav"),
		options{apply: true, species: []string{"amerob"}, workers: 1})
	require.NoError(t, err)

	require.Len(t, q.peakParams, 1)
	assert.Equal(t, "2", q.peakParams[0].XenoCantoID)
}

func TestSweep_VerificationFailureSkipsUpload(t *testing.T) {
	q := &fakeStore{rows: []store.ListRecordingsForTranscodeRow{wavRow()}}
	obj := newFakeObjects()

	// A text file is not audio, so transcoding fails and nothing may be written.
	fetch := func(context.Context, string) (string, int64, func(), error) {
		path := filepath.Join(t.TempDir(), "junk.mp3")
		require.NoError(t, os.WriteFile(path, []byte("not audio at all"), 0o600))
		return path, 16, func() {}, nil
	}

	rep, err := sweep(context.Background(), io.Discard, q, obj, fetch, options{apply: true, workers: 1})
	require.NoError(t, err)

	assert.Empty(t, obj.uploads, "a failed row must leave its original object alone")
	assert.Empty(t, q.peakParams)
	assert.Len(t, rep.Failures, 1)
}

func TestVerify_RejectsNonMP3Output(t *testing.T) {
	data, err := os.ReadFile("../../internal/audio/testdata/stereo.wav")
	require.NoError(t, err)

	err = verify(context.Background(), data, 2.0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not mp3", "must identify the format check as the failing one")
}

func TestVerify_RejectsDurationDrift(t *testing.T) {
	res, err := audio.Transcode(context.Background(), "../../internal/audio/testdata/stereo.wav")
	require.NoError(t, err)

	// The real output is ~2s; claiming a wildly different source duration must
	// trip the drift check even though the mp3 itself is valid.
	err = verify(context.Background(), res.Data, 100.0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duration drifted", "must identify the duration check as the failing one")
}

// With workers > 1, rows are dispatched to a fixed pool rather than one
// goroutine per row. The report must still land on the correct totals no
// matter how the pool interleaves them.
func TestSweep_WorkersGreaterThanOneProcessesAllRowsCorrectly(t *testing.T) {
	fixtureFor := map[string]string{
		"t1": "../../internal/audio/testdata/stereo.wav",
		"t2": "../../internal/audio/testdata/stereo.wav",
		"t3": "../../internal/audio/testdata/stereo.wav",
		"p1": "../../internal/audio/testdata/quiet96.mp3",
	}
	var rows []store.ListRecordingsForTranscodeRow
	for id := range fixtureFor {
		row := wavRow()
		row.XenoCantoID = id
		row.FilePath = "https://media.flockdeck.com/recordings/sonspa/" + id + ".mp3"
		rows = append(rows, row)
	}
	for _, id := range []string{"s1", "s2"} {
		row := wavRow()
		row.XenoCantoID = id
		row.Peaks = make([]int16, 1000)
		rows = append(rows, row)
	}

	q := &fakeStore{rows: rows}
	obj := newFakeObjects()
	fetch := func(ctx context.Context, url string) (string, int64, func(), error) {
		id := strings.TrimSuffix(strings.TrimPrefix(url, "https://media.flockdeck.com/recordings/sonspa/"), ".mp3")
		fixture, ok := fixtureFor[id]
		require.True(t, ok, "unexpected fetch for %s", url)
		return fixtureFetcher(t, fixture)(ctx, url)
	}

	rep, err := sweep(context.Background(), io.Discard, q, obj, fetch, options{apply: true, workers: 3})
	require.NoError(t, err)

	assert.Equal(t, 3, rep.Counts[actionTranscode])
	assert.Equal(t, 1, rep.Counts[actionPeaksOnly])
	assert.Equal(t, 2, rep.Counts[actionSkip])
	assert.Empty(t, rep.Failures)
	assert.Len(t, obj.uploads, 4, "every non-skip row must be uploaded")
}

// Interrupting a sweep cancels the context; the in-flight row's fetch fails
// with context.Canceled as a direct result, and that must not be recorded as
// a row failure (it would flood the terminal with FAIL lines and force a
// non-zero exit on nothing but an interrupt).
func TestSweep_ContextCancellationDuringFetchIsNotRecordedAsFailure(t *testing.T) {
	rows := make([]store.ListRecordingsForTranscodeRow, 5)
	for i := range rows {
		rows[i] = wavRow()
		rows[i].XenoCantoID = fmt.Sprintf("row%d", i)
	}
	q := &fakeStore{rows: rows}
	obj := newFakeObjects()

	ctx, cancel := context.WithCancel(context.Background())
	fetch := func(ctx context.Context, url string) (string, int64, func(), error) {
		cancel()
		return "", 0, func() {}, ctx.Err()
	}

	rep, err := sweep(ctx, io.Discard, q, obj, fetch, options{apply: true, workers: 1})
	require.NoError(t, err)

	assert.Empty(t, rep.Failures, "a row that only failed because the context was canceled must not be recorded")
	assert.Empty(t, obj.uploads)
	assert.Empty(t, q.peakParams)
}
