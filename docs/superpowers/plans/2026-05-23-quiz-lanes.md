# Quiz Lanes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the `user × recording` card model with a two-lane system (`audio` / `image`), each keyed `user × species`, with independent FSRS scheduling, random media selection per review, and per-species lane preferences.

**Architecture:** Squash the existing 3 migrations into one clean schema. New `cards` table uses a `lane TEXT CHECK` column instead of `recording_id`. New `user_species_preferences` table drives card creation. Three new API endpoints (all auth-gated). Frontend Quiz becomes prop-driven (SvelteKit-ready). FSRS scheduling via `go-fsrs`.

**Tech Stack:** Go, chi, sqlc, pgx/v5, go-fsrs, Svelte 5, TypeScript

**Spec:** `docs/superpowers/specs/2026-05-23-quiz-lanes-design.md`

---

## File Map

| Action | Path | Responsibility |
|--------|------|----------------|
| Rewrite | `backend/migrations/001_initial.up.sql` | Squashed final schema |
| Rewrite | `backend/migrations/001_initial.down.sql` | Drop everything |
| Delete | `backend/migrations/002_*.sql` | Squashed away |
| Delete | `backend/migrations/003_*.sql` | Squashed away |
| Rewrite | `backend/internal/store/queries/cards.sql` | New card queries |
| Create | `backend/internal/store/queries/preferences.sql` | Preference queries |
| Regenerate | `backend/internal/store/*.go` | Run `just generate` |
| Add dep | `backend/go.mod`, `backend/go.sum` | go-fsrs |
| Modify | `backend/internal/api/router.go` | Use Querier interface; add routes |
| Create | `backend/internal/api/quiz.go` | GetNextCard + RateCard handlers |
| Create | `backend/internal/api/quiz_test.go` | Handler tests |
| Create | `backend/internal/api/preferences.go` | UpdatePreferences handler |
| Create | `backend/internal/api/preferences_test.go` | Handler tests |
| Modify | `frontend/src/types.ts` | Update BirdCard |
| Modify | `frontend/src/stores/session.ts` | Add lane field |
| Modify | `frontend/src/App.svelte` | Pass groupId + lane props to Quiz |
| Modify | `frontend/src/views/Dashboard.svelte` | Audio/Image buttons per group |
| Modify | `frontend/src/views/Quiz.svelte` | Props + real API fetch |
| Create | `frontend/src/components/ImageQuizCard.svelte` | Image lane quiz card |

---

## Task 1: Squash migrations into one clean schema

**Files:**
- Rewrite: `backend/migrations/001_initial.up.sql`
- Rewrite: `backend/migrations/001_initial.down.sql`
- Delete: `backend/migrations/002_add_recording_type.up.sql`
- Delete: `backend/migrations/002_add_recording_type.down.sql`
- Delete: `backend/migrations/003_rename_recordings_to_species_recordings.up.sql`
- Delete: `backend/migrations/003_rename_recordings_to_species_recordings.down.sql`

- [ ] **Step 1: Rewrite 001_initial.up.sql**

Replace the entire file with:

```sql
CREATE TABLE users (
    id         BIGSERIAL PRIMARY KEY,
    google_id  TEXT        NOT NULL UNIQUE,
    email      TEXT        NOT NULL UNIQUE,
    name       TEXT        NOT NULL,
    picture    TEXT        NOT NULL DEFAULT '',
    is_admin   BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE species (
    id              BIGSERIAL PRIMARY KEY,
    common_name     TEXT NOT NULL,
    scientific_name TEXT NOT NULL,
    ebird_code      TEXT NOT NULL UNIQUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE species_recordings (
    id             BIGSERIAL PRIMARY KEY,
    species_id     BIGINT      NOT NULL REFERENCES species(id),
    xeno_canto_id  TEXT        NOT NULL UNIQUE,
    file_path      TEXT        NOT NULL,
    quality        CHAR(1)     NOT NULL CHECK (quality IN ('A','B','C','D','E')),
    type           TEXT        NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE species_images (
    id           BIGSERIAL PRIMARY KEY,
    species_id   BIGINT      NOT NULL REFERENCES species(id),
    macaulay_id  TEXT        NOT NULL UNIQUE,
    file_path    TEXT        NOT NULL,
    credit       TEXT        NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE groups (
    id          BIGSERIAL PRIMARY KEY,
    name        TEXT    NOT NULL,
    description TEXT    NOT NULL DEFAULT '',
    is_preset   BOOLEAN NOT NULL DEFAULT FALSE,
    owner_id    BIGINT  REFERENCES users(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE group_species (
    group_id   BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    species_id BIGINT NOT NULL REFERENCES species(id) ON DELETE CASCADE,
    PRIMARY KEY (group_id, species_id)
);

CREATE TABLE cards (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    species_id  BIGINT      NOT NULL REFERENCES species(id) ON DELETE CASCADE,
    lane        TEXT        NOT NULL CHECK (lane IN ('audio', 'image')),
    stability   FLOAT       NOT NULL DEFAULT 0,
    difficulty  FLOAT       NOT NULL DEFAULT 0,
    due         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_review TIMESTAMPTZ,
    reps        INT         NOT NULL DEFAULT 0,
    lapses      INT         NOT NULL DEFAULT 0,
    state       SMALLINT    NOT NULL DEFAULT 0, -- 0=new 1=learning 2=review 3=relearning
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, species_id, lane)
);
CREATE INDEX idx_cards_user_lane_due ON cards(user_id, lane, due);

CREATE TABLE user_species_preferences (
    user_id       BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    species_id    BIGINT      NOT NULL REFERENCES species(id) ON DELETE CASCADE,
    audio_enabled BOOLEAN     NOT NULL DEFAULT TRUE,
    image_enabled BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, species_id)
);
```

- [ ] **Step 2: Rewrite 001_initial.down.sql**

```sql
DROP TABLE IF EXISTS user_species_preferences CASCADE;
DROP TABLE IF EXISTS cards CASCADE;
DROP TABLE IF EXISTS group_species CASCADE;
DROP TABLE IF EXISTS groups CASCADE;
DROP TABLE IF EXISTS species_images CASCADE;
DROP TABLE IF EXISTS species_recordings CASCADE;
DROP TABLE IF EXISTS species CASCADE;
DROP TABLE IF EXISTS users CASCADE;
```

- [ ] **Step 3: Delete the squashed migration files**

```bash
rm backend/migrations/002_add_recording_type.up.sql
rm backend/migrations/002_add_recording_type.down.sql
rm backend/migrations/003_rename_recordings_to_species_recordings.up.sql
rm backend/migrations/003_rename_recordings_to_species_recordings.down.sql
```

- [ ] **Step 4: Reset and re-apply migrations**

```bash
just migrate-down  # run enough times to get back to zero, or:
migrate -path backend/migrations -database "$DATABASE_URL" drop -f
just migrate-up
```

Expected: migrations run cleanly with no errors.

- [ ] **Step 5: Commit**

```bash
jj describe -m "Squash migrations: final schema with lanes and preferences"
jj new
```

---

## Task 2: Write sqlc queries

**Files:**
- Rewrite: `backend/internal/store/queries/cards.sql`
- Create: `backend/internal/store/queries/preferences.sql`

- [ ] **Step 1: Rewrite cards.sql**

```sql
-- name: GetNextDueCard :one
SELECT c.id, c.user_id, c.species_id, c.lane,
       c.stability, c.difficulty, c.due, c.last_review,
       c.reps, c.lapses, c.state, c.created_at,
       s.common_name, s.scientific_name
FROM cards c
JOIN species s ON s.id = c.species_id
JOIN group_species gs ON gs.species_id = c.species_id
WHERE c.user_id = $1
  AND gs.group_id = $2
  AND c.lane = $3
  AND c.due <= NOW()
ORDER BY c.due
LIMIT 1;

-- name: GetRandomRecording :one
SELECT file_path FROM species_recordings
WHERE species_id = $1
ORDER BY random()
LIMIT 1;

-- name: GetRandomImage :one
SELECT file_path FROM species_images
WHERE species_id = $1
ORDER BY random()
LIMIT 1;

-- name: GetCard :one
SELECT id, user_id, species_id, lane, stability, difficulty, due,
       last_review, reps, lapses, state, created_at
FROM cards
WHERE user_id = $1 AND species_id = $2 AND lane = $3;

-- name: UpdateCardSchedule :one
UPDATE cards
SET stability   = $4,
    difficulty  = $5,
    due         = $6,
    last_review = NOW(),
    reps        = reps + 1,
    lapses      = $7,
    state       = $8
WHERE user_id = $1 AND species_id = $2 AND lane = $3
RETURNING id, user_id, species_id, lane, stability, difficulty, due,
          last_review, reps, lapses, state, created_at;

-- name: UpsertCard :exec
INSERT INTO cards (user_id, species_id, lane)
VALUES ($1, $2, $3)
ON CONFLICT (user_id, species_id, lane) DO NOTHING;

-- name: DeleteCard :exec
DELETE FROM cards
WHERE user_id = $1 AND species_id = $2 AND lane = $3;
```

- [ ] **Step 2: Create preferences.sql**

```sql
-- name: UpsertPreferences :one
INSERT INTO user_species_preferences (user_id, species_id, audio_enabled, image_enabled)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id, species_id) DO UPDATE
SET audio_enabled = EXCLUDED.audio_enabled,
    image_enabled = EXCLUDED.image_enabled
RETURNING user_id, species_id, audio_enabled, image_enabled, created_at;

-- name: GetPreferences :one
SELECT user_id, species_id, audio_enabled, image_enabled, created_at
FROM user_species_preferences
WHERE user_id = $1 AND species_id = $2;
```

- [ ] **Step 3: Regenerate store types**

```bash
just generate
```

Expected: no errors. Files updated in `backend/internal/store/`. The generated types you'll use in handlers:

```go
// GetNextDueCardParams
type GetNextDueCardParams struct {
    UserID  int64  `db:"user_id"`
    GroupID int64  `db:"group_id"`
    Lane    string `db:"lane"`
}

// GetNextDueCardRow - has all card FSRS fields plus CommonName, ScientificName
// GetNextDueCard returns (GetNextDueCardRow, error)

// GetRandomRecording(ctx, speciesID int64) (string, error)
// GetRandomImage(ctx, speciesID int64) (string, error)

// GetCard(ctx, GetCardParams) (Card, error)
// UpdateCardSchedule(ctx, UpdateCardScheduleParams) (Card, error)
// UpsertCard(ctx, UpsertCardParams) error  -- :exec, no return value
// DeleteCard(ctx, DeleteCardParams) error

// UpsertPreferences(ctx, UpsertPreferencesParams) (UserSpeciesPreference, error)
// GetPreferences(ctx, GetPreferencesParams) (UserSpeciesPreference, error)
```

- [ ] **Step 4: Commit**

```bash
jj describe -m "Rewrite sqlc queries for lane-based cards and preferences"
jj new
```

---

## Task 3: Add go-fsrs dependency

**Files:**
- Modify: `backend/go.mod`, `backend/go.sum`

- [ ] **Step 1: Install go-fsrs**

```bash
cd backend && go get github.com/open-spaced-repetition/go-fsrs/v3
```

Expected: `go.mod` gains `github.com/open-spaced-repetition/go-fsrs/v3 v3.x.x`.

- [ ] **Step 2: Verify the import works**

```bash
cd backend && go build ./...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
jj describe -m "Add go-fsrs dependency"
jj new
```

---

## Task 4: Refactor Handler to accept Querier interface

The current `Handler` holds `queries *store.Queries` (concrete type). Changing it to `store.Querier` (interface) allows test stubs without a real DB.

**Files:**
- Modify: `backend/internal/api/router.go`

- [ ] **Step 1: Update Handler struct and NewRouter**

In `backend/internal/api/router.go`, change `queries *store.Queries` to `queries store.Querier`:

```go
package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jameynakama/lifer/internal/auth"
	"github.com/jameynakama/lifer/internal/store"
	"golang.org/x/oauth2"
)

type RouterConfig struct {
	Queries     store.Querier
	OAuthConfig *oauth2.Config
	JWTSecret   []byte
	FrontendURL string
}

type Handler struct {
	queries     store.Querier
	oauthConfig *oauth2.Config
	jwtSecret   []byte
	frontendURL string
}

func NewRouter(cfg RouterConfig) http.Handler {
	h := &Handler{
		queries:     cfg.Queries,
		oauthConfig: cfg.OAuthConfig,
		jwtSecret:   cfg.JWTSecret,
		frontendURL: cfg.FrontendURL,
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	r.Get("/health", h.healthCheck)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/auth/google", h.googleLogin)
		r.Get("/auth/google/callback", h.googleCallback)

		r.Group(func(r chi.Router) {
			r.Use(auth.RequireAuth(cfg.JWTSecret))
			r.Get("/me", h.getMe)
		})
	})

	return r
}

func (h *Handler) healthCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}
```

- [ ] **Step 2: Verify build**

```bash
cd backend && go build ./...
```

Expected: no errors. `cmd/server/main.go` passes `*store.Queries` which implements `store.Querier`, so it satisfies the interface.

- [ ] **Step 3: Commit**

```bash
jj describe -m "Use Querier interface in Handler for testability"
jj new
```

---

## Task 5: Implement GET /api/v1/groups/:id/next

**Files:**
- Create: `backend/internal/api/quiz.go`
- Create: `backend/internal/api/quiz_test.go`

- [ ] **Step 1: Write the failing test**

Create `backend/internal/api/quiz_test.go`:

```go
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jameynakama/lifer/internal/auth"
	"github.com/jameynakama/lifer/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubQuerier embeds store.Querier (nil) so unimplemented methods panic if called.
// Tests override only the methods they need.
type stubQuerier struct {
	store.Querier
	getNextDueCard    func(ctx context.Context, arg store.GetNextDueCardParams) (store.GetNextDueCardRow, error)
	getRandomRecording func(ctx context.Context, speciesID int64) (string, error)
	getRandomImage    func(ctx context.Context, speciesID int64) (string, error)
	getCard           func(ctx context.Context, arg store.GetCardParams) (store.Card, error)
	updateCardSchedule func(ctx context.Context, arg store.UpdateCardScheduleParams) (store.Card, error)
}

func (s *stubQuerier) GetNextDueCard(ctx context.Context, arg store.GetNextDueCardParams) (store.GetNextDueCardRow, error) {
	return s.getNextDueCard(ctx, arg)
}
func (s *stubQuerier) GetRandomRecording(ctx context.Context, speciesID int64) (string, error) {
	return s.getRandomRecording(ctx, speciesID)
}
func (s *stubQuerier) GetRandomImage(ctx context.Context, speciesID int64) (string, error) {
	return s.getRandomImage(ctx, speciesID)
}
func (s *stubQuerier) GetCard(ctx context.Context, arg store.GetCardParams) (store.Card, error) {
	return s.getCard(ctx, arg)
}
func (s *stubQuerier) UpdateCardSchedule(ctx context.Context, arg store.UpdateCardScheduleParams) (store.Card, error) {
	return s.updateCardSchedule(ctx, arg)
}

func makeHandler(q store.Querier) *Handler {
	return &Handler{queries: q}
}

// injectUserID adds a user ID to the request context, simulating RequireAuth.
func injectUserID(r *http.Request, userID int64) *http.Request {
	ctx := context.WithValue(r.Context(), auth.UserIDKey(), userID)
	return r.WithContext(ctx)
}

// withChiParam sets a chi URL param on the request.
func withChiParam(r *http.Request, key, val string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, val)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func TestGetNextCard_Audio_ReturnsDueCard(t *testing.T) {
	due := pgtype.Timestamptz{}
	require.NoError(t, due.Scan(time.Now().Add(-time.Hour)))

	q := &stubQuerier{
		getNextDueCard: func(_ context.Context, arg store.GetNextDueCardParams) (store.GetNextDueCardRow, error) {
			assert.Equal(t, int64(1), arg.UserID)
			assert.Equal(t, int64(42), arg.GroupID)
			assert.Equal(t, "audio", arg.Lane)
			return store.GetNextDueCardRow{
				SpeciesID:      99,
				Lane:           "audio",
				CommonName:     "Spotted Towhee",
				ScientificName: "Pipilo maculatus",
				Due:            due,
			}, nil
		},
		getRandomRecording: func(_ context.Context, speciesID int64) (string, error) {
			assert.Equal(t, int64(99), speciesID)
			return "https://xeno-canto.org/123/download", nil
		},
		getRandomImage: func(_ context.Context, speciesID int64) (string, error) {
			return "https://cdn.example.com/photo.jpg", nil
		},
	}

	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/groups/42/next?lane=audio", nil)
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.getNextCard(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var body nextCardResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, int64(99), body.SpeciesID)
	assert.Equal(t, "Spotted Towhee", body.CommonName)
	assert.Equal(t, "https://xeno-canto.org/123/download", body.MediaURL)
	assert.Equal(t, "https://cdn.example.com/photo.jpg", body.PhotoURL)
	assert.Equal(t, "audio", body.Lane)
}

func TestGetNextCard_NothingDue_Returns204(t *testing.T) {
	q := &stubQuerier{
		getNextDueCard: func(_ context.Context, _ store.GetNextDueCardParams) (store.GetNextDueCardRow, error) {
			return store.GetNextDueCardRow{}, pgx.ErrNoRows
		},
	}

	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/groups/42/next?lane=audio", nil)
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.getNextCard(w, r)

	assert.Equal(t, http.StatusNoContent, w.Code)
}
```

- [ ] **Step 2: Run the test to confirm it fails**

```bash
cd backend && go test ./internal/api/... -v -run TestGetNextCard
```

Expected: compilation error (handler not defined yet).

- [ ] **Step 3: Export UserIDKey from auth package**

The test uses `auth.UserIDKey()`. Currently `userIDKey` is unexported. Add a getter to `backend/internal/auth/middleware.go`:

```go
// UserIDKey returns the context key used by RequireAuth so tests can inject a user ID.
func UserIDKey() any {
	return userIDKey
}
```

- [ ] **Step 4: Create quiz.go with GetNextCard handler**

Create `backend/internal/api/quiz.go`:

```go
package api

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jameynakama/lifer/internal/auth"
	"github.com/jameynakama/lifer/internal/store"
	"github.com/jackc/pgx/v5"
)

type nextCardResponse struct {
	SpeciesID      int64  `json:"species_id"`
	CommonName     string `json:"common_name"`
	ScientificName string `json:"scientific_name"`
	MediaURL       string `json:"media_url"`
	PhotoURL       string `json:"photo_url"`
	Lane           string `json:"lane"`
}

func (h *Handler) getNextCard(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromCtx(r.Context())

	groupID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid group id", http.StatusBadRequest)
		return
	}

	lane := r.URL.Query().Get("lane")
	if lane != "audio" && lane != "image" {
		http.Error(w, "lane must be audio or image", http.StatusBadRequest)
		return
	}

	card, err := h.queries.GetNextDueCard(r.Context(), store.GetNextDueCardParams{
		UserID:  userID,
		GroupID: groupID,
		Lane:    lane,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if err != nil {
		log.Printf("GetNextDueCard error: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	var mediaURL string
	if lane == "audio" {
		mediaURL, err = h.queries.GetRandomRecording(r.Context(), card.SpeciesID)
	} else {
		mediaURL, err = h.queries.GetRandomImage(r.Context(), card.SpeciesID)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		log.Printf("no media for species %d lane %s", card.SpeciesID, lane)
		http.Error(w, "no media available", http.StatusInternalServerError)
		return
	}
	if err != nil {
		log.Printf("GetRandom media error: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	// For audio lane, fetch a separate photo for the reveal.
	// For image lane, the reveal shows the same photo the user just saw.
	photoURL := mediaURL
	if lane == "audio" {
		photoURL, err = h.queries.GetRandomImage(r.Context(), card.SpeciesID)
		if errors.Is(err, pgx.ErrNoRows) {
			photoURL = ""
		} else if err != nil {
			log.Printf("GetRandomImage error: %v", err)
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}
	}

	writeJSON(w, http.StatusOK, nextCardResponse{
		SpeciesID:      card.SpeciesID,
		CommonName:     card.CommonName,
		ScientificName: card.ScientificName,
		MediaURL:       mediaURL,
		PhotoURL:       photoURL,
		Lane:           lane,
	})
}
```

- [ ] **Step 5: Run the test again**

```bash
cd backend && go test ./internal/api/... -v -run TestGetNextCard
```

Expected: PASS for both tests.

- [ ] **Step 6: Commit**

```bash
jj describe -m "Implement GET /groups/:id/next handler"
jj new
```

---

## Task 6: Implement POST /api/v1/groups/:id/rate

**Files:**
- Modify: `backend/internal/api/quiz.go`
- Modify: `backend/internal/api/quiz_test.go`

- [ ] **Step 1: Write the failing test**

Add to `backend/internal/api/quiz_test.go`:

```go
func TestRateCard_UpdatesSchedule(t *testing.T) {
	now := time.Now()
	due := pgtype.Timestamptz{}
	require.NoError(t, due.Scan(now.Add(24*time.Hour)))

	updatedCard := store.Card{
		UserID:    1,
		SpeciesID: 99,
		Lane:      "audio",
		Stability: 2.5,
		Due:       due,
		State:     2,
	}

	q := &stubQuerier{
		getCard: func(_ context.Context, arg store.GetCardParams) (store.Card, error) {
			assert.Equal(t, int64(1), arg.UserID)
			assert.Equal(t, int64(99), arg.SpeciesID)
			assert.Equal(t, "audio", arg.Lane)
			return store.Card{
				UserID:    1,
				SpeciesID: 99,
				Lane:      "audio",
				State:     0,
			}, nil
		},
		updateCardSchedule: func(_ context.Context, arg store.UpdateCardScheduleParams) (store.Card, error) {
			assert.Equal(t, int64(1), arg.UserID)
			assert.Equal(t, int64(99), arg.SpeciesID)
			assert.Equal(t, "audio", arg.Lane)
			assert.Greater(t, arg.Stability, 0.0)
			return updatedCard, nil
		},
	}

	h := makeHandler(q)
	body := `{"species_id":99,"lane":"audio","rating":3}`
	r := httptest.NewRequest(http.MethodPost, "/api/v1/groups/42/rate", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.rateCard(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var got store.Card
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	assert.Equal(t, int64(99), got.SpeciesID)
	assert.Equal(t, "audio", got.Lane)
}

func TestRateCard_InvalidRating_Returns400(t *testing.T) {
	h := makeHandler(&stubQuerier{})
	body := `{"species_id":99,"lane":"audio","rating":9}`
	r := httptest.NewRequest(http.MethodPost, "/api/v1/groups/42/rate", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.rateCard(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
```

Add `"strings"` to the test file imports.

- [ ] **Step 2: Run to confirm failure**

```bash
cd backend && go test ./internal/api/... -v -run TestRateCard
```

Expected: compilation error (rateCard not defined).

- [ ] **Step 3: Add rateCard to quiz.go**

Add to `backend/internal/api/quiz.go` (also add imports: `"encoding/json"`, `"time"`, `fsrs "github.com/open-spaced-repetition/go-fsrs/v3"`, `"github.com/jackc/pgx/v5/pgtype"`):

```go
type rateCardRequest struct {
	SpeciesID int64  `json:"species_id"`
	Lane      string `json:"lane"`
	Rating    int    `json:"rating"`
}

func (h *Handler) rateCard(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromCtx(r.Context())

	var req rateCardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Rating < 1 || req.Rating > 4 {
		http.Error(w, "rating must be 1-4", http.StatusBadRequest)
		return
	}
	if req.Lane != "audio" && req.Lane != "image" {
		http.Error(w, "lane must be audio or image", http.StatusBadRequest)
		return
	}

	current, err := h.queries.GetCard(r.Context(), store.GetCardParams{
		UserID:    userID,
		SpeciesID: req.SpeciesID,
		Lane:      req.Lane,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, "card not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("GetCard error: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	f := fsrs.DefaultParam()
	fsrsCard := fsrs.Card{
		Stability:  current.Stability,
		Difficulty: current.Difficulty,
		Reps:       uint64(current.Reps),
		Lapses:     uint64(current.Lapses),
		State:      fsrs.State(current.State),
	}
	if current.LastReview.Valid {
		fsrsCard.LastReview = current.LastReview.Time
	}
	if current.Due.Valid {
		fsrsCard.Due = current.Due.Time
	}

	now := time.Now()
	result := f.Repeat(fsrsCard, now)[fsrs.Rating(req.Rating)].Card

	due := pgtype.Timestamptz{}
	if err := due.Scan(result.Due); err != nil {
		log.Printf("scan due error: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	updated, err := h.queries.UpdateCardSchedule(r.Context(), store.UpdateCardScheduleParams{
		UserID:     userID,
		SpeciesID:  req.SpeciesID,
		Lane:       req.Lane,
		Stability:  result.Stability,
		Difficulty: result.Difficulty,
		Due:        due,
		Lapses:     int32(result.Lapses),
		State:      int16(result.State),
	})
	if err != nil {
		log.Printf("UpdateCardSchedule error: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, updated)
}
```

- [ ] **Step 4: Run the test**

```bash
cd backend && go test ./internal/api/... -v -run TestRateCard
```

Expected: PASS for both tests.

- [ ] **Step 5: Commit**

```bash
jj describe -m "Implement POST /groups/:id/rate handler with FSRS scheduling"
jj new
```

---

## Task 7: Implement PUT /api/v1/species/:id/preferences

**Files:**
- Create: `backend/internal/api/preferences.go`
- Create: `backend/internal/api/preferences_test.go`

- [ ] **Step 1: Write the failing test**

Create `backend/internal/api/preferences_test.go`:

```go
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jameynakama/lifer/internal/auth"
	"github.com/jameynakama/lifer/internal/store"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// prefStubQuerier extends stubQuerier with preference + card methods.
type prefStubQuerier struct {
	store.Querier
	upsertPreferences func(ctx context.Context, arg store.UpsertPreferencesParams) (store.UserSpeciesPreference, error)
	upsertCard        func(ctx context.Context, arg store.UpsertCardParams) error
	deleteCard        func(ctx context.Context, arg store.DeleteCardParams) error
}

func (s *prefStubQuerier) UpsertPreferences(ctx context.Context, arg store.UpsertPreferencesParams) (store.UserSpeciesPreference, error) {
	return s.upsertPreferences(ctx, arg)
}
func (s *prefStubQuerier) UpsertCard(ctx context.Context, arg store.UpsertCardParams) error {
	return s.upsertCard(ctx, arg)
}
func (s *prefStubQuerier) DeleteCard(ctx context.Context, arg store.DeleteCardParams) error {
	return s.deleteCard(ctx, arg)
}

func TestUpdatePreferences_EnablesBothLanes(t *testing.T) {
	createdAt := pgtype.Timestamptz{}
	require.NoError(t, createdAt.Scan(nil))

	upsertCalls := map[string]bool{}
	q := &prefStubQuerier{
		upsertPreferences: func(_ context.Context, arg store.UpsertPreferencesParams) (store.UserSpeciesPreference, error) {
			assert.Equal(t, int64(1), arg.UserID)
			assert.Equal(t, int64(55), arg.SpeciesID)
			assert.True(t, arg.AudioEnabled)
			assert.True(t, arg.ImageEnabled)
			return store.UserSpeciesPreference{
				UserID:       arg.UserID,
				SpeciesID:    arg.SpeciesID,
				AudioEnabled: arg.AudioEnabled,
				ImageEnabled: arg.ImageEnabled,
			}, nil
		},
		upsertCard: func(_ context.Context, arg store.UpsertCardParams) error {
			upsertCalls[arg.Lane] = true
			return nil
		},
		deleteCard: func(_ context.Context, _ store.DeleteCardParams) error {
			t.Fatal("deleteCard should not be called when enabling lanes")
			return nil
		},
	}

	h := makeHandler(q)
	body := `{"audio_enabled":true,"image_enabled":true}`
	r := httptest.NewRequest(http.MethodPut, "/api/v1/species/55/preferences", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "55")
	w := httptest.NewRecorder()

	h.updatePreferences(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, upsertCalls["audio"], "should upsert audio card")
	assert.True(t, upsertCalls["image"], "should upsert image card")
	var pref store.UserSpeciesPreference
	require.NoError(t, json.NewDecoder(w.Body).Decode(&pref))
	assert.Equal(t, int64(55), pref.SpeciesID)
}

func TestUpdatePreferences_DisablesAudioLane(t *testing.T) {
	deleteCalls := map[string]bool{}
	q := &prefStubQuerier{
		upsertPreferences: func(_ context.Context, arg store.UpsertPreferencesParams) (store.UserSpeciesPreference, error) {
			return store.UserSpeciesPreference{
				UserID: arg.UserID, SpeciesID: arg.SpeciesID,
				AudioEnabled: arg.AudioEnabled, ImageEnabled: arg.ImageEnabled,
			}, nil
		},
		upsertCard: func(_ context.Context, arg store.UpsertCardParams) error {
			assert.Equal(t, "image", arg.Lane, "only image should be upserted")
			return nil
		},
		deleteCard: func(_ context.Context, arg store.DeleteCardParams) error {
			deleteCalls[arg.Lane] = true
			return nil
		},
	}

	h := makeHandler(q)
	body := `{"audio_enabled":false,"image_enabled":true}`
	r := httptest.NewRequest(http.MethodPut, "/api/v1/species/55/preferences", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "55")
	w := httptest.NewRecorder()

	h.updatePreferences(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, deleteCalls["audio"], "should delete audio card")
	assert.False(t, deleteCalls["image"], "should not delete image card")
}
```

- [ ] **Step 2: Run to confirm failure**

```bash
cd backend && go test ./internal/api/... -v -run TestUpdatePreferences
```

Expected: compilation error.

- [ ] **Step 3: Create preferences.go**

Create `backend/internal/api/preferences.go`:

```go
package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jameynakama/lifer/internal/auth"
	"github.com/jameynakama/lifer/internal/store"
)

type preferencesRequest struct {
	AudioEnabled bool `json:"audio_enabled"`
	ImageEnabled bool `json:"image_enabled"`
}

func (h *Handler) updatePreferences(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromCtx(r.Context())

	speciesID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid species id", http.StatusBadRequest)
		return
	}

	var req preferencesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	pref, err := h.queries.UpsertPreferences(r.Context(), store.UpsertPreferencesParams{
		UserID:       userID,
		SpeciesID:    speciesID,
		AudioEnabled: req.AudioEnabled,
		ImageEnabled: req.ImageEnabled,
	})
	if err != nil {
		log.Printf("UpsertPreferences error: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	for _, lane := range []string{"audio", "image"} {
		enabled := (lane == "audio" && req.AudioEnabled) || (lane == "image" && req.ImageEnabled)
		if enabled {
			if err := h.queries.UpsertCard(r.Context(), store.UpsertCardParams{
				UserID:    userID,
				SpeciesID: speciesID,
				Lane:      lane,
			}); err != nil {
				log.Printf("UpsertCard error: %v", err)
				http.Error(w, "server error", http.StatusInternalServerError)
				return
			}
		} else {
			if err := h.queries.DeleteCard(r.Context(), store.DeleteCardParams{
				UserID:    userID,
				SpeciesID: speciesID,
				Lane:      lane,
			}); err != nil {
				log.Printf("DeleteCard error: %v", err)
				http.Error(w, "server error", http.StatusInternalServerError)
				return
			}
		}
	}

	writeJSON(w, http.StatusOK, pref)
}
```

- [ ] **Step 4: Run the test**

```bash
cd backend && go test ./internal/api/... -v -run TestUpdatePreferences
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
jj describe -m "Implement PUT /species/:id/preferences handler"
jj new
```

---

## Task 8: Register routes and verify full backend test suite

**Files:**
- Modify: `backend/internal/api/router.go`

- [ ] **Step 1: Add routes to NewRouter**

In `backend/internal/api/router.go`, add the three new routes inside the auth-gated group:

```go
r.Group(func(r chi.Router) {
    r.Use(auth.RequireAuth(cfg.JWTSecret))
    r.Get("/me", h.getMe)
    r.Get("/groups/{id}/next", h.getNextCard)
    r.Post("/groups/{id}/rate", h.rateCard)
    r.Put("/species/{id}/preferences", h.updatePreferences)
})
```

- [ ] **Step 2: Build to catch any import issues**

```bash
cd backend && go build ./...
```

Expected: no errors.

- [ ] **Step 3: Run full test suite**

```bash
just test
```

Expected: all tests pass.

- [ ] **Step 4: Commit**

```bash
jj describe -m "Register quiz and preferences routes"
jj new
```

---

## Task 9: Update frontend types and session store

**Files:**
- Modify: `frontend/src/types.ts`
- Modify: `frontend/src/stores/session.ts`

- [ ] **Step 1: Update BirdCard type in types.ts**

```typescript
export interface Stat {
    label: string;
    value: string | number;
}

export interface BirdCard {
    species_id: number;
    common_name: string;
    scientific_name: string;
    media_url: string;    // audio URL (audio lane) or photo URL (image lane)
    photo_url: string;    // always a species photo; used by RevealCard on audio reveal
    lane: 'audio' | 'image';
}

export interface Group {
    id: string;
    name: string;
    is_preset: boolean;
    audio_due: number;
    image_due: number;
}
```

Note: `Group` gains `audio_due` and `image_due` (separate due counts per lane) replacing the single `due_count`. The dashboard mock data will reflect this.

- [ ] **Step 2: Update session store**

Replace `frontend/src/stores/session.ts`:

```typescript
import { writable, type Writable } from "svelte/store";

type Lane = 'audio' | 'image';

type Session = {
    groupId: string | null;
    lane: Lane | null;
};

export const session: Writable<Session> = writable({ groupId: null, lane: null });
```

- [ ] **Step 3: Commit**

```bash
jj describe -m "Update frontend BirdCard type and session store for lanes"
jj new
```

---

## Task 10: Update Dashboard.svelte

Show separate Audio and Image practice buttons per group, each with their own due count.

**Files:**
- Modify: `frontend/src/views/Dashboard.svelte`

- [ ] **Step 1: Replace Dashboard.svelte**

```svelte
<script lang="ts">
  import type { Group } from '../types'
  import { session } from '../stores/session'
  import { view } from '../stores/view'
  import StatsBar from '../components/StatsBar.svelte'
  import GroupList from '../components/GroupList.svelte'

  const MOCK_GROUPS: Group[] = [
    { id: '1', name: 'Pacific Northwest', is_preset: true, audio_due: 8, image_due: 5 },
    { id: '2', name: 'My Warblers', is_preset: false, audio_due: 3, image_due: 0 },
  ]

  const groups = MOCK_GROUPS
  const totalDue = groups.reduce((sum, g) => sum + g.audio_due + g.image_due, 0)

  const stats = [
    { label: 'Due today', value: totalDue },
    { label: 'Day streak', value: 5 },
    { label: 'Species', value: 47 },
  ]

  function startPractice(group: Group, lane: 'audio' | 'image') {
    $session = { groupId: group.id, lane }
    $view = 'quiz'
  }
</script>

<div class="dashboard">
  <StatsBar {stats} />
  <GroupList {groups} onPractice={startPractice} />
</div>

<style>
  .dashboard {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }
</style>
```

- [ ] **Step 2: Update GroupList.svelte to pass lane**

Open `frontend/src/components/GroupList.svelte` and update the `onPractice` prop type and button calls to pass the lane. Check the current file first with `cat frontend/src/components/GroupList.svelte`, then update its `onPractice` signature to `(group: Group, lane: 'audio' | 'image') => void` and add Audio/Image buttons for each group:

```svelte
<script lang="ts">
  import type { Group } from '../types'

  let {
    groups,
    onPractice,
  }: {
    groups: Group[];
    onPractice: (group: Group, lane: 'audio' | 'image') => void;
  } = $props()
</script>

<div class="group-list">
  {#each groups as group}
    <div class="group-card">
      <div class="group-info">
        <span class="group-name">{group.name}</span>
        {#if group.is_preset}<span class="preset-badge">Preset</span>{/if}
      </div>
      <div class="group-actions">
        {#if group.audio_due > 0}
          <button class="btn-lane" onclick={() => onPractice(group, 'audio')}>
            🔊 Audio · {group.audio_due}
          </button>
        {/if}
        {#if group.image_due > 0}
          <button class="btn-lane" onclick={() => onPractice(group, 'image')}>
            👁 Image · {group.image_due}
          </button>
        {/if}
        {#if group.audio_due === 0 && group.image_due === 0}
          <span class="all-done">All done</span>
        {/if}
      </div>
    </div>
  {/each}
</div>

<style>
  .group-list {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }
  .group-card {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: 10px;
    padding: 0.875rem 1rem;
    display: flex;
    justify-content: space-between;
    align-items: center;
    box-shadow: var(--shadow);
  }
  .group-info {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }
  .group-name {
    font-size: 0.9375rem;
    font-weight: 600;
    color: var(--text);
  }
  .preset-badge {
    font-size: 0.625rem;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--text-muted);
    background: var(--border);
    border-radius: 4px;
    padding: 0.125rem 0.375rem;
  }
  .group-actions {
    display: flex;
    gap: 0.5rem;
  }
  .btn-lane {
    background: var(--surface);
    border: 1px solid var(--border);
    color: var(--text-secondary);
    border-radius: 8px;
    padding: 0.375rem 0.625rem;
    font-size: 0.75rem;
    font-weight: 600;
    cursor: pointer;
    font-family: inherit;
    box-shadow: var(--shadow);
  }
  .all-done {
    font-size: 0.75rem;
    color: var(--text-muted);
  }
</style>
```

- [ ] **Step 3: Commit**

```bash
jj describe -m "Dashboard: separate audio/image practice buttons per group"
jj new
```

---

## Task 11: Update App.svelte to pass props to Quiz

**Files:**
- Modify: `frontend/src/App.svelte`

- [ ] **Step 1: Pass groupId and lane as props**

In `frontend/src/App.svelte`, replace the Quiz render line:

```svelte
{:else if $view === 'quiz'}
  <Quiz groupId={$session.groupId!} lane={$session.lane!} />
```

The full updated section (only the relevant part changes):

```svelte
{#if $view === 'dashboard'}
  <Dashboard />
{:else if $view === 'quiz'}
  <Quiz groupId={$session.groupId!} lane={$session.lane!} />
{/if}
```

Also add the session import at the top of App.svelte's script block:

```svelte
<script lang="ts">
  import { auth } from './stores/auth'
  import { view } from './stores/view'
  import { session } from './stores/session'
  ...
</script>
```

- [ ] **Step 2: Commit**

```bash
jj describe -m "App: pass groupId and lane props to Quiz"
jj new
```

---

## Task 12: Update Quiz.svelte with real API and props

**Files:**
- Modify: `frontend/src/views/Quiz.svelte`

- [ ] **Step 1: Replace Quiz.svelte**

```svelte
<script lang="ts">
  import type { BirdCard } from '../types'
  import { view } from '../stores/view'
  import QuizCard from '../components/QuizCard.svelte'
  import ImageQuizCard from '../components/ImageQuizCard.svelte'
  import RevealCard from '../components/RevealCard.svelte'
  import StatsBar from '../components/StatsBar.svelte'

  let { groupId, lane }: { groupId: string; lane: 'audio' | 'image' } = $props()

  let card: BirdCard | null = $state(null)
  let revealed = $state(false)
  let done = $state(false)
  let reviewed = $state(0)
  let loading = $state(true)
  let error = $state('')

  async function fetchNext() {
    loading = true
    error = ''
    try {
      const res = await fetch(`/api/v1/groups/${groupId}/next?lane=${lane}`)
      if (res.status === 204) {
        done = true
        card = null
        return
      }
      if (!res.ok) throw new Error(`Server error ${res.status}`)
      card = await res.json()
    } catch (e) {
      error = 'Failed to load next card.'
    } finally {
      loading = false
    }
  }

  async function onRate(rating: number) {
    if (!card) return
    try {
      await fetch(`/api/v1/groups/${groupId}/rate`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ species_id: card.species_id, lane: card.lane, rating }),
      })
    } catch {
      // non-fatal: FSRS miss is recoverable on next session
    }
    reviewed += 1
    revealed = false
    await fetchNext()
  }

  function onReveal() {
    revealed = true
  }

  const stats = $derived([
    { label: 'Reviewed', value: reviewed },
    { label: 'Lane', value: lane === 'audio' ? '🔊 Audio' : '👁 Image' },
  ])

  // Kick off first fetch
  fetchNext()
</script>

<div class="quiz">
  <StatsBar {stats} />

  {#if loading}
    <p class="status">Loading...</p>
  {:else if error}
    <p class="status error">{error}</p>
  {:else if done}
    <div class="done">
      <p>All done for now!</p>
      <button onclick={() => $view = 'dashboard'}>Back to dashboard</button>
    </div>
  {:else if card}
    {#if revealed}
      <RevealCard {card} {onRate} />
    {:else if lane === 'audio'}
      <QuizCard {card} {onReveal} />
    {:else}
      <ImageQuizCard {card} {onReveal} />
    {/if}
  {/if}
</div>

<style>
  .quiz {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }
  .status {
    text-align: center;
    color: var(--text-muted);
    padding: 2rem 0;
  }
  .error {
    color: #b91c1c;
  }
  .done {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 1rem;
    padding: 2rem 0;
  }
  .done p {
    color: var(--text);
    font-size: 1rem;
    font-weight: 600;
  }
  .done button {
    background: var(--accent);
    color: #fff;
    border: none;
    border-radius: 10px;
    padding: 0.75rem 1.5rem;
    font-size: 0.9375rem;
    font-weight: 600;
    cursor: pointer;
    font-family: inherit;
  }
</style>
```

- [ ] **Step 2: Update QuizCard.svelte to use new BirdCard shape**

`QuizCard.svelte` uses `card.recording_path` -- replace with `card.media_url`:

```svelte
<audio controls src={card.media_url}>Your browser does not support audio.</audio>
```

- [ ] **Step 3: Update RevealCard.svelte to use new BirdCard shape**

`RevealCard.svelte` uses `card.photo_path` -- replace with `card.photo_url`:

```svelte
<img src={card.photo_url} alt={card.common_name} class="photo" />
```

- [ ] **Step 4: Commit**

```bash
jj describe -m "Quiz: real API fetch, prop-driven, lane-aware routing"
jj new
```

---

## Task 13: Create ImageQuizCard.svelte

**Files:**
- Create: `frontend/src/components/ImageQuizCard.svelte`

- [ ] **Step 1: Create the component**

```svelte
<script lang="ts">
  import type { BirdCard } from '../types'
  let { card, onReveal }: { card: BirdCard; onReveal: () => void } = $props()
  let guess: string = $state('')
</script>

<div class="quiz-card">
  <div class="image-wrapper">
    <img src={card.media_url} alt="Identify this bird" class="quiz-photo" />
  </div>
  <input
    bind:value={guess}
    type="text"
    placeholder="Type species name"
    class="guess-input"
  />
  <button class="btn-reveal" onclick={onReveal}>Reveal answer</button>
</div>

<style>
  .quiz-card {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }
  .image-wrapper {
    background: var(--surface);
    border-radius: 8px;
    overflow: hidden;
    box-shadow: var(--shadow);
  }
  .quiz-photo {
    width: 100%;
    display: block;
    max-height: 280px;
    object-fit: cover;
  }
  .guess-input {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: 8px;
    padding: 0.6875rem 0.875rem;
    width: 100%;
    font-size: 0.875rem;
    color: var(--text);
    font-family: inherit;
    outline: none;
  }
  .guess-input:focus {
    border-color: var(--accent);
  }
  .btn-reveal {
    background: var(--surface);
    border: 1px solid var(--border);
    color: var(--text-secondary);
    border-radius: 10px;
    padding: 0.75rem;
    width: 100%;
    font-size: 0.875rem;
    font-weight: 600;
    cursor: pointer;
    font-family: inherit;
    box-shadow: var(--shadow);
  }
</style>
```

- [ ] **Step 2: Run the frontend dev server and verify both lanes render**

```bash
just frontend
```

- Open the app in a browser.
- Confirm the dashboard shows Audio/Image buttons per group (mocked).
- Confirm clicking Audio navigates to quiz with audio player.
- Confirm clicking Image navigates to quiz with photo (the mock data won't load real URLs, but the component should render without errors in the console).

- [ ] **Step 3: Commit**

```bash
jj describe -m "Add ImageQuizCard component for image lane"
jj new
```

---

## Self-Review Notes

- All three API endpoints are behind `RequireAuth` (registered inside the auth group in Task 8).
- `UserIDKey()` export in Task 5 Step 3 is required for test compilation -- do not skip it.
- The `stubQuerier` in `quiz_test.go` and `prefStubQuerier` in `preferences_test.go` are separate because they override different method sets. Do not merge them.
- `UpsertCard` is `:exec` (no return value) -- the handler calls it but doesn't use a return.
- After Task 1, the migration tool's version tracking table (`schema_migrations`) may need clearing if you get a "dirty" state. Use `migrate -path backend/migrations -database "$DATABASE_URL" force 1` if needed.
- `go-fsrs` import path may be `github.com/open-spaced-repetition/go-fsrs` (v2) or `.../go-fsrs/v3` depending on the installed version. After `go get`, check `go.mod` and use the exact path shown there in the import.
