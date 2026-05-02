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

## What's next
- [ ] Google Cloud Console: create OAuth credentials, fill `.env`
- [ ] Svelte frontend scaffold in `frontend/`
- [ ] "Login with Google" button → verify full auth flow end-to-end
- [ ] Ingestion script: eBird API for species lists, xeno-canto for recordings (A/B quality only), Macaulay Library for images
- [ ] FSRS implementation and quiz loop endpoints
- [ ] Group management (admin: create presets; users: create custom groups)
- [ ] Admin UI for catalog management

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
createdb lifer_dev
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
