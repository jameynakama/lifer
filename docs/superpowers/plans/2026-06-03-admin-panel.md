# Admin Panel Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a `/admin` UI for media CRUD (images + recordings) protected by `RequireAdmin` middleware, with R2 upload/delete and direct DB management.

**Architecture:** Backend adds `RequireAdmin` middleware, a new `r2.Client.Delete` method, four SQL queries, and five admin API handlers under `/api/v1/admin/`. Frontend adds two routes (`/admin`, `/admin/species/[ebird_code]`) with plain forms and fetch calls -- no TanStack Query, reload on success.

**Tech Stack:** Go/chi/sqlc/pgx (backend), SvelteKit adapter-static SPA (frontend), Cloudflare R2 via aws-sdk-go-v2

---

## File Map

**Create:**
- `backend/internal/api/admin.go` -- admin handlers (detail, upload, delete)
- `backend/internal/api/admin_test.go` -- tests for admin handlers
- `frontend/src/routes/admin/+layout.svelte` -- auth guard, redirect non-admins
- `frontend/src/routes/admin/+layout.test.ts` -- redirect test
- `frontend/src/routes/admin/+page.svelte` -- species search + paginated list
- `frontend/src/routes/admin/+page.test.ts` -- render test
- `frontend/src/routes/admin/species/[ebird_code]/+page.svelte` -- images + recordings CRUD
- `frontend/src/routes/admin/species/[ebird_code]/+page.test.ts` -- render test

**Modify:**
- `backend/internal/r2/client.go` -- add `Delete` and `KeyFor` methods
- `backend/internal/r2/client_test.go` -- test `Delete`
- `backend/internal/auth/middleware.go` -- add `RequireAdmin`, export `IsAdminKey()`
- `backend/internal/api/router.go` -- add `R2Client` to `RouterConfig`/`Handler`, mount admin subrouter
- `backend/internal/store/queries/species.sql` -- add 4 queries, update `SearchSpecies`
- `frontend/src/routes/+layout.svelte` -- add Admin nav link
- `frontend/src/stores/auth.ts` -- add `is_admin` to User type

---

## Task 1: r2.Client.Delete and KeyFor methods

**Files:**
- Modify: `backend/internal/r2/client.go`
- Modify: `backend/internal/r2/client_test.go`

- [ ] **Step 1: Write the failing test for Delete**

Add to `backend/internal/r2/client_test.go`:

```go
func TestDelete_SendsDeleteRequest(t *testing.T) {
	var gotMethod, gotPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	c, err := r2.NewWithEndpoint(ts.URL, "key", "secret", "flockdeck", "https://pub.example.com")
	require.NoError(t, err)

	err = c.Delete(context.Background(), "images/sonspa/admin-abc123.jpg")
	require.NoError(t, err)

	assert.Equal(t, http.MethodDelete, gotMethod)
	assert.Contains(t, gotPath, "images/sonspa/admin-abc123.jpg")
}

func TestKeyFor_StripsPublicURL(t *testing.T) {
	c, err := r2.NewWithEndpoint("http://localhost", "key", "secret", "flockdeck", "https://pub.example.com")
	require.NoError(t, err)

	key := c.KeyFor("https://pub.example.com/images/sonspa/admin-abc123.jpg")
	assert.Equal(t, "images/sonspa/admin-abc123.jpg", key)
}
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
cd backend && go test ./internal/r2/... -run "TestDelete|TestKeyFor" -v
```

Expected: FAIL (method not defined)

- [ ] **Step 3: Implement Delete and KeyFor in client.go**

Add to `backend/internal/r2/client.go` after the `Upload` method:

```go
// Delete removes a single object at key from the bucket.
func (c *Client) Delete(ctx context.Context, key string) error {
	_, err := c.s3.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("r2: delete %s: %w", key, err)
	}
	return nil
}

// KeyFor derives the R2 object key from a full public URL by stripping the public URL prefix.
func (c *Client) KeyFor(fileURL string) string {
	return strings.TrimPrefix(fileURL, c.pubURL+"/")
}
```

Add `"strings"` to the import block.

- [ ] **Step 4: Run tests to confirm they pass**

```bash
cd backend && go test ./internal/r2/... -v
```

Expected: PASS (all r2 tests)

- [ ] **Step 5: Commit**

```bash
jj describe -m "feat: add r2.Client.Delete and KeyFor methods" && jj new
```

---

## Task 2: SQL queries and code generation

**Files:**
- Modify: `backend/internal/store/queries/species.sql`
- Run: `just generate` (regenerates `backend/internal/store/species.sql.go`)

- [ ] **Step 1: Update SearchSpecies and add four new queries**

In `backend/internal/store/queries/species.sql`, update `SearchSpecies` and add the four new queries. The full file after changes:

```sql
-- name: SearchSpecies :many
SELECT
    ebird_code,
    common_name,
    scientific_name,
    (SELECT file_path FROM species_images WHERE species_code = species.ebird_code LIMIT 1) AS image_url
FROM species
WHERE common_name ILIKE '%' || $1 || '%'
   OR scientific_name ILIKE '%' || $1 || '%'
   OR ebird_code ILIKE '%' || $1 || '%'
ORDER BY common_name
LIMIT 50;

-- name: ListSpecies :many
SELECT
    ebird_code,
    common_name,
    scientific_name,
    COUNT(*) OVER() AS total_count,
    (SELECT file_path FROM species_images WHERE species_code = species.ebird_code LIMIT 1) AS image_url
FROM species
ORDER BY common_name
LIMIT $1 OFFSET $2;

-- name: GetSpeciesByCode :one
SELECT ebird_code, common_name, scientific_name
FROM species
WHERE ebird_code = $1;

-- name: GetSpeciesRecordings :many
SELECT xeno_canto_id, file_path, quality, type
FROM species_recordings
WHERE species_code = $1
ORDER BY quality, type;

-- name: GetSpeciesImages :many
SELECT macaulay_id, file_path, credit
FROM species_images
WHERE species_code = $1;

-- name: GetDecksForSpecies :many
SELECT deck_id
FROM deck_species
WHERE species_code = $1
  AND deck_id IN (SELECT id FROM decks WHERE owner_id = $2);

-- name: ListAllSpecies :many
SELECT
    ebird_code,
    common_name,
    scientific_name,
    (SELECT file_path FROM species_images WHERE species_code = species.ebird_code LIMIT 1) AS image_url
FROM species
ORDER BY common_name;

-- name: GetImageByID :one
SELECT macaulay_id, species_code, file_path, credit, created_at
FROM species_images
WHERE macaulay_id = $1;

-- name: GetRecordingByID :one
SELECT xeno_canto_id, species_code, file_path, quality, type, created_at
FROM species_recordings
WHERE xeno_canto_id = $1;

-- name: DeleteImage :exec
DELETE FROM species_images WHERE macaulay_id = $1;

-- name: DeleteRecording :exec
DELETE FROM species_recordings WHERE xeno_canto_id = $1;
```

- [ ] **Step 2: Regenerate sqlc types**

```bash
just generate
```

Expected: no errors; `backend/internal/store/species.sql.go` updated with `GetImageByID`, `GetRecordingByID`, `DeleteImage`, `DeleteRecording` functions and updated `SearchSpecies`.

- [ ] **Step 3: Confirm the backend still compiles**

```bash
cd backend && go build ./...
```

Expected: no errors

- [ ] **Step 4: Commit**

```bash
jj describe -m "feat: add admin SQL queries, update SearchSpecies to match ebird_code" && jj new
```

---

## Task 3: RequireAdmin middleware

**Files:**
- Modify: `backend/internal/auth/middleware.go`

- [ ] **Step 1: Write the failing tests**

Create `backend/internal/auth/middleware_test.go`:

```go
package auth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jameynakama/flockdeck/internal/auth"
	"github.com/stretchr/testify/assert"
)

func okHandler(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }

func TestRequireAdmin_AdminUserPasses(t *testing.T) {
	h := auth.RequireAdmin(http.HandlerFunc(okHandler))
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := context.WithValue(r.Context(), auth.UserIDKey(), int64(1))
	ctx = context.WithValue(ctx, auth.IsAdminKey(), true)
	r = r.WithContext(ctx)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequireAdmin_NonAdminReturns403(t *testing.T) {
	h := auth.RequireAdmin(http.HandlerFunc(okHandler))
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := context.WithValue(r.Context(), auth.UserIDKey(), int64(1))
	ctx = context.WithValue(ctx, auth.IsAdminKey(), false)
	r = r.WithContext(ctx)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestRequireAdmin_MissingAdminContextReturns403(t *testing.T) {
	h := auth.RequireAdmin(http.HandlerFunc(okHandler))
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)

	assert.Equal(t, http.StatusForbidden, w.Code)
}
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
cd backend && go test ./internal/auth/... -v
```

Expected: FAIL (IsAdminKey and RequireAdmin not defined)

- [ ] **Step 3: Add IsAdminKey export and RequireAdmin to middleware.go**

Add to `backend/internal/auth/middleware.go` after the existing `IsAdminFromCtx` function:

```go
// IsAdminKey returns the context key used by RequireAuth so tests can inject isAdmin.
func IsAdminKey() any {
	return isAdminKey
}

// RequireAdmin returns 403 if the request context does not have isAdmin=true.
// Must be used after RequireAuth.
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !IsAdminFromCtx(r.Context()) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
```

- [ ] **Step 4: Run tests to confirm they pass**

```bash
cd backend && go test ./internal/auth/... -v
```

Expected: PASS (3 tests)

- [ ] **Step 5: Commit**

```bash
jj describe -m "feat: add RequireAdmin middleware and export IsAdminKey" && jj new
```

---

## Task 4: Extend Handler with r2Client and wire admin routes

**Files:**
- Modify: `backend/internal/api/router.go`

- [ ] **Step 1: Add R2Client to RouterConfig and Handler**

In `backend/internal/api/router.go`, update `RouterConfig` and `Handler`:

```go
type RouterConfig struct {
	Queries     store.Querier
	OAuthConfig *oauth2.Config
	JWTSecret   []byte
	FrontendURL string
	R2Client    *r2.Client
}

type Handler struct {
	queries     store.Querier
	oauthConfig *oauth2.Config
	jwtSecret   []byte
	frontendURL string
	r2Client    *r2.Client
}
```

Update the `NewRouter` function body to assign r2Client:

```go
h := &Handler{
    queries:     cfg.Queries,
    oauthConfig: cfg.OAuthConfig,
    jwtSecret:   cfg.JWTSecret,
    frontendURL: cfg.FrontendURL,
    r2Client:    cfg.R2Client,
}
```

Add the admin subrouter at the end of the `/api/v1` route block, after the existing group:

```go
r.With(auth.RequireAuth(cfg.JWTSecret), auth.RequireAdmin).Route("/admin", func(r chi.Router) {
    r.Get("/species/{ebird_code}", h.adminGetSpeciesDetail)
    r.Post("/species/{ebird_code}/images", h.adminUploadImage)
    r.Post("/species/{ebird_code}/recordings", h.adminUploadRecording)
    r.Delete("/species/{ebird_code}/images/{macaulay_id}", h.adminDeleteImage)
    r.Delete("/species/{ebird_code}/recordings/{xeno_canto_id}", h.adminDeleteRecording)
})
```

Add import for `"github.com/jameynakama/flockdeck/internal/r2"` to the import block.

- [ ] **Step 2: Add stub handlers in admin.go so it compiles**

Create `backend/internal/api/admin.go`:

```go
package api

import (
	"net/http"
)

func (h *Handler) adminGetSpeciesDetail(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func (h *Handler) adminUploadImage(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func (h *Handler) adminUploadRecording(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func (h *Handler) adminDeleteImage(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func (h *Handler) adminDeleteRecording(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}
```

- [ ] **Step 3: Update cmd/server/main.go to construct and pass R2Client**

The server doesn't currently build an r2.Client. Add R2 env vars to `config` and wire it up.

In `backend/cmd/server/main.go`, add to the `config` struct:

```go
r2AccountID    string
r2AccessKey    string
r2SecretKey    string
r2Bucket       string
r2PublicURL    string
```

Add to `loadConfig()` return value:

```go
r2AccountID: required("R2_ACCOUNT_ID"),
r2AccessKey:  required("R2_ACCESS_KEY_ID"),
r2SecretKey:  required("R2_SECRET_ACCESS_KEY"),
r2Bucket:     required("R2_BUCKET_NAME"),
r2PublicURL:  required("R2_PUBLIC_URL"),
```

In `main()`, after constructing `queries`, add:

```go
r2c, err := r2.New(cfg.r2AccountID, cfg.r2AccessKey, cfg.r2SecretKey, cfg.r2Bucket, cfg.r2PublicURL)
if err != nil {
    log.Fatalf("r2 client: %v", err)
}
```

Update `api.NewRouter(api.RouterConfig{...})` to include:

```go
R2Client: r2c,
```

Add `"github.com/jameynakama/flockdeck/internal/r2"` to the import block.

- [ ] **Step 4: Confirm it compiles**

```bash
cd backend && go build ./...
```

Expected: no errors

- [ ] **Step 5: Run existing tests to confirm nothing is broken**

```bash
cd backend && go test ./...
```

Expected: all existing tests PASS

- [ ] **Step 6: Commit**

```bash
jj describe -m "feat: add R2Client to Handler, wire admin subrouter with stub handlers" && jj new
```

---

## Task 5: Admin GET detail handler

**Files:**
- Modify: `backend/internal/api/admin.go`
- Modify: `backend/internal/api/admin_test.go`

- [ ] **Step 1: Write the failing test**

Create `backend/internal/api/admin_test.go`:

```go
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jameynakama/flockdeck/internal/auth"
	"github.com/jameynakama/flockdeck/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type adminStubQuerier struct {
	store.Querier
	getImageByID        func(ctx context.Context, macaulayID string) (store.SpeciesImage, error)
	getRecordingByID    func(ctx context.Context, xenoCantoID string) (store.SpeciesRecording, error)
	getSpeciesImages    func(ctx context.Context, speciesCode string) ([]store.GetSpeciesImagesRow, error)
	getSpeciesRecordings func(ctx context.Context, speciesCode string) ([]store.GetSpeciesRecordingsRow, error)
	deleteImage         func(ctx context.Context, macaulayID string) error
	deleteRecording     func(ctx context.Context, xenoCantoID string) error
	upsertSpeciesImage  func(ctx context.Context, arg store.UpsertSpeciesImageParams) (store.SpeciesImage, error)
	upsertRecording     func(ctx context.Context, arg store.UpsertRecordingParams) (store.SpeciesRecording, error)
}

func (s *adminStubQuerier) GetImageByID(ctx context.Context, macaulayID string) (store.SpeciesImage, error) {
	return s.getImageByID(ctx, macaulayID)
}
func (s *adminStubQuerier) GetRecordingByID(ctx context.Context, xenoCantoID string) (store.SpeciesRecording, error) {
	return s.getRecordingByID(ctx, xenoCantoID)
}
func (s *adminStubQuerier) GetSpeciesImages(ctx context.Context, speciesCode string) ([]store.GetSpeciesImagesRow, error) {
	return s.getSpeciesImages(ctx, speciesCode)
}
func (s *adminStubQuerier) GetSpeciesRecordings(ctx context.Context, speciesCode string) ([]store.GetSpeciesRecordingsRow, error) {
	return s.getSpeciesRecordings(ctx, speciesCode)
}
func (s *adminStubQuerier) DeleteImage(ctx context.Context, macaulayID string) error {
	return s.deleteImage(ctx, macaulayID)
}
func (s *adminStubQuerier) DeleteRecording(ctx context.Context, xenoCantoID string) error {
	return s.deleteRecording(ctx, xenoCantoID)
}
func (s *adminStubQuerier) UpsertSpeciesImage(ctx context.Context, arg store.UpsertSpeciesImageParams) (store.SpeciesImage, error) {
	return s.upsertSpeciesImage(ctx, arg)
}
func (s *adminStubQuerier) UpsertRecording(ctx context.Context, arg store.UpsertRecordingParams) (store.SpeciesRecording, error) {
	return s.upsertRecording(ctx, arg)
}

// injectAdmin sets userID and isAdmin=true in the request context.
func injectAdmin(r *http.Request, userID int64) *http.Request {
	ctx := context.WithValue(r.Context(), auth.UserIDKey(), userID)
	ctx = context.WithValue(ctx, auth.IsAdminKey(), true)
	return r.WithContext(ctx)
}

func TestAdminGetSpeciesDetail_ReturnsImagesAndRecordings(t *testing.T) {
	q := &adminStubQuerier{
		getSpeciesImages: func(_ context.Context, code string) ([]store.GetSpeciesImagesRow, error) {
			assert.Equal(t, "sonspa", code)
			return []store.GetSpeciesImagesRow{
				{MacaulayID: "img1", FilePath: "https://pub.example.com/images/sonspa/img1.jpg", Credit: "Photographer"},
			}, nil
		},
		getSpeciesRecordings: func(_ context.Context, code string) ([]store.GetSpeciesRecordingsRow, error) {
			assert.Equal(t, "sonspa", code)
			return []store.GetSpeciesRecordingsRow{
				{XenoCantoID: "rec1", FilePath: "https://pub.example.com/recordings/sonspa/rec1.mp3", Quality: "A", Type: "song"},
			}, nil
		},
	}
	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/species/sonspa", nil)
	r = injectAdmin(r, 1)
	r = withChiParam(r, "ebird_code", "sonspa")
	w := httptest.NewRecorder()

	h.adminGetSpeciesDetail(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Len(t, body["images"], 1)
	assert.Len(t, body["recordings"], 1)
}
```

- [ ] **Step 2: Run the test to confirm it fails**

```bash
cd backend && go test ./internal/api/... -run TestAdminGetSpeciesDetail -v
```

Expected: FAIL (stub methods don't match generated types yet -- adjust type names from `store.GetSpeciesImagesRow` etc. to match what `just generate` actually produced)

- [ ] **Step 3: Check the generated types**

```bash
grep -n "GetSpeciesImages\|GetSpeciesRecordings" backend/internal/store/species.sql.go | head -10
```

Adjust the test stub types to match the actual generated return types (they may be `store.SpeciesImage`/`store.SpeciesRecording` directly or row types). Update the stub accordingly.

- [ ] **Step 4: Implement adminGetSpeciesDetail in admin.go**

Replace the stub in `backend/internal/api/admin.go`:

```go
package api

import (
	"crypto/rand"
	"fmt"
	"log"
	"net/http"
	"path/filepath"

	"github.com/go-chi/chi/v5"
)

type adminSpeciesDetailResponse struct {
	Images     any `json:"images"`
	Recordings any `json:"recordings"`
}

func (h *Handler) adminGetSpeciesDetail(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "ebird_code")

	images, err := h.queries.GetSpeciesImages(r.Context(), code)
	if err != nil {
		log.Printf("admin: get images for %s: %v", code, err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	recordings, err := h.queries.GetSpeciesRecordings(r.Context(), code)
	if err != nil {
		log.Printf("admin: get recordings for %s: %v", code, err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, adminSpeciesDetailResponse{
		Images:     images,
		Recordings: recordings,
	})
}

func adminID() string {
	b := make([]byte, 8)
	rand.Read(b) //nolint:errcheck
	return fmt.Sprintf("admin-%x", b)
}

func (h *Handler) adminUploadImage(w http.ResponseWriter, r *http.Request)      { http.Error(w, "not implemented", http.StatusNotImplemented) }
func (h *Handler) adminUploadRecording(w http.ResponseWriter, r *http.Request)  { http.Error(w, "not implemented", http.StatusNotImplemented) }
func (h *Handler) adminDeleteImage(w http.ResponseWriter, r *http.Request)      { http.Error(w, "not implemented", http.StatusNotImplemented) }
func (h *Handler) adminDeleteRecording(w http.ResponseWriter, r *http.Request)  { http.Error(w, "not implemented", http.StatusNotImplemented) }
```

- [ ] **Step 5: Run tests to confirm they pass**

```bash
cd backend && go test ./internal/api/... -run TestAdminGetSpeciesDetail -v
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
jj describe -m "feat: implement adminGetSpeciesDetail handler" && jj new
```

---

## Task 6: Admin DELETE handlers

**Files:**
- Modify: `backend/internal/api/admin.go`
- Modify: `backend/internal/api/admin_test.go`

- [ ] **Step 1: Write failing tests**

Add to `backend/internal/api/admin_test.go`:

```go
func TestAdminDeleteImage_DeletesFromR2AndDB(t *testing.T) {
	var deletedKey string
	var deletedID string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deletedKey = r.URL.Path
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	r2c, err := r2.NewWithEndpoint(ts.URL, "key", "secret", "flockdeck", "https://pub.example.com")
	require.NoError(t, err)

	q := &adminStubQuerier{
		getImageByID: func(_ context.Context, id string) (store.SpeciesImage, error) {
			return store.SpeciesImage{
				MacaulayID: id,
				FilePath:   "https://pub.example.com/images/sonspa/img1.jpg",
			}, nil
		},
		deleteImage: func(_ context.Context, id string) error {
			deletedID = id
			return nil
		},
	}
	h := &Handler{queries: q, r2Client: r2c}
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/species/sonspa/images/img1", nil)
	req = injectAdmin(req, 1)
	req = withChiParams(req, map[string]string{"ebird_code": "sonspa", "macaulay_id": "img1"})
	w := httptest.NewRecorder()

	h.adminDeleteImage(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "img1", deletedID)
	assert.Contains(t, deletedKey, "images/sonspa/img1.jpg")
}

func TestAdminDeleteRecording_DeletesFromR2AndDB(t *testing.T) {
	var deletedKey string
	var deletedID string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deletedKey = r.URL.Path
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	r2c, err := r2.NewWithEndpoint(ts.URL, "key", "secret", "flockdeck", "https://pub.example.com")
	require.NoError(t, err)

	q := &adminStubQuerier{
		getRecordingByID: func(_ context.Context, id string) (store.SpeciesRecording, error) {
			return store.SpeciesRecording{
				XenoCantoID: id,
				FilePath:    "https://pub.example.com/recordings/sonspa/rec1.mp3",
			}, nil
		},
		deleteRecording: func(_ context.Context, id string) error {
			deletedID = id
			return nil
		},
	}
	h := &Handler{queries: q, r2Client: r2c}
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/species/sonspa/recordings/rec1", nil)
	req = injectAdmin(req, 1)
	req = withChiParams(req, map[string]string{"ebird_code": "sonspa", "xeno_canto_id": "rec1"})
	w := httptest.NewRecorder()

	h.adminDeleteRecording(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "rec1", deletedID)
	assert.Contains(t, deletedKey, "recordings/sonspa/rec1.mp3")
}
```

Add `"github.com/jameynakama/flockdeck/internal/r2"` to the admin_test.go imports.

- [ ] **Step 2: Run tests to confirm they fail**

```bash
cd backend && go test ./internal/api/... -run "TestAdminDelete" -v
```

Expected: FAIL (not implemented)

- [ ] **Step 3: Implement adminDeleteImage and adminDeleteRecording**

Replace the stub one-liners in `backend/internal/api/admin.go`:

```go
func (h *Handler) adminDeleteImage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "macaulay_id")

	img, err := h.queries.GetImageByID(r.Context(), id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err := h.r2Client.Delete(r.Context(), h.r2Client.KeyFor(img.FilePath)); err != nil {
		log.Printf("admin: delete image R2 %s: %v", id, err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	if err := h.queries.DeleteImage(r.Context(), id); err != nil {
		log.Printf("admin: delete image DB %s: %v", id, err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) adminDeleteRecording(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "xeno_canto_id")

	rec, err := h.queries.GetRecordingByID(r.Context(), id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err := h.r2Client.Delete(r.Context(), h.r2Client.KeyFor(rec.FilePath)); err != nil {
		log.Printf("admin: delete recording R2 %s: %v", id, err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	if err := h.queries.DeleteRecording(r.Context(), id); err != nil {
		log.Printf("admin: delete recording DB %s: %v", id, err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
```

- [ ] **Step 4: Run tests to confirm they pass**

```bash
cd backend && go test ./internal/api/... -run "TestAdminDelete" -v
```

Expected: PASS

- [ ] **Step 5: Run all tests**

```bash
cd backend && go test ./...
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
jj describe -m "feat: implement adminDeleteImage and adminDeleteRecording handlers" && jj new
```

---

## Task 7: Admin POST upload handlers

**Files:**
- Modify: `backend/internal/api/admin.go`
- Modify: `backend/internal/api/admin_test.go`

- [ ] **Step 1: Write failing tests**

Add to `backend/internal/api/admin_test.go`:

```go
func TestAdminUploadImage_UploadsToR2AndInsertsDB(t *testing.T) {
	var uploadedKey, uploadedContentType string
	var insertedParams store.UpsertSpeciesImageParams

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uploadedKey = r.URL.Path
		uploadedContentType = r.Header.Get("Content-Type")
		w.Header().Set("ETag", `"abc"`)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	r2c, err := r2.NewWithEndpoint(ts.URL, "key", "secret", "flockdeck", "https://pub.example.com")
	require.NoError(t, err)

	q := &adminStubQuerier{
		upsertSpeciesImage: func(_ context.Context, arg store.UpsertSpeciesImageParams) (store.SpeciesImage, error) {
			insertedParams = arg
			return store.SpeciesImage{MacaulayID: arg.MacaulayID}, nil
		},
	}
	h := &Handler{queries: q, r2Client: r2c}

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", "sparrow.jpg")
	fw.Write([]byte("fake image data"))
	mw.WriteField("credit", "John Doe")
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/species/sonspa/images", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req = injectAdmin(req, 1)
	req = withChiParam(req, "ebird_code", "sonspa")
	w := httptest.NewRecorder()

	h.adminUploadImage(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, uploadedKey, "images/sonspa/")
	assert.Contains(t, uploadedKey, ".jpg")
	assert.Equal(t, "image/jpeg", uploadedContentType)
	assert.Equal(t, "sonspa", insertedParams.SpeciesCode)
	assert.Equal(t, "John Doe", insertedParams.Credit)
	assert.HasPrefix(t, insertedParams.MacaulayID, "admin-")
}

func TestAdminUploadRecording_UploadsToR2AndInsertsDB(t *testing.T) {
	var uploadedKey string
	var insertedParams store.UpsertRecordingParams

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uploadedKey = r.URL.Path
		w.Header().Set("ETag", `"abc"`)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	r2c, err := r2.NewWithEndpoint(ts.URL, "key", "secret", "flockdeck", "https://pub.example.com")
	require.NoError(t, err)

	q := &adminStubQuerier{
		upsertRecording: func(_ context.Context, arg store.UpsertRecordingParams) (store.SpeciesRecording, error) {
			insertedParams = arg
			return store.SpeciesRecording{XenoCantoID: arg.XenoCantoID}, nil
		},
	}
	h := &Handler{queries: q, r2Client: r2c}

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", "sparrow.mp3")
	fw.Write([]byte("fake audio data"))
	mw.WriteField("quality", "A")
	mw.WriteField("type", "song")
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/species/sonspa/recordings", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req = injectAdmin(req, 1)
	req = withChiParam(req, "ebird_code", "sonspa")
	w := httptest.NewRecorder()

	h.adminUploadRecording(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, uploadedKey, "recordings/sonspa/")
	assert.Contains(t, uploadedKey, ".mp3")
	assert.Equal(t, "sonspa", insertedParams.SpeciesCode)
	assert.Equal(t, "A", insertedParams.Quality)
	assert.Equal(t, "song", insertedParams.Type)
	assert.HasPrefix(t, insertedParams.XenoCantoID, "admin-")
}
```

Add `"bytes"` and `"mime/multipart"` to the imports in admin_test.go.

Note: `assert.HasPrefix` may not exist in testify -- use `assert.True(t, strings.HasPrefix(insertedParams.MacaulayID, "admin-"))` with `"strings"` imported instead.

- [ ] **Step 2: Run tests to confirm they fail**

```bash
cd backend && go test ./internal/api/... -run "TestAdminUpload" -v
```

Expected: FAIL (not implemented)

- [ ] **Step 3: Map of content types by extension**

Add to `backend/internal/api/admin.go`:

```go
var extContentType = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".webp": "image/webp",
	".mp3":  "audio/mpeg",
	".ogg":  "audio/ogg",
	".wav":  "audio/wav",
	".flac": "audio/flac",
}
```

- [ ] **Step 4: Implement adminUploadImage**

Replace the stub in `backend/internal/api/admin.go`:

```go
func (h *Handler) adminUploadImage(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "ebird_code")
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	credit := r.FormValue("credit")
	if credit == "" {
		credit = "admin upload"
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	contentType := extContentType[ext]
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	id := adminID()
	key := fmt.Sprintf("images/%s/%s%s", code, id, ext)
	fileURL, err := h.r2Client.Upload(r.Context(), key, contentType, file)
	if err != nil {
		log.Printf("admin: upload image R2 %s: %v", key, err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	img, err := h.queries.UpsertSpeciesImage(r.Context(), store.UpsertSpeciesImageParams{
		MacaulayID:  id,
		SpeciesCode: code,
		FilePath:    fileURL,
		Credit:      credit,
	})
	if err != nil {
		log.Printf("admin: insert image DB %s: %v", id, err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, img)
}
```

Add `"strings"` and `"github.com/jameynakama/flockdeck/internal/store"` to imports if not already present.

- [ ] **Step 5: Implement adminUploadRecording**

Replace the stub:

```go
func (h *Handler) adminUploadRecording(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "ebird_code")
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	quality := r.FormValue("quality")
	if quality == "" {
		quality = "A"
	}
	recType := r.FormValue("type")
	if recType == "" {
		recType = "song"
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	contentType := extContentType[ext]
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	id := adminID()
	key := fmt.Sprintf("recordings/%s/%s%s", code, id, ext)
	fileURL, err := h.r2Client.Upload(r.Context(), key, contentType, file)
	if err != nil {
		log.Printf("admin: upload recording R2 %s: %v", key, err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	rec, err := h.queries.UpsertRecording(r.Context(), store.UpsertRecordingParams{
		XenoCantoID: id,
		SpeciesCode: code,
		FilePath:    fileURL,
		Quality:     quality,
		Type:        recType,
	})
	if err != nil {
		log.Printf("admin: insert recording DB %s: %v", id, err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, rec)
}
```

- [ ] **Step 6: Run upload tests**

```bash
cd backend && go test ./internal/api/... -run "TestAdminUpload" -v
```

Expected: PASS

- [ ] **Step 7: Run all backend tests**

```bash
cd backend && go test ./...
```

Expected: all PASS

- [ ] **Step 8: Commit**

```bash
jj describe -m "feat: implement adminUploadImage and adminUploadRecording handlers" && jj new
```

---

## Task 8: Frontend auth User type and admin nav link

**Files:**
- Modify: `frontend/src/stores/auth.ts`
- Modify: `frontend/src/routes/+layout.svelte`
- Modify: `frontend/src/routes/layout.test.ts`

- [ ] **Step 1: Add is_admin to User type**

Update `frontend/src/stores/auth.ts`:

```ts
import { writable, type Writable } from "svelte/store";

interface User {
    id: number;
    email: string;
    name: string;
    is_admin: boolean;
}

type Auth = User | null;

export const auth: Writable<Auth> = writable(null);
```

- [ ] **Step 2: Add Admin nav link in layout**

In `frontend/src/routes/+layout.svelte`, update the `<nav>` block:

```svelte
<nav>
  <a href="/decks">Decks</a>
  <a href="/explore">Explore</a>
  {#if $auth?.is_admin}
    <a href="/admin">Admin</a>
  {/if}
</nav>
```

- [ ] **Step 3: Add test for admin nav link**

Add to `frontend/src/routes/layout.test.ts`:

```ts
it('shows Admin link when user is admin', async () => {
  const user = { id: 1, email: 'test@example.com', name: 'Test User', is_admin: true }
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve(user) }))
  render(Layout)
  await vi.waitFor(() => {
    expect(screen.getByRole('link', { name: /admin/i })).toBeInTheDocument()
  })
})

it('hides Admin link for non-admin users', async () => {
  const user = { id: 1, email: 'test@example.com', name: 'Test User', is_admin: false }
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve(user) }))
  render(Layout)
  await vi.waitFor(() => {
    expect(screen.getByText('FlockDeck')).toBeInTheDocument()
  })
  expect(screen.queryByRole('link', { name: /admin/i })).not.toBeInTheDocument()
})
```

- [ ] **Step 4: Run frontend tests**

```bash
cd frontend && npm test
```

Expected: PASS (existing + new layout tests)

- [ ] **Step 5: Commit**

```bash
jj describe -m "feat: add is_admin to User type, add Admin nav link for admin users" && jj new
```

---

## Task 9: Frontend admin layout (auth guard)

**Files:**
- Create: `frontend/src/routes/admin/+layout.svelte`
- Create: `frontend/src/routes/admin/+layout.test.ts`

- [ ] **Step 1: Write the failing test**

Create `frontend/src/routes/admin/+layout.test.ts`:

```ts
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/svelte'
import { auth } from '$stores/auth'
import Layout from './+layout.svelte'

const mockGoto = vi.fn()
vi.mock('$app/navigation', () => ({ goto: mockGoto }))

beforeEach(() => {
  auth.set(null)
  mockGoto.mockClear()
})

describe('Admin Layout', () => {
  it('redirects non-admin users to /', async () => {
    auth.set({ id: 1, email: 'test@example.com', name: 'Test', is_admin: false })
    render(Layout)
    await vi.waitFor(() => {
      expect(mockGoto).toHaveBeenCalledWith('/')
    })
  })

  it('redirects unauthenticated users to /', async () => {
    auth.set(null)
    render(Layout)
    await vi.waitFor(() => {
      expect(mockGoto).toHaveBeenCalledWith('/')
    })
  })

  it('renders children for admin users', async () => {
    auth.set({ id: 1, email: 'admin@example.com', name: 'Admin', is_admin: true })
    render(Layout)
    await vi.waitFor(() => {
      expect(screen.getByText(/admin/i)).toBeInTheDocument()
    })
  })
})
```

- [ ] **Step 2: Run test to confirm it fails**

```bash
cd frontend && npm test -- --run src/routes/admin/+layout.test.ts
```

Expected: FAIL (file not found)

- [ ] **Step 3: Create the admin layout**

Create `frontend/src/routes/admin/+layout.svelte`:

```svelte
<script lang="ts">
  import { goto } from '$app/navigation'
  import { auth } from '$stores/auth'

  let { children } = $props()

  $effect(() => {
    if (!$auth?.is_admin) {
      goto('/')
    }
  })
</script>

{#if $auth?.is_admin}
  <div class="admin-container">
    <h1>Admin</h1>
    {@render children?.()}
  </div>
{/if}

<style>
  .admin-container {
    max-width: 900px;
    margin: 0 auto;
    padding: 1rem 1.5rem;
  }
</style>
```

- [ ] **Step 4: Run tests**

```bash
cd frontend && npm test
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
jj describe -m "feat: add admin layout with auth guard" && jj new
```

---

## Task 10: Frontend admin search page

**Files:**
- Create: `frontend/src/routes/admin/+page.svelte`
- Create: `frontend/src/routes/admin/+page.test.ts`

- [ ] **Step 1: Write the failing test**

Create `frontend/src/routes/admin/+page.test.ts`:

```ts
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/svelte'
import { auth } from '$stores/auth'
import Page from './+page.svelte'

beforeEach(() => {
  auth.set({ id: 1, email: 'admin@example.com', name: 'Admin', is_admin: true })
})

describe('Admin search page', () => {
  it('renders a search form', () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve({ results: [], count: 0, next: null, previous: null }) }))
    render(Page)
    expect(screen.getByRole('textbox')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /search/i })).toBeInTheDocument()
  })

  it('renders species results as links', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({
        results: [{ ebird_code: 'sonspa', common_name: 'Song Sparrow', scientific_name: 'Melospiza melodia', image_url: null }],
        count: 1,
        next: null,
        previous: null,
      }),
    }))
    render(Page)
    await vi.waitFor(() => {
      expect(screen.getByRole('link', { name: /song sparrow/i })).toBeInTheDocument()
    })
  })
})
```

- [ ] **Step 2: Run test to confirm it fails**

```bash
cd frontend && npm test -- --run src/routes/admin/+page.test.ts
```

Expected: FAIL

- [ ] **Step 3: Create the admin search page**

Create `frontend/src/routes/admin/+page.svelte`:

```svelte
<script lang="ts">
  import { page } from '$app/state'

  interface SpeciesResult {
    ebird_code: string
    common_name: string
    scientific_name: string
    image_url: string | null
  }

  interface SpeciesPage {
    results: SpeciesResult[]
    count: number
    next: string | null
    previous: string | null
  }

  let results: SpeciesResult[] = $state([])
  let count = $state(0)
  let next: string | null = $state(null)
  let previous: string | null = $state(null)
  let loading = $state(false)

  const q = $derived(page.url.searchParams.get('q') ?? '')
  const offset = $derived(Number(page.url.searchParams.get('offset') ?? '0'))
  const limit = 25

  $effect(() => {
    const params = new URLSearchParams({ limit: String(limit), offset: String(offset) })
    if (q) params.set('q', q)
    loading = true
    fetch(`/api/v1/species?${params}`)
      .then(r => r.json())
      .then((data: SpeciesPage) => {
        results = data.results
        count = data.count
        next = data.next
        previous = data.previous
      })
      .catch(() => {})
      .finally(() => { loading = false })
  })
</script>

<form method="GET">
  <input type="text" name="q" value={q} placeholder="Search species by name or code" />
  <button type="submit">Search</button>
</form>

<p>{count} species{q ? ` matching "${q}"` : ''}</p>

{#if loading}
  <p>Loading...</p>
{:else}
  <ul>
    {#each results as sp}
      <li>
        <a href="/admin/species/{sp.ebird_code}">
          {sp.common_name} <span class="muted">({sp.ebird_code})</span>
        </a>
      </li>
    {/each}
  </ul>
{/if}

<div class="pagination">
  {#if previous}
    <a href="?q={q}&offset={offset - limit}">← Previous</a>
  {/if}
  {#if next}
    <a href="?q={q}&offset={offset + limit}">Next →</a>
  {/if}
</div>

<style>
  form { display: flex; gap: 0.5rem; margin-bottom: 1rem; }
  input { flex: 1; padding: 0.4rem 0.6rem; }
  ul { list-style: none; padding: 0; }
  li { padding: 0.4rem 0; border-bottom: 1px solid var(--border); }
  a { color: var(--text); text-decoration: none; }
  a:hover { color: var(--text-secondary); }
  .muted { color: var(--text-muted); font-size: 0.875rem; }
  .pagination { display: flex; gap: 1rem; margin-top: 1rem; }
</style>
```

- [ ] **Step 4: Run tests**

```bash
cd frontend && npm test
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
jj describe -m "feat: add admin species search page" && jj new
```

---

## Task 11: Frontend admin species detail page

**Files:**
- Create: `frontend/src/routes/admin/species/[ebird_code]/+page.svelte`
- Create: `frontend/src/routes/admin/species/[ebird_code]/+page.test.ts`

- [ ] **Step 1: Write the failing test**

Create `frontend/src/routes/admin/species/[ebird_code]/+page.test.ts`:

```ts
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/svelte'
import { auth } from '$stores/auth'
import Page from './+page.svelte'

beforeEach(() => {
  auth.set({ id: 1, email: 'admin@example.com', name: 'Admin', is_admin: true })
})

describe('Admin species detail page', () => {
  it('renders images and recordings sections', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({
        images: [{ macaulay_id: 'img1', file_path: 'https://example.com/img1.jpg', credit: 'Photographer' }],
        recordings: [{ xeno_canto_id: 'rec1', file_path: 'https://example.com/rec1.mp3', quality: 'A', type: 'song' }],
      }),
    }))
    render(Page)
    await vi.waitFor(() => {
      expect(screen.getByText(/images/i)).toBeInTheDocument()
      expect(screen.getByText(/recordings/i)).toBeInTheDocument()
    })
  })
})
```

- [ ] **Step 2: Run test to confirm it fails**

```bash
cd frontend && npm test -- --run "src/routes/admin/species"
```

Expected: FAIL

- [ ] **Step 3: Create the admin species detail page**

Create `frontend/src/routes/admin/species/[ebird_code]/+page.svelte`:

```svelte
<script lang="ts">
  import { page } from '$app/state'

  interface SpeciesImage {
    macaulay_id: string
    file_path: string
    credit: string
  }

  interface SpeciesRecording {
    xeno_canto_id: string
    file_path: string
    quality: string
    type: string
  }

  const ebirdCode = $derived(page.params.ebird_code)

  let images: SpeciesImage[] = $state([])
  let recordings: SpeciesRecording[] = $state([])
  let loading = $state(true)
  let error = $state('')

  async function loadDetail() {
    loading = true
    error = ''
    try {
      const res = await fetch(`/api/v1/admin/species/${ebirdCode}`)
      if (!res.ok) throw new Error('Failed to load')
      const data = await res.json()
      images = data.images ?? []
      recordings = data.recordings ?? []
    } catch {
      error = 'Failed to load species detail.'
    } finally {
      loading = false
    }
  }

  $effect(() => { loadDetail() })

  async function deleteImage(macaulayID: string) {
    if (!confirm(`Delete image ${macaulayID}?`)) return
    const res = await fetch(`/api/v1/admin/species/${ebirdCode}/images/${macaulayID}`, { method: 'DELETE' })
    if (res.ok) {
      images = images.filter(i => i.macaulay_id !== macaulayID)
    } else {
      alert('Delete failed')
    }
  }

  async function deleteRecording(xenoCantoID: string) {
    if (!confirm(`Delete recording ${xenoCantoID}?`)) return
    const res = await fetch(`/api/v1/admin/species/${ebirdCode}/recordings/${xenoCantoID}`, { method: 'DELETE' })
    if (res.ok) {
      recordings = recordings.filter(r => r.xeno_canto_id !== xenoCantoID)
    } else {
      alert('Delete failed')
    }
  }

  async function uploadImage(e: SubmitEvent) {
    e.preventDefault()
    const form = e.target as HTMLFormElement
    const res = await fetch(`/api/v1/admin/species/${ebirdCode}/images`, {
      method: 'POST',
      body: new FormData(form),
    })
    if (res.ok) {
      const img: SpeciesImage = await res.json()
      images = [...images, img]
      form.reset()
    } else {
      alert('Upload failed')
    }
  }

  async function uploadRecording(e: SubmitEvent) {
    e.preventDefault()
    const form = e.target as HTMLFormElement
    const res = await fetch(`/api/v1/admin/species/${ebirdCode}/recordings`, {
      method: 'POST',
      body: new FormData(form),
    })
    if (res.ok) {
      const rec: SpeciesRecording = await res.json()
      recordings = [...recordings, rec]
      form.reset()
    } else {
      alert('Upload failed')
    }
  }
</script>

<a href="/admin">← Back to search</a>
<h2>{ebirdCode}</h2>

{#if loading}
  <p>Loading...</p>
{:else if error}
  <p class="error">{error}</p>
{:else}
  <section>
    <h3>Images ({images.length})</h3>
    <div class="image-grid">
      {#each images as img}
        <div class="image-card">
          <img src={img.file_path} alt={img.credit} />
          <p class="credit">{img.credit}</p>
          <p class="id">{img.macaulay_id}</p>
          <button onclick={() => deleteImage(img.macaulay_id)}>Delete</button>
        </div>
      {/each}
    </div>

    <details>
      <summary>Upload new image</summary>
      <form onsubmit={uploadImage} enctype="multipart/form-data">
        <label>File: <input type="file" name="file" accept="image/*" required /></label>
        <label>Credit: <input type="text" name="credit" placeholder="Photographer name" /></label>
        <button type="submit">Upload</button>
      </form>
    </details>
  </section>

  <section>
    <h3>Recordings ({recordings.length})</h3>
    <table>
      <thead>
        <tr><th>ID</th><th>Quality</th><th>Type</th><th>Actions</th></tr>
      </thead>
      <tbody>
        {#each recordings as rec}
          <tr>
            <td>{rec.xeno_canto_id}</td>
            <td>{rec.quality}</td>
            <td>{rec.type}</td>
            <td>
              <audio src={rec.file_path} controls></audio>
              <button onclick={() => deleteRecording(rec.xeno_canto_id)}>Delete</button>
            </td>
          </tr>
        {/each}
      </tbody>
    </table>

    <details>
      <summary>Upload new recording</summary>
      <form onsubmit={uploadRecording} enctype="multipart/form-data">
        <label>File: <input type="file" name="file" accept="audio/*" required /></label>
        <label>
          Quality:
          <select name="quality">
            <option>A</option><option>B</option><option>C</option><option>D</option><option>E</option>
          </select>
        </label>
        <label>
          Type:
          <select name="type">
            <option>song</option><option>call</option>
          </select>
        </label>
        <button type="submit">Upload</button>
      </form>
    </details>
  </section>
{/if}

<style>
  section { margin-top: 2rem; }
  .image-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(160px, 1fr)); gap: 1rem; margin: 1rem 0; }
  .image-card { border: 1px solid var(--border); border-radius: 6px; padding: 0.5rem; }
  .image-card img { width: 100%; aspect-ratio: 1; object-fit: cover; border-radius: 4px; }
  .credit, .id { font-size: 0.75rem; color: var(--text-muted); margin: 0.25rem 0; }
  table { width: 100%; border-collapse: collapse; }
  th, td { text-align: left; padding: 0.5rem; border-bottom: 1px solid var(--border); font-size: 0.875rem; }
  audio { height: 28px; vertical-align: middle; }
  details { margin-top: 1rem; }
  summary { cursor: pointer; color: var(--text-secondary); }
  form { display: flex; flex-direction: column; gap: 0.5rem; margin-top: 0.75rem; max-width: 400px; }
  label { display: flex; flex-direction: column; gap: 0.25rem; font-size: 0.875rem; }
  .error { color: var(--danger, red); }
</style>
```

- [ ] **Step 4: Run all frontend tests**

```bash
cd frontend && npm test
```

Expected: PASS

- [ ] **Step 5: Run all backend tests one final time**

```bash
cd backend && go test ./...
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
jj describe -m "feat: add admin species detail page with image/recording CRUD" && jj new
```

---

## Done

All admin panel functionality is implemented:
- `RequireAdmin` middleware protects `/api/v1/admin/*`
- Images and recordings can be viewed, deleted, and uploaded via the admin UI
- R2 and DB stay in sync on all operations
- `/admin` is only accessible to users with `is_admin: true`
