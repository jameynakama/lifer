package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jameynakama/flockdeck/internal/ebird"
	"github.com/jameynakama/flockdeck/internal/macaulay"
	"github.com/jameynakama/flockdeck/internal/store"
	"github.com/jameynakama/flockdeck/internal/xenocanto"
)

var testEntry = ebird.TaxonomyEntry{
	SpeciesCode:   "sonspa",
	CommonName:    "Song Sparrow",
	SciName:       "Melospiza melodia",
	FamilyComName: "New World Sparrows",
}

// xcServer serves one A-quality song and no calls.
func xcServer(t *testing.T) *xenocanto.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Query().Get("query"), "type:song") {
			fmt.Fprint(w, `{"recordings":[{"id":"XC1","type":"song","q":"A","length":"0:30","file":"https://example.com/xc1.mp3","rec":"Recordist"}]}`)
			return
		}
		fmt.Fprint(w, `{"recordings":[]}`)
	}))
	t.Cleanup(srv.Close)
	return xenocanto.NewWithBaseURL("key", srv.URL)
}

// macServer serves one photo, or only errors when status != 0.
func macServer(t *testing.T, status int) *macaulay.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if status != 0 {
			w.WriteHeader(status)
			return
		}
		fmt.Fprint(w, `{"results":{"content":[{"assetId":"ML1","userDisplayName":"Photographer"}]}}`)
	}))
	t.Cleanup(srv.Close)
	return macaulay.NewWithBaseURL("key", srv.URL)
}

// happyStore upserts everything and records the media params it saw.
func happyStore(t *testing.T) (*stubUpserter, *[]store.UpsertRecordingParams, *[]store.UpsertSpeciesImageParams) {
	var recs []store.UpsertRecordingParams
	var imgs []store.UpsertSpeciesImageParams
	q := &stubUpserter{
		t: t,
		upsertSpecies: func(arg store.UpsertSpeciesParams) (store.Species, error) {
			return store.Species{EbirdCode: arg.EbirdCode}, nil
		},
		upsertRecording: func(arg store.UpsertRecordingParams) (store.SpeciesRecording, error) {
			recs = append(recs, arg)
			return store.SpeciesRecording{}, nil
		},
		upsertSpeciesImage: func(arg store.UpsertSpeciesImageParams) (store.SpeciesImage, error) {
			imgs = append(imgs, arg)
			return store.SpeciesImage{}, nil
		},
	}
	return q, &recs, &imgs
}

func TestIngestSpecies_DampRun_UpsertsPlaceholderMedia(t *testing.T) {
	q, recs, imgs := happyStore(t)
	stats, err := ingestSpecies(context.Background(), q, xcServer(t), macServer(t, 0),
		testEntry, 4, 3, 0, nil, nil, nil, 0, discard)
	if err != nil {
		t.Fatalf("Should ingest without error, got %v", err)
	}
	if len(stats.failures) != 0 {
		t.Fatalf("Should have no failures, got %v", stats.failures)
	}
	if stats.recordings != 1 || stats.images != 1 {
		t.Errorf("Should count 1 recording and 1 image, got %d/%d", stats.recordings, stats.images)
	}
	if len(*recs) != 1 || (*recs)[0].XenoCantoID != "XC1" || (*recs)[0].FilePath != "placeholder://recordings/sonspa/XC1.mp3" {
		t.Errorf("Should upsert placeholder recording for XC1, got %+v", *recs)
	}
	if len(*imgs) != 1 || (*imgs)[0].MacaulayID != "ML1" || (*imgs)[0].FilePath != "placeholder://images/sonspa/ML1.jpg" {
		t.Errorf("Should upsert placeholder image for ML1, got %+v", *imgs)
	}
}

func TestIngestSpecies_MacaulayError_RecordedNotFatal(t *testing.T) {
	q, recs, _ := happyStore(t)
	stats, err := ingestSpecies(context.Background(), q, xcServer(t), macServer(t, http.StatusInternalServerError),
		testEntry, 4, 3, 0, nil, nil, nil, 0, discard)
	if err != nil {
		t.Fatalf("Should not be fatal (recordings may have succeeded), got %v", err)
	}
	if len(stats.failures) != 1 || !strings.Contains(stats.failures[0], "macaulay photos") {
		t.Errorf("Should record the macaulay failure for the report, got %v", stats.failures)
	}
	if stats.recordings != 1 || len(*recs) != 1 {
		t.Errorf("Should still have ingested the recording, got %d", stats.recordings)
	}
}

func TestIngestSpecies_XCSearchError_RecordedAndImagesContinue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	xc := xenocanto.NewWithBaseURL("key", srv.URL)

	q, _, imgs := happyStore(t)
	stats, err := ingestSpecies(context.Background(), q, xc, macServer(t, 0),
		testEntry, 4, 3, 0, nil, nil, nil, 0, discard)
	if err != nil {
		t.Fatalf("Should not be fatal, got %v", err)
	}
	if len(stats.failures) != 2 {
		t.Errorf("Should record song and call search failures, got %v", stats.failures)
	}
	if stats.images != 1 || len(*imgs) != 1 {
		t.Errorf("Should still ingest images after search failures, got %d images", stats.images)
	}
}

func TestIngestSpecies_BannedImagesAreSkipped(t *testing.T) {
	q, _, imgs := happyStore(t)
	banned := map[string]struct{}{"ML1": {}}
	stats, err := ingestSpecies(context.Background(), q, xcServer(t), macServer(t, 0),
		testEntry, 4, 3, 0, nil, nil, banned, 0, discard)
	if err != nil {
		t.Fatalf("Should ingest without error, got %v", err)
	}
	if stats.images != 0 || len(*imgs) != 0 {
		t.Errorf("Should skip banned image ML1, got %d upserts", len(*imgs))
	}
}

func TestIngestSpecies_UpsertSpeciesError_IsFatal(t *testing.T) {
	q := &stubUpserter{
		t: t,
		upsertSpecies: func(store.UpsertSpeciesParams) (store.Species, error) {
			return store.Species{}, fmt.Errorf("db down")
		},
	}
	_, err := ingestSpecies(context.Background(), q, xcServer(t), macServer(t, 0),
		testEntry, 4, 3, 0, nil, nil, nil, 0, discard)
	if err == nil || !strings.Contains(err.Error(), "upsert species") {
		t.Errorf("Should fail fast when the species row can't be written, got %v", err)
	}
}

func TestIngestSpecies_UpsertsFamily(t *testing.T) {
	var got store.UpsertSpeciesParams
	q, _, _ := happyStore(t)
	q.upsertSpecies = func(arg store.UpsertSpeciesParams) (store.Species, error) {
		got = arg
		return store.Species{EbirdCode: arg.EbirdCode}, nil
	}
	_, err := ingestSpecies(context.Background(), q, xcServer(t), macServer(t, 0),
		testEntry, 4, 3, 0, nil, nil, nil, 0, discard)
	if err != nil {
		t.Fatalf("Should ingest without error, got %v", err)
	}
	if !got.Family.Valid || got.Family.String != "New World Sparrows" {
		t.Errorf("Should upsert the eBird family name, got %+v", got.Family)
	}
}

func TestIngestSpecies_EmptyFamily_UpsertsNull(t *testing.T) {
	entry := testEntry
	entry.FamilyComName = ""
	var got store.UpsertSpeciesParams
	q, _, _ := happyStore(t)
	q.upsertSpecies = func(arg store.UpsertSpeciesParams) (store.Species, error) {
		got = arg
		return store.Species{EbirdCode: arg.EbirdCode}, nil
	}
	_, err := ingestSpecies(context.Background(), q, xcServer(t), macServer(t, 0),
		entry, 4, 3, 0, nil, nil, nil, 0, discard)
	if err != nil {
		t.Fatalf("Should ingest without error, got %v", err)
	}
	if got.Family.Valid {
		t.Errorf("Should upsert NULL family when eBird has none, got %+v", got.Family)
	}
}
