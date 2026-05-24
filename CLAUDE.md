# Lifer -- Claude Code Context

## What this is
A spaced repetition web app for bird song/call and image identification practice. Users hear a recording or see a photo, type the species name via a typeahead, reveal the answer, and advance. Rating is automatic: correct → Good (3), wrong → Again (1). FSRS drives scheduling. Groups let users practice by region (eBird-sourced presets) or custom species sets.

## Stack decisions
- **Go + chi + sqlc + PostgreSQL** -- Go backend as a learning project. chi is idiomatic stdlib-style routing. sqlc generates typed Go from raw SQL -- no ORM. Schema changes caught at compile time.
- **SvelteKit (adapter-static, SPA mode)** -- migrated from plain Vite. File-based routing, `export const ssr = false` in `+layout.ts`, `fallback: 'index.html'` in adapter config. Go serves the built static files and all `/api/*` routes.
- **Google OAuth → HttpOnly cookie JWT** -- no passwords to manage. JWT stored in HttpOnly cookie (not localStorage) so JS can't read/steal it. State param in OAuth flow prevents CSRF.
- **FSRS over SM-2** -- more accurate memory model, still simple to integrate. Backend endpoints exist; full FSRS scheduling not yet implemented.
- **WaveSurfer.js** -- waveform audio player. Uses a native `<audio>` element (passed as `media:`) to avoid CORS issues with xeno-canto CDN URLs; fake peaks provided for visual bars since XHR decode is blocked.

## Data model
- `users` -- google_id, email, name, picture, is_admin
- `species` -- common_name, scientific_name, ebird_code
- `species_recordings` -- species FK, xeno_canto_id, file_path (local relative path or external URL), quality (A-E), type (raw string from xeno-canto, e.g. "song", "call")
- `species_images` -- species FK, macaulay_id, file_path, credit
- `groups` -- preset (eBird-sourced) or user-created; owner_id null for presets
- `group_species` -- join table
- `cards` -- user × recording with FSRS fields: stability, difficulty, due, last_review, reps, lapses, state

Cards are user-scoped. Species/recordings/images are global shared catalog. is_admin flag on users controls catalog management.

## What's built
- [x] Go module scaffold: `backend/cmd/server`, `internal/api`, `internal/auth`, `internal/store`
- [x] PostgreSQL migrations (`just migrate-up`)
- [x] sqlc config + generated types (run `just generate` after schema changes)
- [x] chi router with Logger/Recoverer/RequestID middleware
- [x] Google OAuth flow: `/api/v1/auth/google` → Google → `/api/v1/auth/google/callback`
- [x] JWT sign/verify, RequireAuth middleware, user ID in context
- [x] `GET /health`, `GET /api/v1/me` (auth-protected)
- [x] Justfile with test/run/build/migrate-up/migrate-down/generate/migration
- [x] Docker Compose for PostgreSQL (port 5435)
- [x] Full styling: dark/light themes (CSS custom properties), Inter font, atmospheric login, token-based components
- [x] Theme toggle (sun/moon) with localStorage persistence + OS preference fallback
- [x] `cmd/ingest` binary: eBird taxonomy + region species lists → xeno-canto (API v3, quality A/B) + Macaulay Library photos; buffered channel worker pool (default 5); idempotent upserts; post-run cleanup removes species missing recordings or images from DB and disk; `--skip-media` flag stores external URLs instead of downloading files (no `ASSETS_DIR` required)
- [x] Migration 002: `type text` column on `species_recordings`
- [x] Migration 003: rename `recordings` → `species_recordings`
- [x] SvelteKit migration: adapter-static SPA mode, file-based routing under `src/routes/`
- [x] Group detail page with species search (typeahead) and add/remove
- [x] `GET /api/v1/groups/:id/next` -- returns next due card (204 when done)
- [x] `POST /api/v1/groups/:id/rate` -- accepts `{ species_id, lane, rating }`
- [x] Quiz page: fetches real cards, audio lane (WavePlayer) + image lane (photo), auto-rates correct→3/wrong→1, single Next button (no self-reporting)
- [x] SpeciesTypeahead component: filters by common/scientific name, ARIA combobox pattern, `onmousedown` (not `onclick`) to avoid blur-before-click race
- [x] QuizCard + ImageQuizCard: typeahead auto-check (`selected.id === card.species_id`), `{#key card.species_id}` to reset state between cards
- [x] RevealCard: shows result banner (correct/incorrect), reference photo, species names, Next button
- [x] WavePlayer: WaveSurfer.js with native `<audio>` element to avoid CORS; fake peaks for bar visualization
- [x] FSRS scheduling: `github.com/open-spaced-repetition/go-fsrs/v3`; `rateCard` fetches current card state, runs `f.Next()`, persists updated stability/difficulty/due/lapses/state

## What's next

### 1. R2 media hosting (pre-launch)
- Currently using `--skip-media` (external CDN URLs) which works for local dev
- For production: teach `cmd/ingest` to stream audio/images from xeno-canto/Macaulay directly into R2 via S3-compatible PUT (no temp files needed -- pipe `io.Reader` straight through)
- Fixes CORS issues, URL rot, and unreliable scrubbing on streaming responses

### 2. Ad-hoc / practice mode
- Study all cards in a group at any time, not just due ones -- cycle through every species regardless of schedule
- Results don't count toward FSRS ratings (no POST to `/rate`)
- Useful for drilling a group before a birdwatching trip, or learning new species before they've been scheduled

### 3. Catalog / "Learn" view
- Browse all species, filterable by region and alphabetically
- Shows recordings, photos, and species info
- Users can add species to custom groups from list or detail view
- Requires backend search/filter API (too many species to load all client-side)
- Design group management UI alongside this (same "add to list" action)

### 4. Group management
- Admin: create/edit preset groups (region-based)
- Users: create custom groups, add/remove species
- Shared UI surface with catalog view

### 5. Admin UI
- Catalog management: add/edit species, recordings, images

## Bugs / fixes
- **Enter to submit answer** -- pressing Enter in the SpeciesTypeahead should trigger "Reveal answer" (same as clicking the button) when a species is selected

## Key non-obvious choices
- sqlc lives in `backend/` -- run `just generate` from repo root after any migration change
- `go.mod` is in `backend/`, not repo root (monorepo with `frontend/` alongside)
- OAuth state stored as short-lived cookie (5min) to prevent CSRF -- verified on callback
- `UpsertUser` updates name/picture on every login so profile changes from Google sync automatically
- Generated sqlc files (`internal/store/*.go` except `queries/`) are committed -- don't `.gitignore` them
- SvelteKit mounts into `<div style="display: contents">`, not `<div id="app">` -- `#app` CSS is dead after migration; use `.app-container` in `+layout.svelte`
- WaveSurfer: xeno-canto CDN has no CORS headers, so XHR decode fails. Pass `media: audio` (native `<audio>` element) -- no XHR. Provide fake `peaks` so WaveSurfer draws bars immediately without needing to decode. Without peaks, `ready` fires on `canplay` which is slower.
- Quiz auto-rating: no self-reporting; `correct ? 3 : 1` posted to `/rate`. Correct = `selected.id === card.species_id`.

## Local setup on a new machine
```
git clone https://github.com/jameynakama/lifer
cd lifer
cp .env.example .env   # fill in values
docker compose up -d   # starts postgres on port 5433
just migrate-up
just run               # starts on :8080
just                   # runs tests (default)
```

## External APIs

All three are **ingestion-only** -- hit them once to populate the DB, never needed again at runtime. By default, ingest stores external URLs directly in `file_path` (`--skip-media` flag); pass without the flag to download files locally instead.

### eBird API
- **Used for:** regional species checklists → preset groups (e.g. "Pacific Northwest")
- **Auth:** free API key, request at ebird.org/api/keygen
- **Key endpoint:** `GET /v2/product/spplist/{regionCode}` returns species codes for a region
- **Region codes:** e.g. `US-WA` for Washington state, `US-OR` for Oregon
- **Docs:** https://documenter.getpostman.com/view/664302/S1ENwy59

### Xeno-canto
- **Used for:** bird call/song recordings (the core quiz content)
- **Auth:** API key required (v3); free for registered members with verified email
- **Approach:** query separately for `type:song` and `type:call` using `en:"common name"` (lowercased, quoted); filter to quality A or B (A-first); store raw `type` string and `file_path` in `species_recordings` -- either the xeno-canto download URL (`--skip-media`) or a local relative path
- **Audio URL format:** `https://xeno-canto.org/{id}/download` -- confirmed works as `<audio src>` in browsers
- **Key endpoint:** `GET https://xeno-canto.org/api/3/recordings?key={key}&query=type:call%20en:%22cooper%27s%20hawk%22` -- **v3, not v2**
- **Do NOT use** `gen:` + `sp:` -- eBird reclassifies genera faster than xeno-canto updates (e.g. Cooper's Hawk is `Astur` in eBird but `Accipiter` in xeno-canto), causing silent empty results. `en:` by common name is accurate and stable.
- **Encoding gotcha:** `en:` with multi-word names requires `%20` for spaces inside the quotes, NOT `+`. Use `url.PathEscape` (not `url.Values.Encode`) to build the query parameter.
- Do NOT use `q:A` in query -- filter client-side instead.
- **License:** all recordings are Creative Commons -- safe to store and serve
- **Docs:** https://xeno-canto.org/explore/api

### Macaulay Library (Cornell Lab)
- **Used for:** species photos shown on quiz reveal and for future image ID quizzes
- **Auth:** same eBird API key (`X-eBirdApiToken` header)
- **Approach:** search by species code, store top-rated photos in `species_images` -- either the Macaulay CDN URL (`--skip-media`) or a local relative path
- **Image URL format:** `https://cdn.download.ams.birds.cornell.edu/api/v1/asset/{assetId}/large` -- works directly as `<img src>`
- **Key endpoint:** `GET https://search.macaulaylibrary.org/api/v1/search?taxonCode={speciesCode}&mediaType=photo&sort=rating_rank_desc&count={n}` -- **not api.ebird.org**
- **Response shape:** `{ results: { content: [ { assetId, userDisplayName, ... } ] } }`
- **Note:** same Cornell/eBird ecosystem -- one API key covers eBird checklists and Macaulay media
