# FlockDeck

Spaced repetition practice for bird song and call identification.

Play a recording, guess the species, reveal the answer, rate your confidence. Practice by region or build custom species groups. Scheduling powered by FSRS.

## Stack

- **Backend** -- Go, chi, sqlc, PostgreSQL
- **Frontend** -- Svelte (Vite)
- **Auth** -- Google OAuth

## Local setup

**Requirements:** Go 1.22+, Node 20+, PostgreSQL, [just](https://github.com/casey/just), [sqlc](https://sqlc.dev), [golang-migrate](https://github.com/golang-migrate/migrate)

```bash
brew install just sqlc golang-migrate
```

**First run:**

```bash
git clone https://github.com/jameynakama/flockdeck
cd flockdeck
cp .env.example .env   # fill in values
just migrate-up
just run               # backend on http://localhost:8080
```

**Google OAuth credentials:** [console.cloud.google.com](https://console.cloud.google.com) -- create a project, enable the Google+ API, create OAuth 2.0 credentials. Set the redirect URI to `http://localhost:8080/api/v1/auth/google/callback`.

## Commands

```
just            # run tests (default)
just run        # start backend and frontend servers
just build      # build binary to backend/bin/flockdeck
just migrate-up        # run pending migrations
just migrate-down      # roll back one migration
just generate          # regenerate sqlc types after schema changes
just migration name=X  # create new migration files
```

## Pushing catalog data to prod

After a local ingest run, transfer only the species/recordings/images tables (leave users, cards, groups untouched):

```bash
pg_dump --clean \
  --table=species \
  --table=species_recordings \
  --table=species_images \
  $DATABASE_URL > catalog.sql

psql $PROD_DATABASE_URL < catalog.sql
```

## Data sources

- **eBird API** -- regional species checklists (preset groups)
- **Xeno-canto** -- bird call/song recordings (CC licensed, A/B quality)
- **Macaulay Library** -- bird photos (Cornell Lab)
