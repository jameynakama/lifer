package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jameynakama/flockdeck/internal/ebird"
	"github.com/jameynakama/flockdeck/internal/r2"
	"github.com/jameynakama/flockdeck/internal/xenocanto"
)

func nopSend(any) {}

func TestFilterBySpecies(t *testing.T) {
	taxMap := map[string]ebird.TaxonomyEntry{
		"busti": {SpeciesCode: "busti", CommonName: "Bushtit"},
		"rukin": {SpeciesCode: "rukin", CommonName: "Ruby-crowned Kinglet"},
		"amro":  {SpeciesCode: "amro", CommonName: "American Robin"},
	}
	codes := []string{"busti", "rukin", "amro"}

	tests := []struct {
		name    string
		want    []string
		wantOut []string
	}{
		{"ebird code", []string{"busti"}, []string{"busti"}},
		{"common name case-insensitive", []string{"bushtit"}, []string{"busti"}},
		{"mixed code and name", []string{"busti", "ruby-crowned kinglet"}, []string{"busti", "rukin"}},
		{"multiple codes", []string{"rukin", "amro"}, []string{"rukin", "amro"}},
		{"no match is excluded", []string{"busti", "doesnotexist"}, []string{"busti"}},
		{"empty want", []string{}, []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterBySpecies(codes, taxMap, tt.want)
			if len(got) != len(tt.wantOut) {
				t.Fatalf("filterBySpecies() len = %d, want %d (got %v, want %v)", len(got), len(tt.wantOut), got, tt.wantOut)
			}
			for i := range tt.wantOut {
				if got[i] != tt.wantOut[i] {
					t.Errorf("filterBySpecies()[%d] = %q, want %q", i, got[i], tt.wantOut[i])
				}
			}
		})
	}
}

func TestFilterArbitrary(t *testing.T) {
	tests := []struct {
		name     string
		codes    []string
		complete map[string]struct{}
		want     []string
	}{
		{
			name:     "empty complete set passes all through",
			codes:    []string{"AMRO", "BCCH", "NOCA"},
			complete: map[string]struct{}{},
			want:     []string{"AMRO", "BCCH", "NOCA"},
		},
		{
			name:     "complete species are removed",
			codes:    []string{"AMRO", "BCCH", "NOCA"},
			complete: map[string]struct{}{"BCCH": {}},
			want:     []string{"AMRO", "NOCA"},
		},
		{
			name:     "all complete returns empty slice",
			codes:    []string{"AMRO", "BCCH"},
			complete: map[string]struct{}{"AMRO": {}, "BCCH": {}},
			want:     []string{},
		},
		{
			name:     "nil codes returns empty slice",
			codes:    nil,
			complete: map[string]struct{}{"AMRO": {}},
			want:     []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterArbitrary(tt.codes, tt.complete)
			if len(got) != len(tt.want) {
				t.Fatalf("filterComplete() len = %d, want %d (got %v, want %v)", len(got), len(tt.want), got, tt.want)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("filterComplete()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestLoadOverrides(t *testing.T) {
	t.Run("valid file with entries and comments", func(t *testing.T) {
		f, err := os.CreateTemp(t.TempDir(), "overrides*.txt")
		if err != nil {
			t.Fatalf("create temp file: %v", err)
		}
		defer f.Close()
		f.WriteString(`
# XC overrides

comrav=Corvus:corax  # Common Raven -- eBird/XC genus mismatch
norgos=Accipiter:atricapillus
`)
		got, err := loadOverrides(f.Name())
		if err != nil {
			t.Fatalf("loadOverrides() error = %v", err)
		}
		want := map[string][2]string{
			"comrav": {"Corvus", "corax"},
			"norgos": {"Accipiter", "atricapillus"},
		}
		if len(got) != len(want) {
			t.Fatalf("len = %d, want %d: %v", len(got), len(want), got)
		}
		for k, v := range want {
			if got[k] != v {
				t.Errorf("[%s] = %v, want %v", k, got[k], v)
			}
		}
	})

	t.Run("missing file returns empty map", func(t *testing.T) {
		got, err := loadOverrides(t.TempDir() + "/nonexistent.txt")
		if err != nil {
			t.Fatalf("Should return nil error for missing file, got %v", err)
		}
		if len(got) != 0 {
			t.Errorf("Should return empty map, got %v", got)
		}
	})

	t.Run("malformed entry returns error", func(t *testing.T) {
		f, err := os.CreateTemp(t.TempDir(), "overrides*.txt")
		if err != nil {
			t.Fatalf("create temp file: %v", err)
		}
		defer f.Close()
		f.WriteString("badentry\n")
		_, err = loadOverrides(f.Name())
		if err == nil {
			t.Error("Should return error for malformed entry")
		}
	})

	t.Run("missing colon in genus:species returns error", func(t *testing.T) {
		f, err := os.CreateTemp(t.TempDir(), "overrides*.txt")
		if err != nil {
			t.Fatalf("create temp file: %v", err)
		}
		defer f.Close()
		f.WriteString("comrav=Corvuscorax\n")
		_, err = loadOverrides(f.Name())
		if err == nil {
			t.Error("Should return error for missing colon")
		}
	})
}

func TestLoadManualSkips(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "skip*.txt")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	defer f.Close()

	f.WriteString(`
# Species skipped during ingest

# Protected -- XC redacts download URLs
spoowl  # Spotted Owl

# No XC coverage / recent split with no separate records
cocboo1  # Cocos Booby

# Erroneous eBird reports for this region
zebfin2  # Zebra Finch (captive/escaped birds only)

# Other
yellbutt
`)
	res := loadManualSkips(f.Name())

	want := 4
	if len(res) != want {
		t.Errorf("loadManualSkips() len == %d; want %d", len(res), want)
	}

	for k := range map[string]struct{}{
		"spoowl":   {},
		"cocboo1":  {},
		"zebfin2":  {},
		"yellbutt": {},
	} {
		if _, ok := res[k]; !ok {
			t.Errorf("loadManualSkips()[%s] not found", k)
		}
	}
}

func TestInterleaveRecordings(t *testing.T) {
	rec := func(id string) xenocanto.Recording { return xenocanto.Recording{ID: id} }
	songs := []xenocanto.Recording{rec("s1"), rec("s2"), rec("s3")}
	calls := []xenocanto.Recording{rec("c1"), rec("c2"), rec("c3")}

	tests := []struct {
		name         string
		songs, calls []xenocanto.Recording
		n            int
		wantIDs      []string
	}{
		{"only calls fills budget", nil, calls, 4, []string{"c1", "c2", "c3"}},
		{"only songs fills budget", songs, nil, 4, []string{"s1", "s2", "s3"}},
		{"interleaves when both available", songs, calls, 4, []string{"s1", "c1", "s2", "c2"}},
		{"cap respected", songs, calls, 2, []string{"s1", "c1"}},
		{"call-heavy fills remaining from calls", []xenocanto.Recording{rec("s1")}, calls, 4, []string{"s1", "c1", "c2", "c3"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := interleaveRecordings(tt.songs, tt.calls, tt.n)
			if len(got) != len(tt.wantIDs) {
				t.Fatalf("len = %d, want %d: %v", len(got), len(tt.wantIDs), got)
			}
			for i, id := range tt.wantIDs {
				if got[i].ID != id {
					t.Errorf("[%d] = %q, want %q", i, got[i].ID, id)
				}
			}
		})
	}
}

func TestFetchAndUpload_Success(t *testing.T) {
	src := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("audio bytes"))
	}))
	defer src.Close()

	var uploaded string
	r2s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			b, _ := io.ReadAll(r.Body)
			uploaded = string(b)
			w.Header().Set("ETag", `"x"`)
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer r2s.Close()

	r2c, err := r2.NewWithEndpoint(r2s.URL, "k", "s", "bucket", "https://pub.example.com")
	if err != nil {
		t.Fatalf("Should create r2 client: %v", err)
	}

	url, err := fetchAndUpload(context.Background(), r2c, src.URL, "recordings/busti/123.mp3", "audio/mpeg", 0, nopSend)
	if err != nil {
		t.Fatalf("Should fetch and upload without error: %v", err)
	}
	if url != "https://pub.example.com/recordings/busti/123.mp3" {
		t.Errorf("Should return public URL, got %q", url)
	}
	if uploaded != "audio bytes" {
		t.Errorf("Should upload source body, got %q", uploaded)
	}
}

func TestFetchAndUpload_SourceNonOK_ReturnsError(t *testing.T) {
	src := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer src.Close()

	r2c, _ := r2.NewWithEndpoint("http://localhost:1", "k", "s", "bucket", "https://pub.example.com")
	_, err := fetchAndUpload(context.Background(), r2c, src.URL, "key", "audio/mpeg", 0, nopSend)
	if err == nil {
		t.Error("Should return error for non-200 source response")
	}
}

func TestFetchAndUpload_Retries429(t *testing.T) {
	origDelays := retryDelays
	retryDelays = []time.Duration{0, 0, 0}
	t.Cleanup(func() { retryDelays = origDelays })

	attempts := 0
	src := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Write([]byte("ok"))
	}))
	defer src.Close()

	r2s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", `"x"`)
		w.WriteHeader(http.StatusOK)
	}))
	defer r2s.Close()

	r2c, _ := r2.NewWithEndpoint(r2s.URL, "k", "s", "bucket", "https://pub.example.com")
	_, err := fetchAndUpload(context.Background(), r2c, src.URL, "key", "audio/mpeg", 0, nopSend)
	if err != nil {
		t.Fatalf("Should succeed after retries: %v", err)
	}
	if attempts != 3 {
		t.Errorf("Should take 3 attempts, got %d", attempts)
	}
}

func TestFetchAndUpload_Retries503(t *testing.T) {
	origDelays := retryDelays
	retryDelays = []time.Duration{0, 0, 0}
	t.Cleanup(func() { retryDelays = origDelays })

	attempts := 0
	src := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Write([]byte("ok"))
	}))
	defer src.Close()

	r2s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", `"x"`)
		w.WriteHeader(http.StatusOK)
	}))
	defer r2s.Close()

	r2c, _ := r2.NewWithEndpoint(r2s.URL, "k", "s", "bucket", "https://pub.example.com")
	_, err := fetchAndUpload(context.Background(), r2c, src.URL, "key", "audio/mpeg", 0, nopSend)
	if err != nil {
		t.Fatalf("Should succeed after 503 retries: %v", err)
	}
	if attempts != 3 {
		t.Errorf("Should take 3 attempts, got %d", attempts)
	}
}

func TestFetchAndUpload_SendsMessageSequence(t *testing.T) {
	src := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("data"))
	}))
	defer src.Close()

	r2s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", `"x"`)
		w.WriteHeader(http.StatusOK)
	}))
	defer r2s.Close()

	r2c, _ := r2.NewWithEndpoint(r2s.URL, "k", "s", "bucket", "https://pub.example.com")

	var msgs []any
	send := func(msg any) { msgs = append(msgs, msg) }

	_, err := fetchAndUpload(context.Background(), r2c, src.URL, "recordings/amro/XC1.mp3", "audio/mpeg", 2, send)
	if err != nil {
		t.Fatalf("Should succeed: %v", err)
	}

	if len(msgs) != 3 {
		t.Fatalf("Should send 3 messages (fetch, upload, done), got %d: %v", len(msgs), msgs)
	}
	if _, ok := msgs[0].(fetchStartedMsg); !ok {
		t.Errorf("msg[0] should be fetchStartedMsg, got %T", msgs[0])
	}
	if _, ok := msgs[1].(uploadStartedMsg); !ok {
		t.Errorf("msg[1] should be uploadStartedMsg, got %T", msgs[1])
	}
	done, ok := msgs[2].(uploadDoneMsg)
	if !ok {
		t.Errorf("msg[2] should be uploadDoneMsg, got %T", msgs[2])
	}
	if done.err != nil {
		t.Errorf("uploadDoneMsg should have nil err, got %v", done.err)
	}
	if done.workerID != 2 {
		t.Errorf("Should carry workerID 2, got %d", done.workerID)
	}
}

func TestRecordOutcome(t *testing.T) {
	tests := []struct {
		name        string
		stats       ingestStats
		err         error
		wantFailed  []string
		wantMissing bool
	}{
		{
			name:       "ingest error appends to recorded failures",
			stats:      ingestStats{failures: []string{"recording XC1: boom"}},
			err:        fmt.Errorf("upsert species: db down"),
			wantFailed: []string{"recording XC1: boom", "ingest: upsert species: db down"},
		},
		{
			name:       "partial failures alone mark the species failed",
			stats:      ingestStats{failures: []string{"image ML1: 404"}, recordings: 2, images: 1},
			wantFailed: []string{"image ML1: 404"},
		},
		{
			name:        "no failures but zero recordings is missing media",
			stats:       ingestStats{recordings: 0, images: 3},
			wantMissing: true,
		},
		{
			name:        "no failures but zero images is missing media",
			stats:       ingestStats{recordings: 2, images: 0},
			wantMissing: true,
		},
		{
			name:  "full success records nothing",
			stats: ingestStats{recordings: 2, images: 3},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			failed := map[string][]string{}
			missing := map[string]ingestStats{}
			recordOutcome(failed, missing, "sonspa", tt.stats, tt.err)

			if tt.wantFailed == nil {
				if _, ok := failed["sonspa"]; ok {
					t.Errorf("Should not mark species failed, got %v", failed["sonspa"])
				}
			} else if !slices.Equal(failed["sonspa"], tt.wantFailed) {
				t.Errorf("Should record failures %v, got %v", tt.wantFailed, failed["sonspa"])
			}

			if _, ok := missing["sonspa"]; ok != tt.wantMissing {
				t.Errorf("Should have missing-media entry = %v, got %v", tt.wantMissing, ok)
			}
		})
	}
}

func TestDryRunExit_ExitsZero(t *testing.T) {
	exitCode := -1
	dryRunExit(io.Discard, func(code int) { exitCode = code })
	if exitCode != 0 {
		t.Errorf("Should exit 0, got %d", exitCode)
	}
}

func TestDryRunExit_PrintsDryRunMessage(t *testing.T) {
	var buf bytes.Buffer
	dryRunExit(&buf, func(int) {})
	if !strings.Contains(strings.ToLower(buf.String()), "dry run") {
		t.Errorf("Should mention dry run in output, got %q", buf.String())
	}
}
