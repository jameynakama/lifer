# R2 Media Hosting + Natural Key Schema

**Date:** 2026-05-27  
**Status:** Approved

## Summary

Two related changes shipped together:

1. **Natural key schema** -- replace `BIGSERIAL` integer PKs on `species`, `species_recordings`, and `species_images` with their existing natural text keys (`ebird_code`, `xeno_canto_id`, `macaulay_id`). All FK columns in `cards`, `group_species`, and `user_species_preferences` updated to match.
2. **R2 media hosting** -- replace the local-disk download path in `cmd/ingest` with streaming upload to a Cloudflare R2 bucket. Store full public R2 URLs in `file_path`. Remove `--skip-media` flag.

No changes to server startup, auth, JWT, group CRUD, or FSRS scheduling logic. Frontend changes are minor field renames.

## Motivation

### Natural keys
`BIGSERIAL` IDs are non-deterministic across DB resets. Re-ingesting into a fresh DB produces different IDs, making any dump fragment referencing species (cards, group memberships) inconsistent. Natural keys (`ebird_code` etc.) are stable, human-readable, and intrinsic to the data -- the same ingest run produces the same identifiers regardless of DB state. Doing this pre-launch is zero-cost: no user data exists to migrate.

### R2 media
The current `--skip-media` flag stored external CDN URLs (xeno-canto, Macaulay) as a workaround for not having object storage. Problems at production scale:
- xeno-canto CDN has no CORS headers -- audio scrubbing is unreliable
- URL rot: external URLs can change or disappear
- Streaming xeno-canto responses are chunked -- seek/scrub fails

R2 stores a stable copy under our control. Deployment workflow: ingest locally → data goes to R2 → dump DB → restore in prod. Every environment reads from the same bucket.

## Schema Changes

Squashed into `001_initial.up.sql` (no migration history to preserve).

### `species`
```
BEFORE: id BIGSERIAL PRIMARY KEY, ebird_code TEXT NOT NULL UNIQUE
AFTER:  ebird_code TEXT PRIMARY KEY  (id column removed)
```

### `species_recordings`
```
BEFORE: id BIGSERIAL PRIMARY KEY, species_id BIGINT FK, xeno_canto_id TEXT UNIQUE
AFTER:  xeno_canto_id TEXT PRIMARY KEY, species_code TEXT NOT NULL REFERENCES species(ebird_code)
```

### `species_images`
```
BEFORE: id BIGSERIAL PRIMARY KEY, species_id BIGINT FK, macaulay_id TEXT UNIQUE
AFTER:  macaulay_id TEXT PRIMARY KEY, species_code TEXT NOT NULL REFERENCES species(ebird_code)
```

### `cards`
```
BEFORE: species_id BIGINT NOT NULL REFERENCES species(id), UNIQUE(user_id, species_id, lane)
AFTER:  species_code TEXT NOT NULL REFERENCES species(ebird_code), UNIQUE(user_id, species_code, lane)
```

### `group_species`
```
BEFORE: species_id BIGINT REFERENCES species(id), PK (group_id, species_id)
AFTER:  species_code TEXT REFERENCES species(ebird_code), PK (group_id, species_code)
```

### `user_species_preferences`
```
BEFORE: species_id BIGINT REFERENCES species(id), PK (user_id, species_id)
AFTER:  species_code TEXT REFERENCES species(ebird_code), PK (user_id, species_code)
```

### Unchanged
`groups`, `users` -- no natural key; these travel with the dump intact.

## New `internal/r2` Package

```go
type Client struct {
    s3     *s3.Client
    bucket string
    pubURL string // e.g. "https://pub-abc.r2.dev" or "https://media.lifer.app"
}

func New(accountID, accessKeyID, secretKey, bucket, pubURL string) (*Client, error)

// Upload streams body to R2 at the given key, returns the full public URL.
// Buffers body in memory for Content-Length (files are small: a few MB max).
func (c *Client) Upload(ctx context.Context, key, contentType string, body io.Reader) (string, error)
```

**Configuration:**
- Endpoint: `https://<accountID>.r2.cloudflarestorage.com`
- Region: `auto` (Cloudflare R2 requirement)
- Credentials: static `accessKeyID` / `secretKey`
- Path-style addressing (`UsePathStyle: true`)

**Dependencies added to `go.mod`:**
- `github.com/aws/aws-sdk-go-v2/config`
- `github.com/aws/aws-sdk-go-v2/credentials`
- `github.com/aws/aws-sdk-go-v2/service/s3`

**Tests:** `internal/r2/client_test.go` uses an `httptest.Server` standing in for the S3 endpoint.

## `cmd/ingest` Changes

### Removed
- `--skip-media` flag
- `ASSETS_DIR` environment variable
- `downloadFile` function and all its tests (`TestDownloadFile*`)
- All `if skipMedia { ... } else { ... }` branching in `ingestSpecies`

### Added
Environment variables (all required):
```
R2_ACCOUNT_ID
R2_ACCESS_KEY_ID
R2_SECRET_ACCESS_KEY
R2_BUCKET_NAME
R2_PUBLIC_URL      # e.g. https://pub-abc123.r2.dev
```

### `ingestSpecies` signature change
```go
// Before
func ingestSpecies(..., assetsDir string, skipMedia bool) error

// After
func ingestSpecies(..., r2c *r2.Client) error
```

### Object key scheme
- Recordings: `recordings/<ebird_code>/<xeno_canto_id>.mp3`
- Images:     `images/<ebird_code>/<macaulay_id>.jpg`

`file_path` in DB stores the full public URL: `R2_PUBLIC_URL + "/" + key`.

### FK reference change
```go
// Before: sp.ID (int64)
q.UpsertRecording(ctx, store.UpsertRecordingParams{SpeciesID: sp.ID, ...})

// After: sp.EbirdCode (string)
q.UpsertRecording(ctx, store.UpsertRecordingParams{SpeciesCode: sp.EbirdCode, ...})
```

## SQL Query Changes

All queries updated after schema change; `just generate` regenerates Go types.

Key changes:
- `UpsertSpecies` -- returns `ebird_code` as PK; no `id` column
- `UpsertRecording` / `UpsertSpeciesImage` -- `species_id` param → `species_code`
- `GetRandomRecording` / `GetRandomImage` -- param type `BIGINT` → `TEXT`
- `GetNextDueCard` -- joins on `species_code`; returns `ebird_code` instead of numeric `id`
- `GetCard` / `UpdateCardSchedule` / `UpsertCard` / `DeleteCard` -- `species_id` → `species_code`
- `AddSpeciesToGroup` / `RemoveSpeciesFromGroup` -- `species_id` → `species_code`
- `ListGroupSpecies` -- no `id` column; `ebird_code` is the identifier
- `ListUserGroups` -- join uses `species_code`
- `SearchSpecies` -- no `id` column; returns `ebird_code` as identifier
- `UpsertPreferences` / `GetPreferences` -- `species_id` → `species_code`

## Go API Handler Changes

After `just generate`, sqlc types change automatically. Manual handler updates:

**`internal/api/quiz.go`**
```go
// Before
type nextCardResponse struct { SpeciesID int64 `json:"species_id"` ... }
type rateCardRequest  struct { SpeciesID int64 `json:"species_id"` ... }

// After
type nextCardResponse struct { EbirdCode string `json:"ebird_code"` ... }
type rateCardRequest  struct { EbirdCode string `json:"ebird_code"` ... }
```

**`internal/api/groups.go`**
```go
// Before
type addSpeciesRequest struct { SpeciesID int64 `json:"species_id"` }
// strconv.ParseInt(chi.URLParam(r, "species_id"), 10, 64)

// After
type addSpeciesRequest struct { EbirdCode string `json:"ebird_code"` }
// chi.URLParam(r, "ebird_code")  -- no parsing needed, it's already a string
```

**`internal/api/router.go`**
```
DELETE /groups/{id}/species/{species_id}  →  /groups/{id}/species/{ebird_code}
```

**`internal/api/preferences.go`** -- `species_id` int64 param → `ebird_code` string throughout.

## Frontend Changes

All in `frontend/src/`:

**`routes/groups/[id]/+page.svelte`**
- `s.id` → `s.ebird_code` (keyed each, find, filter, button handlers)
- `JSON.stringify({ species_id: speciesId })` → `{ ebird_code: ebirdCode }`
- DELETE URL: `/groups/${groupId}/species/${s.ebird_code}`

**`routes/groups/[id]/quiz/+page.svelte`**
- `selected.id === card.species_id` → `selected.ebird_code === card.ebird_code`
- `{#key card.species_id}` → `{#key card.ebird_code}`
- `JSON.stringify({ species_id: card.species_id, ... })` → `{ ebird_code: card.ebird_code, ... }`

## Environment Variable Summary

### Removed from `.env.example`
```
ASSETS_DIR
```

### Added to `.env.example`
```
R2_ACCOUNT_ID=your-account-id
R2_ACCESS_KEY_ID=your-access-key-id
R2_SECRET_ACCESS_KEY=your-secret-access-key
R2_BUCKET_NAME=lifer-media
R2_PUBLIC_URL=https://pub-xxx.r2.dev
```

## Testing

- **`internal/r2`**: unit tests with `httptest.Server` acting as S3 endpoint; cover successful upload, non-200 source, non-200 R2 PUT
- **`cmd/ingest/main_test.go`**: remove `TestDownloadFile*`; `TestFilterBySpecies` / `TestFilterComplete` unchanged; `ingestSpecies` is integration-level (requires DB + live APIs) and remains untested directly -- the `r2.Client` is tested via its own package
- **API tests**: update `species_id` int64 references to `ebird_code` string throughout

## What Does Not Change

- Server startup (`cmd/server/main.go`)
- Auth, JWT, OAuth flow
- Group CRUD (name, create, delete, update)
- FSRS scheduling logic
- WavePlayer / frontend audio playback
- `file_path` column name (still stores a URL string -- just an R2 URL now instead of a CDN URL)
