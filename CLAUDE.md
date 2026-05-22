# Lifer -- Claude Code Context

## What this is
A spaced repetition web app for bird song/call and image identification practice. Users hear a recording, type the species name, reveal the answer, and rate confidence 1-4. FSRS drives scheduling. Groups let users practice by region (eBird-sourced presets) or custom species sets.

## Stack decisions
- **Go + chi + sqlc + PostgreSQL** -- Go backend as a learning project. chi is idiomatic stdlib-style routing. sqlc generates typed Go from raw SQL -- no ORM. Schema changes caught at compile time.
- **Svelte (Vite, no SvelteKit yet)** -- lighter than React, less boilerplate, compiles to vanilla JS. Good for a quiz app with real interactivity (audio, reveal, rating).
- **Google OAuth → HttpOnly cookie JWT** -- no passwords to manage. JWT stored in HttpOnly cookie (not localStorage) so JS can't read/steal it. State param in OAuth flow prevents CSRF.
- **FSRS over SM-2** -- more accurate memory model, still simple to integrate. Not yet implemented -- cards table is ready, algorithm comes next.

## Data model
- `users` -- google_id, email, name, picture, is_admin
- `species` -- common_name, scientific_name, ebird_code
- `recordings` -- species FK, xeno_canto_id, file_path, quality (A-E)
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
- [x] Svelte 5 frontend: Login, Dashboard, Quiz views + StatsBar, GroupList, QuizCard, RevealCard components
- [x] Google OAuth login button, auth check on load, state-based routing
- [x] Full styling: dark/light themes (CSS custom properties), Inter font, atmospheric login, token-based components
- [x] Theme toggle (sun/moon) with localStorage persistence + OS preference fallback

## What's next

### 1. Ingestion scripts (do this first -- needed for real data)
- Get eBird API key at ebird.org/api/keygen (free, requires account)
- `cmd/ingest/main.go` binary: region code arg → eBird species list → goroutine worker pool → xeno-canto (recordings, quality A/B) + Macaulay Library (photos) in parallel
- Worker pool ~5-10 concurrent (respect xeno-canto rate limits); `errgroup` + semaphore pattern
- Upserts idempotent on `xeno_canto_id` / `macaulay_id` -- safe to re-run
- eBird regions are state-level codes (`US-WA`, `US-OR`); "Pacific Northwest" preset = union of multiple state runs
- Store MP3s and photos locally (or S3 later); file paths in DB

**eBird → xeno-canto species lookup:**
- eBird taxonomy gives `sciName: "Melospiza melodia"` -- split on space to get `gen` + `sp`
- Query xeno-canto as `gen:Melospiza+sp:melodia` (case-insensitive, no encoding issues)
- Do NOT use `en:` (common name) -- spacing/case makes it unreliable

**Recording type strategy:**
- xeno-canto `type` field contains values like `"song"`, `"call"`, `"call, song"`, `"alarm call, call, song, gurgle song, various calls"` (very free-form)
- Query separately for `type:song` and `type:call`, grab 2-3 of each per species (~4-6 total)
- Skip `type:alarm` for now -- multi-word type filtering is broken in xeno-canto API (can't query "alarm call" in any encoding)
- Store the raw `type` string in `recordings.type` column -- useful for display and future filtering
- **Why type variety matters:** species like Song Sparrow have a simple "chip" call all year but dozens of song variants in spring; quizzing on both is intentional

**Schema addition needed:** `recordings` table needs a `type text` column (not in current migration)

### 2. FSRS + quiz endpoints
- `GET /api/v1/groups/:id/next` -- returns next due card for the group
- `POST /api/v1/groups/:id/rate` -- takes rating 1-4, updates FSRS fields (stability, difficulty, due, state)
- Swap `MOCK_CARDS` in `Quiz.svelte` for real fetch calls

### 3. SvelteKit migration (before catalog view)
- Current plain Vite + store-based routing will get unwieldy with more views
- Migrate before adding the catalog/browse view

### 4. Catalog / "Learn" view
- Browse all species, filterable by region and alphabetically
- Shows recordings, photos, and species info
- Users can add species to custom groups from list or detail view
- Requires SvelteKit routing + backend search/filter API (too many species to load all client-side)
- Design group management UI alongside this (same "add to list" action)

### 5. Group management
- Admin: create/edit preset groups (region-based)
- Users: create custom groups, add/remove species
- Shared UI surface with catalog view

### 6. Admin UI
- Catalog management: add/edit species, recordings, images

## Key non-obvious choices
- sqlc lives in `backend/` -- run `just generate` from repo root after any migration change
- `go.mod` is in `backend/`, not repo root (monorepo with `frontend/` alongside)
- OAuth state stored as short-lived cookie (5min) to prevent CSRF -- verified on callback
- `UpsertUser` updates name/picture on every login so profile changes from Google sync automatically
- Generated sqlc files (`internal/store/*.go` except `queries/`) are committed -- don't `.gitignore` them

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

All three are **ingestion-only** -- hit them once to populate the DB, store assets locally or S3, never needed again at runtime.

### eBird API
- **Used for:** regional species checklists → preset groups (e.g. "Pacific Northwest")
- **Auth:** free API key, request at ebird.org/api/keygen
- **Key endpoint:** `GET /v2/product/spplist/{regionCode}` returns species codes for a region
- **Region codes:** e.g. `US-WA` for Washington state, `US-OR` for Oregon
- **Docs:** https://documenter.getpostman.com/view/664302/S1ENwy59

### Xeno-canto
- **Used for:** bird call/song recordings (the core quiz content)
- **Auth:** free API key for registered members with verified email
- **Approach:** search by species scientific name, filter to quality A or B, download MP3s, store file path in `recordings.file_path`
- **Key endpoint:** `GET /api/2/recordings?query={scientific_name}+q:A` 
- **License:** all recordings are Creative Commons -- safe to store and serve
- **Docs:** https://xeno-canto.org/explore/api

### Macaulay Library (Cornell Lab)
- **Used for:** species photos shown on quiz reveal and for future image ID quizzes
- **Auth:** same eBird API key
- **Approach:** search by species code, download a few photos per species, store in `species_images`
- **Key endpoint:** `GET /v2/ref/media/best?species={speciesCode}&mediaType=photo`
- **Note:** same Cornell/eBird ecosystem -- one API key covers both eBird checklists and Macaulay media
