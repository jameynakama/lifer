# Lifer -- Claude Code Context

## What this is
A spaced repetition web app for bird song/call and image identification practice. Users hear a recording or see a photo, type the species name via a typeahead, reveal the answer, and advance. Rating is automatic: correct → Good (3), wrong → Again (1). FSRS drives scheduling. Groups let users practice by region (eBird-sourced presets) or custom species sets.

## Stack decisions
- **Go + chi + sqlc + PostgreSQL** -- Go backend as a learning project. chi is idiomatic stdlib-style routing. sqlc generates typed Go from raw SQL -- no ORM. Schema changes caught at compile time.
- **SvelteKit (adapter-static, SPA mode)** -- migrated from plain Vite. File-based routing, `export const ssr = false` in `+layout.ts`, `fallback: 'index.html'` in adapter config. Go serves the built static files and all `/api/*` routes.
- **Google OAuth → HttpOnly cookie JWT** -- no passwords to manage. JWT stored in HttpOnly cookie (not localStorage) so JS can't read/steal it. State param in OAuth flow prevents CSRF.
- **FSRS over SM-2** -- more accurate memory model, still simple to integrate. Full FSRS scheduling implemented via `github.com/open-spaced-repetition/go-fsrs/v3`.
- **WaveSurfer.js** -- waveform audio player. Uses a native `<audio>` element (passed as `media:`) to avoid CORS issues with CDN URLs; fake peaks provided for visual bars since XHR decode is blocked.
- **Cloudflare R2** -- object storage for all media. `cmd/ingest` streams audio/images from xeno-canto/Macaulay directly to R2 via aws-sdk-go-v2 S3-compatible API. Full public R2 URLs stored in `file_path`. Every environment reads from the same bucket.

## Data model
- `users` -- google_id, email, name, picture, is_admin; BIGSERIAL PK
- `species` -- **`ebird_code TEXT PRIMARY KEY`**, common_name, scientific_name
- `species_recordings` -- **`xeno_canto_id TEXT PRIMARY KEY`**, `species_code TEXT FK → species(ebird_code)`, file_path (full R2 public URL), quality (A-E), type (e.g. "song", "call")
- `species_images` -- **`macaulay_id TEXT PRIMARY KEY`**, `species_code TEXT FK → species(ebird_code)`, file_path (full R2 public URL), credit
- `groups` -- preset (eBird-sourced) or user-created; owner_id null for presets; BIGSERIAL PK
- `group_species` -- join table; PK is `(group_id, species_code)`
- `cards` -- user × species × lane with FSRS fields; UNIQUE `(user_id, species_code, lane)`

Natural text keys on species/recordings/images are stable across DB resets -- re-ingesting into a fresh DB produces the same identifiers. Cards are user-scoped. Species/recordings/images are global shared catalog.

## What's built
- [x] Go module scaffold: `backend/cmd/server`, `internal/api`, `internal/auth`, `internal/store`
- [x] PostgreSQL migrations (`just migrate-up`) -- squashed into `001_initial.up.sql` (natural key schema)
- [x] sqlc config + generated types (run `just generate` after schema changes)
- [x] chi router with Logger/Recoverer/RequestID middleware
- [x] Google OAuth flow: `/api/v1/auth/google` → Google → `/api/v1/auth/google/callback`
- [x] JWT sign/verify, RequireAuth middleware, user ID in context
- [x] `GET /health`, `GET /api/v1/me` (auth-protected)
- [x] Justfile with test/run/build/migrate-up/migrate-down/generate/migration
- [x] Docker Compose for PostgreSQL (port 5435)
- [x] Full styling: dark/light themes (CSS custom properties), Inter font, atmospheric login, token-based components
- [x] Theme toggle (sun/moon) with localStorage persistence + OS preference fallback
- [x] `cmd/ingest` binary: eBird taxonomy + region species lists → xeno-canto (API v3, quality A/B) + Macaulay Library photos; streams directly to Cloudflare R2 (no local temp files); buffered channel worker pool (default 5); idempotent upserts; post-run cleanup removes species missing recordings or images
- [x] `internal/r2` package: S3-compatible R2 upload client; `New` / `NewWithEndpoint` (testable with httptest); buffers body in memory then PutObject with ContentLength
- [x] SvelteKit migration: adapter-static SPA mode, file-based routing under `src/routes/`
- [x] Group detail page with species search (typeahead) and add/remove
- [x] `GET /api/v1/groups/:id/next` -- returns next due card (204 when done)
- [x] `POST /api/v1/groups/:id/rate` -- accepts `{ ebird_code, lane, rating }`
- [x] Quiz page: fetches real cards, audio lane (WavePlayer) + image lane (photo), auto-rates correct→3/wrong→1, single Next button (no self-reporting)
- [x] SpeciesTypeahead component: filters by common/scientific name, ARIA combobox pattern, `onmousedown` (not `onclick`) to avoid blur-before-click race
- [x] QuizCard + ImageQuizCard: typeahead auto-check (`selected.ebird_code === card.ebird_code`), `{#key card.ebird_code}` to reset state between cards
- [x] RevealCard: shows result banner (correct/incorrect), reference photo, species names, Next button
- [x] WavePlayer: WaveSurfer.js with native `<audio>` element to avoid CORS; fake peaks for bar visualization
- [x] FSRS scheduling: `rateCard` fetches current card state, runs `f.Next()`, persists updated stability/difficulty/due/lapses/state
- [x] **Explore feature:**
  - `GET /api/v1/species` -- unified paginated list + search; `{ count, next, previous, results }` shape; `limit`/`offset` params; `next`/`previous` are absolute URLs; `image_url` (first image for species, nullable) included in both list and search results
  - `GET /api/v1/species/:ebird_code` -- species detail with recordings + images
  - `GET /api/v1/species/:ebird_code/groups` -- which of the current user's groups contain this species
  - `/explore` page: searchable (debounced), paginated species list with `SpeciesRow` tiles (44×44 thumbnail + names + group button)
  - `/explore/[ebird_code]` page: name-first detail layout with `RecordingsList` (WavePlayer per row), `PhotoGrid` (2-col, click to open full-res, hover for credit overlay), `GroupDropdown`
  - `GroupDropdown`: per-species dropdown with checkboxes for each user group (checked = member), inline "create new group" input; TanStack Query mutations for add/remove/create; clicks inside dropdown don't bubble to window close handler
  - `@tanstack/svelte-query` added; `QueryClientProvider` wraps the layout
  - Groups list page now shows `audio_due`/`image_due` badge counts per group
  - `GET /api/v1/species/all` -- returns all species (no pagination) for client-side filtering; shared cache for Explore and group search typeahead
  - Client-side filtering: Explore page and group species search both use the `/species/all` cache via TanStack Query (`staleTime: Infinity`); no per-keystroke API hits
- [x] **Practice mode (free drill):**
  - `GET /api/v1/groups/:id/practice?lane=` -- returns all species in group with random media URLs; COALESCE-safe SQL, ordered by common_name; access check allows owner or preset groups, blocks other users' private groups
  - `/groups/[id]/practice` page: fetches all species upfront, Fisher-Yates shuffle, index cycling, no POST to `/rate`; "Practiced: X/Y" stat; done screen with "Practice Again" (reshuffle) + "Back to Group"
  - Groups list: "Free Practice" toggle -- shows banner + swaps due badges for quick "▶ Audio" / "◉ Image" launch buttons per group
  - Group detail: "Study Audio/Image" (filled, FSRS queue → `/quiz`) + "Practice Audio/Image" (outline, free drill → `/practice`)
  - Per-species audio/image lane preferences (`PUT /api/v1/species/:ebird_code/preferences`, `user_species_preferences` table): backend + frontend UI complete

## What's next

### 1. Explore -- region filter (separate from the Explore list/detail already built)
- eBird region codes are not stored in the DB -- no "which regions contain this species" endpoint exists
- Instead: client requests subregions from eBird on demand, then intersects the species list with our catalog
- Flow: user picks state → frontend hits backend proxy → backend calls `GET /v2/ref/region/list/subnational2/{stateCode}` → returns county list; user picks county → backend calls `GET /v2/product/spplist/{countyCode}` → returns species codes → backend intersects with our DB and returns matches
- Must proxy through backend to keep the eBird API key server-side
- `regionType` values: `country`, `subnational1` (states/provinces), `subnational2` (counties)

### 2. Admin UI
- Catalog management: add/edit species, recordings, images



## Key non-obvious choices
- sqlc lives in `backend/` -- run `just generate` from repo root after any migration change
- `go.mod` is in `backend/`, not repo root (monorepo with `frontend/` alongside)
- OAuth state stored as short-lived cookie (5min) to prevent CSRF -- verified on callback
- `UpsertUser` updates name/picture on every login so profile changes from Google sync automatically
- Generated sqlc files (`internal/store/*.go` except `queries/`) are committed -- don't `.gitignore` them
- SvelteKit mounts into `<div style="display: contents">`, not `<div id="app">` -- `#app` CSS is dead after migration; use `.app-container` in `+layout.svelte`
- WaveSurfer: CDN URLs may lack CORS headers. Pass `media: audio` (native `<audio>` element) -- no XHR. Provide fake `peaks` so WaveSurfer draws bars immediately. Without peaks, `ready` fires on `canplay` which is slower.
- Quiz auto-rating: no self-reporting; `correct ? 3 : 1` posted to `/rate`. Correct = `selected.ebird_code === card.ebird_code`.
- R2 ingest workflow: ingest locally (writes to R2 bucket) → dump DB → restore in prod. All environments share the same R2 bucket. No per-environment media copies needed.
- Natural PKs on species tables mean re-ingesting into a fresh DB produces identical IDs -- cards/group memberships stay valid across resets.
- TanStack Query v6 (`@tanstack/svelte-query`): use `createQuery(() => ({...}))` -- options wrapped in a function so Svelte 5 rune reads are tracked reactively. Returns a reactive object directly (not a store); access `.data`, `.isPending`, `.isError` without `$` prefix. Mutations use `createMutation(() => ({...}))` same pattern.
- `GET /api/v1/species` pagination: `next`/`previous` are absolute URL strings (or null), constructed server-side from `r.Host` + `X-Forwarded-Proto` header. `count` comes from `COUNT(*) OVER()` window function -- no second query needed.
- `GET /api/v1/species/all` is a separate unpaginated endpoint used exclusively for client-side filtering. TanStack Query key `['species', 'all']` with `staleTime: Infinity` -- fetched once per session, shared across Explore and group species typeahead. If catalog grows beyond regional scale, revert to server-side search (the paginated endpoint is still in place).

## Local setup on a new machine
```
git clone https://github.com/jameynakama/lifer
cd lifer
cp .env.example .env   # fill in values (including R2_* vars)
docker compose up -d   # starts postgres on port 5433
just migrate-up
just run               # starts on :8080
just                   # runs tests (default)
```

## Ingest workflow
```bash
# Ingest a region (streams media to R2, writes metadata to local DB)
just ingest US-OR

# Useful flags
just ingest --species busti US-OR          # single species
just ingest --skip-complete US-OR US-WA    # skip already-ingested species
just ingest --max-recordings 2 US-OR       # fewer recordings per species

# XC taxonomy overrides -- for species where XC uses a different genus than eBird.
# Run a full ingest first; the MISSING MEDIA report at the end lists affected species.
# Research the correct XC genus at xeno-canto.org, then re-run:
just ingest --xc-override "comrav=Corvus:corax,amgos=Accipiter:gentilis" --skip-complete US-OR

# After ingest: dump local DB and restore in prod
pg_dump $DATABASE_URL | psql $PROD_DATABASE_URL
```

## External APIs

All three are **ingestion-only** -- hit them once to populate the DB and R2, never needed again at runtime.

### eBird API
- **Used for:** regional species checklists → preset groups (e.g. "Pacific Northwest")
- **Auth:** free API key, request at ebird.org/api/keygen
- **Key endpoint:** `GET /v2/product/spplist/{regionCode}` returns species codes for a region
- **Region codes:** e.g. `US-WA` for Washington state, `US-OR` for Oregon
- **Docs:** https://documenter.getpostman.com/view/664302/S1ENwy59

### Xeno-canto
- **Used for:** bird call/song recordings (the core quiz content)
- **Auth:** API key required (v3); free for registered members with verified email
- **Approach:** query separately for `type:song` and `type:call` using `gen:Genus sp:species` parsed from eBird's `SciName`; filter to quality A or B; stream download URL directly to R2 via `fetchAndUpload`
- **Key endpoint:** `GET https://xeno-canto.org/api/3/recordings?key={key}&query=type:call%20gen:Corvus%20sp:corax` -- **v3, not v2**
- **Do NOT use** `en:` -- XC uses different English names than eBird (e.g. "Northern Raven" vs "Common Raven", "Scrub Jay" vs "California Scrub-Jay"), causing silent empty results.
- **Taxonomy lag:** eBird reclassifies genera faster than XC updates. When `gen:+sp:` returns 0 results, use `--xc-override code=Genus:species` to supply the genus/species XC actually uses. Run `just ingest --skip-complete <region>` after a full ingest to see the miss report and identify which species need overrides.
- **Encoding:** use `url.PathEscape` (not `url.Values.Encode`) -- XC needs `%20` for spaces, not `+`.
- Do NOT use `q:A` in query -- filter client-side instead.
- **License:** all recordings are Creative Commons -- safe to store and serve
- **Docs:** https://xeno-canto.org/explore/api

### Macaulay Library (Cornell Lab)
- **Used for:** species photos shown on quiz reveal and for future image ID quizzes
- **Auth:** same eBird API key (`X-eBirdApiToken` header)
- **Approach:** search by species code, store top-rated photos in `species_images`; stream Macaulay CDN URL to R2 via `fetchAndUpload`
- **Key endpoint:** `GET https://search.macaulaylibrary.org/api/v1/search?taxonCode={speciesCode}&mediaType=photo&sort=rating_rank_desc&count={n}` -- **not api.ebird.org**
- **Response shape:** `{ results: { content: [ { assetId, userDisplayName, ... } ] } }`
- **Note:** same Cornell/eBird ecosystem -- one API key covers eBird checklists and Macaulay media
