# FlockDeck -- Claude Code Context

## What this is
Spaced repetition web app for bird song/call and image ID practice. Hear a recording or see a photo, type the species via typeahead, reveal, advance. Rating is automatic (correct → Good/3, wrong → Again/1); FSRS drives scheduling. Decks (eBird-sourced region presets or user-created) scope practice. Live at flockdeck.com.

## Stack
- **Go + chi + sqlc + PostgreSQL** -- no ORM; sqlc generates typed Go from raw SQL
- **SvelteKit** (adapter-static, SPA mode) -- `ssr = false` in `+layout.ts`, `fallback: 'index.html'`
- **Google OAuth → HttpOnly cookie JWT** (30-day expiry, cookie and claim set separately)
- **FSRS** via `github.com/open-spaced-repetition/go-fsrs/v3`
- **Cloudflare R2** for all media -- full public URLs (`https://media.flockdeck.com/...`) stored in `file_path`; every environment shares one bucket

## Working on it
- Task runner: read `Justfile` for all commands. Key: `just generate` (sqlc) must run after any schema change
- After making changes, review this file: remove anything completed or now derivable from the code; add only what's non-obvious or future work
- `go.mod` is in `backend/`, not repo root (monorepo with `frontend/` alongside)
- Generated sqlc files (`backend/internal/store/*.go` except `queries/`) are committed -- don't `.gitignore` them
- Docker Compose postgres on port **5435**; backend on `$PORT` (8081 in `.env.example`)
- New machine: clone → `cp .env.example .env` (fill in, incl. `R2_*`) → `docker compose up -d` → `just migrate-up` → `just run`
- Deploy: push to main → GH Actions tests + SSH deploy to DigitalOcean droplet (`flockdeck.service`). nginx serves `frontend/build` (SPA `try_files` fallback) and proxies `/api/` + `/health` to Go on 8082. Cloudflare in front (Flexible SSL -- origin is HTTP only; Go does NOT serve the frontend in prod)

## Data model
- `users` -- google_id, email, name, picture, is_admin; BIGSERIAL PK
- `species` -- **`ebird_code TEXT PRIMARY KEY`**, common_name, scientific_name
- `species_recordings` -- **`xeno_canto_id TEXT PK`**, species_code FK, file_path (R2 URL), quality (A-E), type ("song"/"call"), credit, locked
- `species_images` -- **`macaulay_id TEXT PK`**, species_code FK, file_path (R2 URL), credit, locked
- `decks` -- preset (eBird-sourced, `owner_id IS NULL`) or user-created; BIGSERIAL PK
- `deck_species` -- join table; PK `(deck_id, species_code)`
- `cards` -- user × species × lane with FSRS fields; UNIQUE `(user_id, species_code, lane)`
- `user_species_preferences` -- per-user audio/image lane toggles per species
- `review_log` -- one row per quiz answer: rating, `guessed_species_code` (NULL = "I don't know"; SET NULL on species delete), `media_id` (xeno-canto/macaulay ID shown, deliberately no FK), reviewed_at. Written in the same tx as the FSRS card update; feeds `/stats`

Natural text keys on species/recordings/images are stable across DB resets -- re-ingesting into a fresh DB produces identical IDs, so cards/deck memberships survive. Cards are user-scoped; the species catalog is global.

## What's next

### 1. Magic link auth (non-Google users)
- **Provider:** Resend (free tier 3k emails/month; domain verification is a few DNS records)
- **Flow:** email → short-lived token → link to `/api/v1/auth/magic?token=...` → verify, upsert user, issue existing HttpOnly JWT cookie
- **Schema:** `magic_link_tokens (email, token, expires_at, used)`
- Coexists with Google OAuth, matched on email in `users` -- no passwords ever
- **`/about` feedback form (follow-up):** the `/about` page currently uses a plain `mailto:` to a plus-alias (`nakamajamey+flockdeck@gmail.com`) -- filterable/disposable since the repo is public. Once Resend is wired up here, replace it with a feedback form that POSTs to the backend and emails via Resend, keeping the address out of the client entirely

### 2. Explore -- region filter
- Region codes are NOT stored in the DB; intersect on demand instead
- Flow: user picks state → backend proxy → eBird `GET /v2/ref/region/list/subnational2/{stateCode}` → user picks county → `GET /v2/product/spplist/{countyCode}` → intersect species codes with our catalog
- Must proxy through backend to keep the eBird API key server-side
- `regionType`: `country`, `subnational1` (states), `subnational2` (counties)

### 3. Run the transcode backfill
- Ingest transcodes and peak-normalizes every new recording, and extracts waveform peaks, but stored recordings from before that shipped are untouched: most are still the original uncompressed audio with a NULL `peaks` column, and the player falls back to generated bars for them
- `just transcode --limit 50` first to size the sweep against production (dry run, nothing written), then `just transcode --apply` for the full backfill. `ListRecordingsForTranscode` only returns rows with `peaks IS NULL`, so `--limit` always samples real remaining work; the backfill is complete once a sweep returns zero rows
- The `just transcode` recipe `cd`s into `backend/` first, same as `ingest` -- a `--file` path for local single-file transcoding needs to be absolute or given relative to `backend/`, not the repo root
- `generatePeaks` in `frontend/src/components/WavePlayer.svelte` stays permanently: `RecordingsList.svelte`'s admin view renders `<WavePlayer>` with no `peaks` prop (`GetSpeciesRecordings` doesn't select the column), so it always needs the generated fallback

## Key non-obvious choices
- OAuth state stored as short-lived cookie (5min) to prevent CSRF -- verified on callback
- `UpsertUser` updates name/picture on every login so Google profile changes sync
- SvelteKit mounts into `<div style="display: contents">` -- `#app` CSS is dead; use `.app-container` in `+layout.svelte`
- WavePlayer never decodes audio client-side: it hands WaveSurfer a native `<audio>` element (`media: audio` -- no XHR, so no CORS dependency) plus precomputed peaks from the card payload; recordings that predate the peaks backfill fall back to generated bars
- Recordings are stored mono at 96 kbps mp3, peak-normalized to -1 dBFS, transcoded at ingest by `internal/audio`. `just transcode` re-runs that pass over already-stored objects, overwriting in place at the existing key so `file_path` never changes; it's a dry run unless `--apply` is passed
- Quiz auto-rating: no self-reporting; `correct ? 3 : 1` posted to `/rate` along with `guessed_species_code` (null = skip) and `media_id` -- both optional so old clients keep working
- `/stats` is computed live per request (every query `WHERE user_id`); lane tabs are `?lane=` on the same endpoint, ear-vs-eye exists only on the combined view. FSRS retrievability via go-fsrs `GetRetrievability` (package-level `statsFSRS`), so stats use the exact scheduler curve
- R2 ingest workflow: ingest runs from the laptop against whichever DB `DATABASE_URL` points at (these days usually prod directly); media goes to the one shared R2 bucket either way. No per-environment media copies
- TanStack Query v6: `createQuery(() => ({...}))` -- options wrapped in a function so Svelte 5 rune reads are tracked. Returns a reactive object (not a store) -- access `.data`/`.isPending` without `$`. Same pattern for `createMutation`
- `GET /api/v1/species` pagination: `next`/`previous` are absolute URLs built from `r.Host` + `X-Forwarded-Proto`; `count` via `COUNT(*) OVER()` window function
- `GET /api/v1/species/all` (unpaginated) feeds all client-side filtering: TanStack key `['species', 'all']`, `staleTime: Infinity`, fetched once per session, shared by Explore and deck typeahead. If the catalog outgrows regional scale, revert to the server-side paginated search (still in place)
- Admin routes: backend `requireAdmin` middleware + `/admin/+layout.svelte` redirects to `/` if `!$auth?.is_admin`. Toggle admin via `PATCH /api/v1/admin/users/{id}`

## Known issues (deferred)
- **OAuth + installed Chrome PWA (mostly fixed)** -- per-flow `oauth_state_<state>` cookies (deleted on first use) fixed same-profile clobbering, and state failures now redirect to `/?error=auth_state` instead of white-screening. Remaining edge: PWA in a *different* Chrome profile (different cookie jar) still fails the state check, but lands on the login page, which displays a "sign-in didn't complete" notice from the `error` param. If it resurfaces: the callback logs whether the cookie was missing (different profile) vs mismatched.

## Ingest workflow
Run from the laptop with `DATABASE_URL` pointed at the target DB (commonly prod directly; the old `pg_dump | psql` dump→restore also works). Media lands in the shared R2 bucket regardless. See `just ingest --help` for flags (`--species`, `--skip-complete`, `--max-recordings`). Ctrl+c cancels in-flight work cleanly. New species metadata columns backfill with `just ingest --refresh-metadata` (DB-wide, taxonomy-only, seconds -- no regions or media; XC/R2 env not required). Note `--skip-complete` filters species BEFORE `UpsertSpecies`, so normal region runs don't refresh metadata for already-complete species.

XC taxonomy overrides: when XC uses a different genus than eBird, run full ingest first -- the MISSING MEDIA report lists affected species -- then re-run with `--xc-override "code=Genus:species"`.

## External APIs
All three are **ingestion-only** -- never hit at runtime.

- **eBird:** regional checklists → preset decks
- **Xeno-canto (v3, not v2):** query `gen:Genus sp:species` -- **do NOT use `en:`** (XC English names differ from eBird → silent empty results); **do NOT use `q:A`** (filter quality client-side); use `url.PathEscape` not `url.Values.Encode` (XC needs `%20` not `+`)
- **Macaulay Library:** same eBird key, `X-eBirdApiToken` header; host is **search.macaulaylibrary.org**, not api.ebird.org. Use **`/api/v2/search`** -- v1 is gone (404). v2 returns a **bare JSON array** (no `results.content` wrapper) and `assetId` as a **number**, formatted to string for `macaulay_id`
