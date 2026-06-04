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

Natural text keys on species/recordings/images are stable across DB resets -- re-ingesting into a fresh DB produces identical IDs, so cards/deck memberships survive. Cards are user-scoped; the species catalog is global.

## What's next

### 1. Magic link auth (non-Google users)
- **Provider:** Resend (free tier 3k emails/month; domain verification is a few DNS records)
- **Flow:** email → short-lived token → link to `/api/v1/auth/magic?token=...` → verify, upsert user, issue existing HttpOnly JWT cookie
- **Schema:** `magic_link_tokens (email, token, expires_at, used)`
- Coexists with Google OAuth, matched on email in `users` -- no passwords ever

### 2. Recording normalization
- Some recordings (e.g. American Barn Owl) are startlingly loud compared to others
- Goal: normalize audio levels to a consistent dB ceiling before storing in R2
- Options: process at ingest time (ffmpeg loudnorm filter) or run a one-off normalization job after the fact
- Needs: ffmpeg available in ingest environment, R2 re-upload of normalized files, decision on target loudness (e.g. -16 LUFS)

### 3. Explore -- region filter
- Region codes are NOT stored in the DB; intersect on demand instead
- Flow: user picks state → backend proxy → eBird `GET /v2/ref/region/list/subnational2/{stateCode}` → user picks county → `GET /v2/product/spplist/{countyCode}` → intersect species codes with our catalog
- Must proxy through backend to keep the eBird API key server-side
- `regionType`: `country`, `subnational1` (states), `subnational2` (counties)

### 4. About page (`/about`)
- What FlockDeck is and how it works
- Credit/attribution for eBird, Xeno-canto (CC-licensed recordings), Macaulay Library photos
- Link to source repos / contact

### 5. Stats page (`/stats`)
- Per-user aggregate stats pulled from `cards` table: total reviews, correct rate, cards by FSRS state (new/learning/review/relearning)
- Needs a new backend endpoint (e.g. `GET /api/v1/stats`) -- no new schema required
- Possible charts: reviews over time, due forecast, accuracy by deck or species

## Key non-obvious choices
- OAuth state stored as short-lived cookie (5min) to prevent CSRF -- verified on callback
- `UpsertUser` updates name/picture on every login so Google profile changes sync
- SvelteKit mounts into `<div style="display: contents">` -- `#app` CSS is dead; use `.app-container` in `+layout.svelte`
- WaveSurfer: CDN URLs may lack CORS headers. Pass `media: audio` (native `<audio>`, no XHR) + fake `peaks` so bars draw immediately
- Quiz auto-rating: no self-reporting; `correct ? 3 : 1` posted to `/rate`. Correct = `selected.ebird_code === card.ebird_code`
- R2 ingest workflow: ingest locally (writes to shared R2 bucket) → dump DB → restore in prod. No per-environment media copies
- TanStack Query v6: `createQuery(() => ({...}))` -- options wrapped in a function so Svelte 5 rune reads are tracked. Returns a reactive object (not a store) -- access `.data`/`.isPending` without `$`. Same pattern for `createMutation`
- `GET /api/v1/species` pagination: `next`/`previous` are absolute URLs built from `r.Host` + `X-Forwarded-Proto`; `count` via `COUNT(*) OVER()` window function
- `GET /api/v1/species/all` (unpaginated) feeds all client-side filtering: TanStack key `['species', 'all']`, `staleTime: Infinity`, fetched once per session, shared by Explore and deck typeahead. If the catalog outgrows regional scale, revert to the server-side paginated search (still in place)
- Admin routes: backend `requireAdmin` middleware + `/admin/+layout.svelte` redirects to `/` if `!$auth?.is_admin`. Toggle admin via `PATCH /api/v1/admin/users/{id}`

## Known issues (deferred)
- **OAuth "invalid state" with installed Chrome PWA** -- logging in from both a Chrome tab and the shortcut app white-screens on the callback state check. `site.webmanifest` has no `scope`, so Chrome link-capturing pulls the Google redirect into the app window. Unconfirmed: (H1) same-profile clobbering of the single fixed-name `oauth_state` cookie vs (H2) PWA in a different Chrome profile = different cookie jar = cookie missing. Discriminator: log whether the cookie is *missing* (H2) or *mismatched* (H1) in the callback. Fixes: H1 → per-flow `oauth_state_<state>` cookies deleted after use (also closes the 5-min replay window); H2 → HMAC-signed state or manifest `scope` change. Deferred as an extreme edge case.

## Ingest workflow
Ingest locally (writes to shared R2 bucket) → `pg_dump $DATABASE_URL | psql $PROD_DATABASE_URL`. See `just ingest --help` for flags (`--species`, `--skip-complete`, `--max-recordings`).

XC taxonomy overrides: when XC uses a different genus than eBird, run full ingest first -- the MISSING MEDIA report lists affected species -- then re-run with `--xc-override "code=Genus:species"`.

## External APIs
All three are **ingestion-only** -- never hit at runtime.

- **eBird:** regional checklists → preset decks
- **Xeno-canto (v3, not v2):** query `gen:Genus sp:species` -- **do NOT use `en:`** (XC English names differ from eBird → silent empty results); **do NOT use `q:A`** (filter quality client-side); use `url.PathEscape` not `url.Values.Encode` (XC needs `%20` not `+`)
- **Macaulay Library:** same eBird key, `X-eBirdApiToken` header; host is **search.macaulaylibrary.org**, not api.ebird.org
