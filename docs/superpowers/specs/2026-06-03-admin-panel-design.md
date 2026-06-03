# Admin Panel Design

**Date:** 2026-06-03
**Scope:** Media CRUD (images + recordings) for catalog management

## Problem

Bad catalog media (e.g. leucistic bird photos) currently requires manual R2 edits + raw SQL on the
host to fix. The admin panel gives an authenticated admin a UI to delete, upload, and replace media
without touching the infrastructure directly.

## Scope

- Media only: images and recordings. Species names/codes are out of scope for now.
- Single admin user (Jamey). `is_admin` already exists on the `users` table.
- Simple, not fancy: classic form submits for search/pagination, raw fetch for deletes/uploads.

## Architecture

Two new layers on top of the existing stack:

- **Backend:** `RequireAdmin` middleware + six new endpoints under `/api/v1/admin/`
- **Frontend:** two new SvelteKit routes under `/admin/`

No changes to existing user-facing routes or components.

## Backend

### Middleware

`RequireAdmin` reads the user already set in context by `RequireAuth`, checks `IsAdmin bool`,
returns 403 if false. Admin routes are mounted as:

```go
r.With(RequireAuth, RequireAdmin).Route("/api/v1/admin", func(r chi.Router) { ... })
```

### Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/species?q=&limit=&offset=` | Reuse existing paginated species search (no new endpoint needed) |
| GET | `/api/v1/admin/species/:ebird_code` | Detail: all images + all recordings for a species |
| POST | `/api/v1/admin/species/:ebird_code/images` | Upload image → R2 → DB |
| POST | `/api/v1/admin/species/:ebird_code/recordings` | Upload recording → R2 → DB |
| DELETE | `/api/v1/admin/species/:ebird_code/images/:macaulay_id` | R2 delete + DB delete |
| DELETE | `/api/v1/admin/species/:ebird_code/recordings/:xeno_canto_id` | R2 delete + DB delete |

### SQL changes

- `SearchSpecies`: add `OR ebird_code ILIKE '%' || $1 || '%'` to existing WHERE clause so admins
  can search by ebird code in addition to common/scientific name.
- New query `GetImageByID(macaulay_id TEXT)`: `SELECT macaulay_id, file_path, credit FROM species_images WHERE macaulay_id = $1`
- New query `GetRecordingByID(xeno_canto_id TEXT)`: `SELECT xeno_canto_id, file_path, quality, type FROM species_recordings WHERE xeno_canto_id = $1`
- New query `DeleteImage(macaulay_id TEXT)`: `DELETE FROM species_images WHERE macaulay_id = $1`
- New query `DeleteRecording(xeno_canto_id TEXT)`: `DELETE FROM species_recordings WHERE xeno_canto_id = $1`
- Reuse existing `UpsertSpeciesImage` and `UpsertRecording` for creates.

### r2.Client changes

Add one new method:

```go
func (c *Client) Delete(ctx context.Context, key string) error
```

Wraps `s3.DeleteObject`. Key is derived from the stored `file_path` by stripping the public URL
prefix: `strings.TrimPrefix(filePath, c.pubURL+"/")`.

### Upload flow

1. Parse multipart body
2. Detect content type from file part header
3. Generate ID: `admin-{uuid}` (used as macaulay_id or xeno_canto_id)
4. Build R2 key: `images/{ebird_code}/admin-{uuid}.{ext}` or `recordings/{ebird_code}/admin-{uuid}.{ext}`
5. Call `r2.Upload(ctx, key, contentType, fileReader)` → returns public URL
6. Call `UpsertSpeciesImage` or `UpsertRecording` with generated ID + URL + form fields

For images: form fields = `credit` (optional, defaults to `"admin upload"`).
For recordings: form fields = `quality` (A--E, required) and `type` (song/call, required).

### Delete flow

1. Fetch record from DB to get `file_path`
2. Derive R2 key by stripping public URL prefix
3. Call `r2.Delete(ctx, key)`
4. Delete record from DB

## Frontend

### Routes

- `src/routes/admin/+layout.svelte` -- reads `me.is_admin` from user store; redirects to `/` if
  false. Minimal admin shell (header only).
- `src/routes/admin/+page.svelte` -- plain `<form method="GET">` with text input + submit. Reads
  `?q=`, `?limit=`, `?offset=` from URL params; calls `/api/v1/species`; renders species list as
  links to `/admin/species/[ebird_code]` + prev/next pagination buttons.
- `src/routes/admin/species/[ebird_code]/+page.svelte` -- two sections:

**Images section**
- Simple grid of thumbnails (existing `file_path` URLs)
- Each thumbnail has a Delete button → fetch DELETE `/api/v1/admin/species/:code/images/:id` →
  `window.location.reload()`
- Upload form below: file input + credit text field + Submit → fetch POST multipart → reload

**Recordings section**
- Table rows: xeno_canto_id, quality, type, each with a Delete button → fetch DELETE → reload
- Upload form below: file input + quality select (A--E) + type select (song/call) + Submit →
  fetch POST multipart → reload

### Nav

Add `{#if $user?.is_admin}<a href="/admin">Admin</a>{/if}` to the main `+layout.svelte` nav.

## Testing

### Backend

- `RequireAdmin` middleware: authed admin passes, authed non-admin → 403, unauthenticated → 401
- Each handler: httptest server + test DB, assert status codes and response shape (follows existing
  pattern in `internal/api/decks_test.go`)
- `r2.Client.Delete`: httptest.Server asserting correct DELETE request sent (follows existing
  `r2/client_test.go` pattern)

### Frontend

- Admin layout: non-admin user is redirected to `/`
- Admin search page and species detail page render without errors
