# Ingestion Script Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a `cmd/ingest` binary that populates the species, recordings, and species_images catalog from eBird, xeno-canto, and Macaulay Library.

**Architecture:** Three typed API clients in `internal/ebird`, `internal/xenocanto`, `internal/macaulay`. A buffered-channel semaphore + `sync.WaitGroup` worker pool in `cmd/ingest/main.go` orchestrates concurrent per-species ingestion. All DB writes are idempotent upserts.

**Tech Stack:** Go 1.26, chi router (existing), pgx/v5, sqlc, golang-migrate, testify for tests.

---

## File Map

| Path | Action | Responsibility |
|------|--------|---------------|
| `backend/migrations/002_add_recording_type.up.sql` | Create | Add `type` column to recordings |
| `backend/migrations/002_add_recording_type.down.sql` | Create | Drop `type` column |
| `backend/internal/store/queries/ingest.sql` | Create | Upsert queries for species, recordings, images |
| `backend/internal/store/ingest.sql.go` | Generated | sqlc output -- do not edit |
| `backend/internal/store/models.go` | Generated | Gains `Type string` on `Recording` |
| `backend/internal/ebird/client.go` | Create | eBird API client |
| `backend/internal/ebird/client_test.go` | Create | httptest-based tests |
| `backend/internal/xenocanto/client.go` | Create | xeno-canto API client |
| `backend/internal/xenocanto/client_test.go` | Create | httptest-based tests |
| `backend/internal/macaulay/client.go` | Create | Macaulay Library client |
| `backend/internal/macaulay/client_test.go` | Create | httptest-based tests |
| `backend/cmd/ingest/main.go` | Create | Flags, worker pool, orchestration |
| `Justfile` | Modify | Add `just ingest` recipe |
| `.env.example` | Modify | Add `ASSETS_DIR` |

---

## Task 1: Migration -- add `type` to recordings

**Files:**
- Create: `backend/migrations/002_add_recording_type.up.sql`
- Create: `backend/migrations/002_add_recording_type.down.sql`

- [ ] **Step 1: Create up migration**

`backend/migrations/002_add_recording_type.up.sql`:
```sql
ALTER TABLE recordings ADD COLUMN type TEXT NOT NULL DEFAULT '';
```

- [ ] **Step 2: Create down migration**

`backend/migrations/002_add_recording_type.down.sql`:
```sql
ALTER TABLE recordings DROP COLUMN type;
```

- [ ] **Step 3: Verify sqlc parses the new schema**

Run from repo root:
```bash
cd backend && sqlc generate
```
Expected: no errors. Open `backend/internal/store/models.go` and confirm `Recording` struct now has:
```go
Type string `db:"type" json:"type"`
```

- [ ] **Step 4: Commit**

```bash
jj describe -m "migration: add type column to recordings"
jj new
```

---

## Task 2: sqlc ingest queries

**Files:**
- Create: `backend/internal/store/queries/ingest.sql`
- Regenerate: `backend/internal/store/` (run `just generate`)

- [ ] **Step 1: Write the queries**

`backend/internal/store/queries/ingest.sql`:
```sql
-- name: UpsertSpecies :one
INSERT INTO species (common_name, scientific_name, ebird_code)
VALUES ($1, $2, $3)
ON CONFLICT (ebird_code) DO UPDATE
    SET common_name     = EXCLUDED.common_name,
        scientific_name = EXCLUDED.scientific_name
RETURNING *;

-- name: UpsertRecording :one
INSERT INTO recordings (species_id, xeno_canto_id, file_path, quality, type)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (xeno_canto_id) DO UPDATE
    SET file_path = EXCLUDED.file_path,
        quality   = EXCLUDED.quality,
        type      = EXCLUDED.type
RETURNING *;

-- name: UpsertSpeciesImage :one
INSERT INTO species_images (species_id, macaulay_id, file_path, credit)
VALUES ($1, $2, $3, $4)
ON CONFLICT (macaulay_id) DO UPDATE
    SET file_path = EXCLUDED.file_path,
        credit    = EXCLUDED.credit
RETURNING *;
```

- [ ] **Step 2: Generate**

```bash
just generate
```
Expected: no errors. `backend/internal/store/ingest.sql.go` is created with `UpsertSpecies`, `UpsertRecording`, `UpsertSpeciesImage` methods and their `*Params` structs.

- [ ] **Step 3: Confirm generated param struct fields**

Check `backend/internal/store/ingest.sql.go`. You should see:
```go
type UpsertSpeciesParams struct {
    CommonName     string `db:"common_name" json:"common_name"`
    ScientificName string `db:"scientific_name" json:"scientific_name"`
    EbirdCode      string `db:"ebird_code" json:"ebird_code"`
}

type UpsertRecordingParams struct {
    SpeciesID   int64  `db:"species_id" json:"species_id"`
    XenoCantoID string `db:"xeno_canto_id" json:"xeno_canto_id"`
    FilePath    string `db:"file_path" json:"file_path"`
    Quality     string `db:"quality" json:"quality"`
    Type        string `db:"type" json:"type"`
}

type UpsertSpeciesImageParams struct {
    SpeciesID  int64  `db:"species_id" json:"species_id"`
    MacaulayID string `db:"macaulay_id" json:"macaulay_id"`
    FilePath   string `db:"file_path" json:"file_path"`
    Credit     string `db:"credit" json:"credit"`
}
```

- [ ] **Step 4: Run existing tests to ensure nothing broke**

```bash
just test
```
Expected: PASS (no existing tests touch these new queries)

- [ ] **Step 5: Commit**

```bash
jj describe -m "sqlc: add upsert queries for species, recordings, images"
jj new
```

---

## Task 3: Add testify

- [ ] **Step 1: Add testify**

```bash
cd backend && go get github.com/stretchr/testify@latest
```

- [ ] **Step 2: Tidy**

```bash
cd backend && go mod tidy
```

- [ ] **Step 3: Commit**

```bash
jj describe -m "deps: add testify for tests"
jj new
```

---

## Task 4: eBird API client

**Files:**
- Create: `backend/internal/ebird/client.go`
- Create: `backend/internal/ebird/client_test.go`

- [ ] **Step 1: Write the failing tests**

`backend/internal/ebird/client_test.go`:
```go
package ebird

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaxonomy(t *testing.T) -> void {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/ref/taxonomy/ebird", r.URL.Path)
		assert.Equal(t, "testkey", r.Header.Get("X-eBirdApiToken"))
		json.NewEncoder(w).Encode([]TaxonomyEntry{
			{SpeciesCode: "soospa", CommonName: "Song Sparrow", SciName: "Melospiza melodia", Category: "species"},
			{SpeciesCode: "norcaw", CommonName: "Northwestern Crow", SciName: "Corvus caurinus", Category: "species"},
		})
	}))
	defer srv.Close()

	c := newWithBaseURL("testkey", srv.URL)
	entries, err := c.Taxonomy(context.Background())
	require.NoError(t, err)
	assert.Len(t, entries, 2)
	assert.Equal(t, "soospa", entries[0].SpeciesCode)
	assert.Equal(t, "Melospiza melodia", entries[0].SciName)
}

func TestTaxonomyHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := newWithBaseURL("bad", srv.URL)
	_, err := c.Taxonomy(context.Background())
	assert.ErrorContains(t, err, "401")
}

func TestSpeciesList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/product/spplist/US-OR", r.URL.Path)
		json.NewEncoder(w).Encode([]string{"soospa", "norcaw", "mallar"})
	}))
	defer srv.Close()

	c := newWithBaseURL("testkey", srv.URL)
	codes, err := c.SpeciesList(context.Background(), "US-OR")
	require.NoError(t, err)
	assert.Equal(t, []string{"soospa", "norcaw", "mallar"}, codes)
}
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
cd backend && go test ./internal/ebird/... -v
```
Expected: compile error -- package doesn't exist yet.

- [ ] **Step 3: Implement the client**

`backend/internal/ebird/client.go`:
```go
package ebird

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type TaxonomyEntry struct {
	SpeciesCode string `json:"speciesCode"`
	CommonName  string `json:"comName"`
	SciName     string `json:"sciName"`
	Category    string `json:"category"`
}

type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

func New(apiKey string) *Client {
	return newWithBaseURL(apiKey, "https://api.ebird.org")
}

func newWithBaseURL(apiKey, baseURL string) *Client {
	return &Client{
		apiKey:     apiKey,
		baseURL:    baseURL,
		httpClient: &http.Client{},
	}
}

func (c *Client) Taxonomy(ctx context.Context) ([]TaxonomyEntry, error) {
	url := c.baseURL + "/v2/ref/taxonomy/ebird?fmt=json"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-eBirdApiToken", c.apiKey)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ebird taxonomy: status %d", resp.StatusCode)
	}
	var entries []TaxonomyEntry
	return entries, json.NewDecoder(resp.Body).Decode(&entries)
}

func (c *Client) SpeciesList(ctx context.Context, regionCode string) ([]string, error) {
	url := fmt.Sprintf("%s/v2/product/spplist/%s", c.baseURL, regionCode)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-eBirdApiToken", c.apiKey)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ebird spplist %s: status %d", regionCode, resp.StatusCode)
	}
	var codes []string
	return codes, json.NewDecoder(resp.Body).Decode(&codes)
}
```

- [ ] **Step 4: Run tests**

```bash
cd backend && go test ./internal/ebird/... -v
```
Expected: PASS (3 tests)

- [ ] **Step 5: Commit**

```bash
jj describe -m "feat: eBird API client with taxonomy and species list"
jj new
```

---

## Task 5: xeno-canto API client

**Files:**
- Create: `backend/internal/xenocanto/client.go`
- Create: `backend/internal/xenocanto/client_test.go`

- [ ] **Step 1: Write the failing tests**

`backend/internal/xenocanto/client_test.go`:
```go
package xenocanto

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/2/recordings", r.URL.Path)
		q := r.URL.Query().Get("query")
		assert.Contains(t, q, "gen:Melospiza")
		assert.Contains(t, q, "sp:melodia")
		assert.Contains(t, q, "type:song")
		json.NewEncoder(w).Encode(apiResponse{
			Recordings: []Recording{
				{ID: "111", Type: "song", Quality: "A", FileURL: "//example.com/111.mp3"},
				{ID: "222", Type: "song", Quality: "B", FileURL: "//example.com/222.mp3"},
				{ID: "333", Type: "song", Quality: "C", FileURL: "//example.com/333.mp3"},
			},
		})
	}))
	defer srv.Close()

	c := newWithBaseURL("", srv.URL)
	recs, err := c.Search(context.Background(), "Melospiza", "melodia", "song")
	require.NoError(t, err)
	// quality C is filtered out
	assert.Len(t, recs, 2)
	assert.Equal(t, "111", recs[0].ID)
	assert.Equal(t, "A", recs[0].Quality)
	assert.Equal(t, "https://example.com/111.mp3", recs[0].FileURL)
	assert.Equal(t, "222", recs[1].ID)
}

func TestSearchHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	c := newWithBaseURL("", srv.URL)
	_, err := c.Search(context.Background(), "Melospiza", "melodia", "song")
	assert.ErrorContains(t, err, "429")
}
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
cd backend && go test ./internal/xenocanto/... -v
```
Expected: compile error -- package doesn't exist yet.

- [ ] **Step 3: Implement the client**

`backend/internal/xenocanto/client.go`:
```go
package xenocanto

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type Recording struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Quality string `json:"q"`
	FileURL string `json:"file"`
}

type apiResponse struct {
	Recordings []Recording `json:"recordings"`
}

type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

func New(apiKey string) *Client {
	return newWithBaseURL(apiKey, "https://xeno-canto.org")
}

func newWithBaseURL(apiKey, baseURL string) *Client {
	return &Client{
		apiKey:     apiKey,
		baseURL:    baseURL,
		httpClient: &http.Client{},
	}
}

// Search returns recordings for genus/species of the given type ("song" or "call"),
// filtered to quality A or B, A-first.
func (c *Client) Search(ctx context.Context, genus, species, recType string) ([]Recording, error) {
	params := url.Values{}
	params.Set("query", fmt.Sprintf("gen:%s sp:%s type:%s", genus, species, recType))
	if c.apiKey != "" {
		params.Set("key", c.apiKey)
	}
	u := fmt.Sprintf("%s/api/2/recordings?%s", c.baseURL, params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("xeno-canto search: status %d", resp.StatusCode)
	}
	var r apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, err
	}
	return filterAndNormalize(r.Recordings), nil
}

func filterAndNormalize(recs []Recording) []Recording {
	var as, bs []Recording
	for _, r := range recs {
		// xeno-canto sometimes returns protocol-relative URLs
		if strings.HasPrefix(r.FileURL, "//") {
			r.FileURL = "https:" + r.FileURL
		}
		switch r.Quality {
		case "A":
			as = append(as, r)
		case "B":
			bs = append(bs, r)
		}
	}
	return append(as, bs...)
}
```

- [ ] **Step 4: Run tests**

```bash
cd backend && go test ./internal/xenocanto/... -v
```
Expected: PASS (2 tests)

- [ ] **Step 5: Commit**

```bash
jj describe -m "feat: xeno-canto API client with quality filtering"
jj new
```

---

## Task 6: Macaulay Library client

**Files:**
- Create: `backend/internal/macaulay/client.go`
- Create: `backend/internal/macaulay/client_test.go`

- [ ] **Step 1: Write the failing tests**

`backend/internal/macaulay/client_test.go`:
```go
package macaulay

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPhotos(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/ref/media/best", r.URL.Path)
		assert.Equal(t, "soospa", r.URL.Query().Get("species"))
		assert.Equal(t, "photo", r.URL.Query().Get("mediaType"))
		assert.Equal(t, "testkey", r.Header.Get("X-eBirdApiToken"))
		json.NewEncoder(w).Encode([]Photo{
			{AssetID: "111111111", UserDisplayName: "Jane Birder"},
			{AssetID: "222222222", UserDisplayName: "John Watcher"},
			{AssetID: "333333333", UserDisplayName: "Alice Finch"},
			{AssetID: "444444444", UserDisplayName: "Bob Sparrow"},
		})
	}))
	defer srv.Close()

	c := newWithBaseURL("testkey", srv.URL)
	photos, err := c.Photos(context.Background(), "soospa", 3)
	require.NoError(t, err)
	assert.Len(t, photos, 3)
	assert.Equal(t, "111111111", photos[0].AssetID)
	assert.Equal(t, "Jane Birder", photos[0].UserDisplayName)
}

func TestPhotosHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	c := newWithBaseURL("bad", srv.URL)
	_, err := c.Photos(context.Background(), "soospa", 3)
	assert.ErrorContains(t, err, "403")
}

func TestPhotoURL(t *testing.T) {
	c := New("key")
	assert.Equal(t,
		"https://cdn.download.ams.birds.cornell.edu/api/v1/asset/12345678/large",
		c.PhotoURL("12345678"),
	)
}
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
cd backend && go test ./internal/macaulay/... -v
```
Expected: compile error -- package doesn't exist yet.

- [ ] **Step 3: Implement the client**

`backend/internal/macaulay/client.go`:
```go
package macaulay

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type Photo struct {
	AssetID         string `json:"assetId"`
	UserDisplayName string `json:"userDisplayName"`
}

type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

func New(apiKey string) *Client {
	return newWithBaseURL(apiKey, "https://api.ebird.org")
}

func newWithBaseURL(apiKey, baseURL string) *Client {
	return &Client{
		apiKey:     apiKey,
		baseURL:    baseURL,
		httpClient: &http.Client{},
	}
}

// Photos returns up to max photos for the given eBird species code.
func (c *Client) Photos(ctx context.Context, speciesCode string, max int) ([]Photo, error) {
	params := url.Values{}
	params.Set("species", speciesCode)
	params.Set("mediaType", "photo")
	u := fmt.Sprintf("%s/v2/ref/media/best?%s", c.baseURL, params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-eBirdApiToken", c.apiKey)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("macaulay photos %s: status %d", speciesCode, resp.StatusCode)
	}
	var photos []Photo
	if err := json.NewDecoder(resp.Body).Decode(&photos); err != nil {
		return nil, err
	}
	if len(photos) > max {
		photos = photos[:max]
	}
	return photos, nil
}

func (c *Client) PhotoURL(assetID string) string {
	return fmt.Sprintf("https://cdn.download.ams.birds.cornell.edu/api/v1/asset/%s/large", assetID)
}
```

- [ ] **Step 4: Run tests**

```bash
cd backend && go test ./internal/macaulay/... -v
```
Expected: PASS (3 tests)

- [ ] **Step 5: Run all tests**

```bash
just test
```
Expected: PASS

- [ ] **Step 6: Commit**

```bash
jj describe -m "feat: Macaulay Library client for species photos"
jj new
```

---

## Task 7: cmd/ingest/main.go

**Files:**
- Create: `backend/cmd/ingest/main.go`

- [ ] **Step 1: Apply the migration to the local database**

Ensure Postgres is running first:
```bash
docker compose up -d
```

Then apply:
```bash
just migrate-up
```
Expected: `2/u 002_add_recording_type` (migration applied)

- [ ] **Step 2: Write main.go**

`backend/cmd/ingest/main.go`:
```go
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
	log.Printf("ingestion complete: %d/%d species", done, total)
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

	perType := maxRec / 2
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
			if err := downloadFile(rec.FileURL, destPath); err != nil {
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
		return nil
	}
	for _, photo := range photos {
		destPath := filepath.Join(assetsDir, "images", entry.SpeciesCode, photo.AssetID+".jpg")
		if err := downloadFile(mac.PhotoURL(photo.AssetID), destPath); err != nil {
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

func downloadFile(rawURL, destPath string) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return err
	}
	resp, err := http.Get(rawURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: status %d", rawURL, resp.StatusCode)
	}
	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env var %s is not set", key)
	}
	return v
}
```

- [ ] **Step 3: Verify it compiles**

```bash
cd backend && go build ./cmd/ingest/...
```
Expected: binary compiles without errors.

- [ ] **Step 4: Run all tests**

```bash
just test
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
jj describe -m "feat: ingest binary with worker pool and API orchestration"
jj new
```

---

## Task 8: Justfile recipe and env vars

**Files:**
- Modify: `Justfile`
- Modify: `.env.example`

- [ ] **Step 1: Add `ASSETS_DIR` to .env.example**

Add this line to `.env.example`:
```
ASSETS_DIR=./data/assets
```

- [ ] **Step 2: Add ingest recipe to Justfile**

Add after the `frontend` recipe:
```just
# Run the ingestion script (usage: just ingest US-OR)
ingest *args:
    cd backend && go run ./cmd/ingest {{ args }}
```

- [ ] **Step 3: Verify the recipe works (dry-run check)**

```bash
just ingest --help
```
Expected: prints flag usage without error:
```
Usage of /tmp/.../ingest:
  -max-images int
        max images per species (default 3)
  -max-recordings int
        max recordings per species (split evenly between song and call) (default 4)
  -workers int
        concurrent worker count (default 5)
```

- [ ] **Step 4: Commit**

```bash
jj describe -m "chore: add just ingest recipe and ASSETS_DIR to env example"
jj new
```

---

## Notes for live testing

Once you have real API keys in `.env` and the DB is running (`docker compose up -d`):

```bash
just migrate-up       # applies the type column migration if not already done
just ingest US-OR     # Oregon birds -- good for local dev (~200 species)
```

Check results:
```sql
SELECT COUNT(*) FROM species;
SELECT COUNT(*) FROM recordings;
SELECT COUNT(*) FROM species_images;
SELECT common_name, type, quality, file_path FROM recordings LIMIT 10;
```
