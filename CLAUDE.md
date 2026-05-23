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
- [x] Svelte 5 frontend: Login, Dashboard, Quiz views + StatsBar, GroupList, QuizCard, RevealCard components
- [x] Google OAuth login button, auth check on load, state-based routing
- [x] Full styling: dark/light themes (CSS custom properties), Inter font, atmospheric login, token-based components
- [x] Theme toggle (sun/moon) with localStorage persistence + OS preference fallback
- [x] `cmd/ingest` binary: eBird taxonomy + region species lists → xeno-canto (API v3, quality A/B) + Macaulay Library photos; buffered channel worker pool (default 5); idempotent upserts; post-run cleanup removes species missing recordings or images from DB and disk; `--skip-media` flag stores external URLs instead of downloading files (no `ASSETS_DIR` required)
- [x] Migration 002: `type text` column on `species_recordings`
- [x] Migration 003: rename `recordings` → `species_recordings`

## What's next

### 1. FSRS + quiz endpoints
- `GET /api/v1/groups/:id/next` -- returns next due card for the group
- `POST /api/v1/groups/:id/rate` -- takes rating 1-4, updates FSRS fields (stability, difficulty, due, state)
- Swap `MOCK_CARDS` in `Quiz.svelte` for real fetch calls

**Quiz lanes:** audio and image recognition are separate FSRS lanes per species, independently scheduled. A wigeon can be mature in image lane but new in audio lane. Lane preference is global per user×species (not per group) -- default both enabled. Data model implications:
- Audio lane: `user × (species, call_type)` -- one FSRS card per species per type ("song"/"call"), but each review shows a **random recording** from that pool so the user generalises rather than memorising one clip. Existing `cards` table needs updating (currently `user × recording`).
- Image lane: `user × species` (recognizing the species from any photo, not memorizing individual photos) -- needs new cards-like structure
- Preference: new `user_species_preferences (user_id, species_id, audio_enabled, image_enabled)` table

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
