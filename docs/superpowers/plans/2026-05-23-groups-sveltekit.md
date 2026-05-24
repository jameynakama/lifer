# Groups + SvelteKit Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate the frontend to SvelteKit SPA mode, build backend group CRUD + species search API, and wire the dashboard to real data so users can create groups, add species, and run quiz sessions.

**Architecture:** SvelteKit in SPA mode (`adapter-static` with `fallback: 'index.html'`), served by the existing Go server. Eight new API endpoints behind `RequireAuth`. The `view` and `session` stores are retired -- routing is `goto()` + URL params. Existing components (`QuizCard`, `RevealCard`, `ImageQuizCard`, `StatsBar`, `GroupList`) are unchanged.

**Tech Stack:** Go + chi + sqlc, Svelte 5 + SvelteKit 2, `@sveltejs/adapter-static`, vitest + `@testing-library/svelte`

---

## File Map

**New backend files:**
- `backend/internal/store/queries/groups.sql` -- group + group_species queries
- `backend/internal/store/queries/species.sql` -- species search query
- `backend/internal/api/groups.go` -- 7 group handlers + `groupOwnerCheck` helper
- `backend/internal/api/groups_test.go` -- group handler tests
- `backend/internal/api/species.go` -- `searchSpecies` handler
- `backend/internal/api/species_test.go` -- species search tests

**Modified backend files:**
- `backend/internal/api/quiz_test.go` -- add `withChiParams` helper
- `backend/internal/api/router.go` -- register 8 new routes

**New frontend files:**
- `frontend/svelte.config.js` -- SvelteKit adapter + aliases
- `frontend/vitest.config.ts` -- vitest config (extracted from vite.config.ts)
- `frontend/src/app.html` -- SvelteKit entry template
- `frontend/src/routes/+layout.svelte` -- auth check, app shell
- `frontend/src/routes/+layout.test.ts`
- `frontend/src/routes/+page.svelte` -- Dashboard (real API)
- `frontend/src/routes/+page.test.ts`
- `frontend/src/routes/groups/+page.svelte` -- Groups list (CRUD)
- `frontend/src/routes/groups/+page.test.ts`
- `frontend/src/routes/groups/[id]/+page.svelte` -- Group detail + species search
- `frontend/src/routes/groups/[id]/+page.test.ts`
- `frontend/src/routes/groups/[id]/quiz/+page.svelte` -- Quiz session
- `frontend/src/routes/groups/[id]/quiz/+page.test.ts`
- `frontend/src/routes/explore/+page.svelte` -- Stub

**Modified frontend files:**
- `frontend/package.json` -- add `@sveltejs/kit`, `@sveltejs/adapter-static`; update scripts
- `frontend/vite.config.ts` -- replace `svelte()` with `sveltekit()`, remove `test` block

**Deleted frontend files:**
- `frontend/src/App.svelte`, `frontend/src/App.test.ts`
- `frontend/src/main.ts`
- `frontend/src/views/Dashboard.svelte`, `frontend/src/views/Dashboard.test.ts`
- `frontend/src/views/Quiz.svelte`, `frontend/src/views/Quiz.test.ts`
- `frontend/src/stores/view.ts`, `frontend/src/stores/session.ts`
- `frontend/src/stores/stores.test.ts`

---

### Task 1: SQL queries -- groups and species

**Files:**
- Create: `backend/internal/store/queries/groups.sql`
- Create: `backend/internal/store/queries/species.sql`

- [ ] **Step 1: Create `groups.sql`**

```sql
-- name: ListUserGroups :many
SELECT g.id, g.name, g.description, g.is_preset, g.owner_id, g.created_at,
    COUNT(CASE WHEN c.lane = 'audio' AND c.due <= NOW() THEN 1 END) AS audio_due,
    COUNT(CASE WHEN c.lane = 'image' AND c.due <= NOW() THEN 1 END) AS image_due
FROM groups g
LEFT JOIN group_species gs ON gs.group_id = g.id
LEFT JOIN cards c ON c.species_id = gs.species_id AND c.user_id = $1
WHERE g.owner_id = $1
GROUP BY g.id
ORDER BY g.name;

-- name: GetGroup :one
SELECT id, name, description, is_preset, owner_id, created_at
FROM groups
WHERE id = $1;

-- name: CreateGroup :one
INSERT INTO groups (name, owner_id)
VALUES ($1, $2)
RETURNING id, name, description, is_preset, owner_id, created_at;

-- name: UpdateGroupName :one
UPDATE groups SET name = $2 WHERE id = $1
RETURNING id, name, description, is_preset, owner_id, created_at;

-- name: DeleteGroup :exec
DELETE FROM groups WHERE id = $1;

-- name: ListGroupSpecies :many
SELECT s.id, s.common_name, s.scientific_name, s.ebird_code
FROM species s
JOIN group_species gs ON gs.species_id = s.id
WHERE gs.group_id = $1
ORDER BY s.common_name;

-- name: AddSpeciesToGroup :exec
INSERT INTO group_species (group_id, species_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: RemoveSpeciesFromGroup :exec
DELETE FROM group_species
WHERE group_id = $1 AND species_id = $2;
```

- [ ] **Step 2: Create `species.sql`**

```sql
-- name: SearchSpecies :many
SELECT id, common_name, scientific_name, ebird_code
FROM species
WHERE common_name ILIKE '%' || $1 || '%'
   OR scientific_name ILIKE '%' || $1 || '%'
ORDER BY common_name
LIMIT 20;
```

- [ ] **Step 3: Commit**

```bash
jj describe -m "feat: add groups and species SQL queries"
jj new
```

---

### Task 2: Generate sqlc types and verify

**Files:**
- Modified (auto-generated): `backend/internal/store/querier.go`, `backend/internal/store/db.go`, `backend/internal/store/groups.sql.go`, `backend/internal/store/species.sql.go`

- [ ] **Step 1: Run `just generate`**

```bash
just generate
```

Expected: no errors. Verify generated files exist:
```bash
ls backend/internal/store/*.go
```

Expected output includes: `groups.sql.go`, `species.sql.go`

- [ ] **Step 2: Verify the build still compiles**

```bash
just test
```

Expected: all existing tests pass.

- [ ] **Step 3: Confirm key generated types**

Open `backend/internal/store/groups.sql.go` and confirm:
- `ListUserGroupsRow` struct has `AudioDue int64` and `ImageDue int64`
- `Group` struct has `OwnerID pgtype.Int8` (nullable -- matches schema: `owner_id BIGINT REFERENCES users(id)` without NOT NULL)
- `CreateGroupParams` has `Name string` and `OwnerID pgtype.Int8`

Open `backend/internal/store/querier.go` and confirm the new methods appear in the interface.

- [ ] **Step 4: Commit**

```bash
jj describe -m "feat: generate sqlc types for groups and species"
jj new
```

---

### Task 3: Group CRUD handlers -- list and create

**Files:**
- Create: `backend/internal/api/groups.go`
- Create: `backend/internal/api/groups_test.go`

- [ ] **Step 1: Write failing tests for `listGroups` and `createGroup`**

Create `backend/internal/api/groups_test.go`:

```go
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jameynakama/lifer/internal/store"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// groupStubQuerier stubs only group-related methods.
type groupStubQuerier struct {
	store.Querier
	listUserGroups       func(ctx context.Context, userID int64) ([]store.ListUserGroupsRow, error)
	createGroup          func(ctx context.Context, arg store.CreateGroupParams) (store.Group, error)
	getGroup             func(ctx context.Context, id int64) (store.Group, error)
	updateGroupName      func(ctx context.Context, arg store.UpdateGroupNameParams) (store.Group, error)
	deleteGroup          func(ctx context.Context, id int64) error
	listGroupSpecies     func(ctx context.Context, groupID int64) ([]store.ListGroupSpeciesRow, error)
	addSpeciesToGroup    func(ctx context.Context, arg store.AddSpeciesToGroupParams) error
	removeSpeciesFromGroup func(ctx context.Context, arg store.RemoveSpeciesFromGroupParams) error
	upsertCard           func(ctx context.Context, arg store.UpsertCardParams) error
}

func (s *groupStubQuerier) ListUserGroups(ctx context.Context, userID int64) ([]store.ListUserGroupsRow, error) {
	return s.listUserGroups(ctx, userID)
}
func (s *groupStubQuerier) CreateGroup(ctx context.Context, arg store.CreateGroupParams) (store.Group, error) {
	return s.createGroup(ctx, arg)
}
func (s *groupStubQuerier) GetGroup(ctx context.Context, id int64) (store.Group, error) {
	return s.getGroup(ctx, id)
}
func (s *groupStubQuerier) UpdateGroupName(ctx context.Context, arg store.UpdateGroupNameParams) (store.Group, error) {
	return s.updateGroupName(ctx, arg)
}
func (s *groupStubQuerier) DeleteGroup(ctx context.Context, id int64) error {
	return s.deleteGroup(ctx, id)
}
func (s *groupStubQuerier) ListGroupSpecies(ctx context.Context, groupID int64) ([]store.ListGroupSpeciesRow, error) {
	return s.listGroupSpecies(ctx, groupID)
}
func (s *groupStubQuerier) AddSpeciesToGroup(ctx context.Context, arg store.AddSpeciesToGroupParams) error {
	return s.addSpeciesToGroup(ctx, arg)
}
func (s *groupStubQuerier) RemoveSpeciesFromGroup(ctx context.Context, arg store.RemoveSpeciesFromGroupParams) error {
	return s.removeSpeciesFromGroup(ctx, arg)
}
func (s *groupStubQuerier) UpsertCard(ctx context.Context, arg store.UpsertCardParams) error {
	return s.upsertCard(ctx, arg)
}

func ownerID(id int64) pgtype.Int8 {
	return pgtype.Int8{Int64: id, Valid: true}
}

func TestListGroups_ReturnsList(t *testing.T) {
	q := &groupStubQuerier{
		listUserGroups: func(_ context.Context, userID int64) ([]store.ListUserGroupsRow, error) {
			assert.Equal(t, int64(1), userID)
			return []store.ListUserGroupsRow{
				{ID: 1, Name: "My Warblers", OwnerID: ownerID(1), AudioDue: 3, ImageDue: 1},
			}, nil
		},
	}
	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/groups", nil)
	r = injectUserID(r, 1)
	w := httptest.NewRecorder()

	h.listGroups(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var body []store.ListUserGroupsRow
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Len(t, body, 1)
	assert.Equal(t, "My Warblers", body[0].Name)
	assert.Equal(t, int64(3), body[0].AudioDue)
}

func TestCreateGroup_ReturnsGroup(t *testing.T) {
	q := &groupStubQuerier{
		createGroup: func(_ context.Context, arg store.CreateGroupParams) (store.Group, error) {
			assert.Equal(t, "Pacific Northwest", arg.Name)
			assert.Equal(t, int64(1), arg.OwnerID.Int64)
			return store.Group{ID: 42, Name: "Pacific Northwest", OwnerID: ownerID(1)}, nil
		},
	}
	h := makeHandler(q)
	body := `{"name":"Pacific Northwest"}`
	r := httptest.NewRequest(http.MethodPost, "/api/v1/groups", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = injectUserID(r, 1)
	w := httptest.NewRecorder()

	h.createGroup(w, r)

	assert.Equal(t, http.StatusCreated, w.Code)
	var got store.Group
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	assert.Equal(t, int64(42), got.ID)
	assert.Equal(t, "Pacific Northwest", got.Name)
}

func TestCreateGroup_EmptyName_Returns400(t *testing.T) {
	h := makeHandler(&groupStubQuerier{})
	body := `{"name":""}`
	r := httptest.NewRequest(http.MethodPost, "/api/v1/groups", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = injectUserID(r, 1)
	w := httptest.NewRecorder()

	h.createGroup(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
cd backend && go test ./internal/api/... 2>&1 | head -20
```

Expected: compile error (listGroups, createGroup not defined yet).

- [ ] **Step 3: Implement `listGroups` and `createGroup` in `groups.go`**

Create `backend/internal/api/groups.go`:

```go
package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jameynakama/lifer/internal/auth"
	"github.com/jameynakama/lifer/internal/store"
)

type createGroupRequest struct {
	Name string `json:"name"`
}

type updateGroupRequest struct {
	Name string `json:"name"`
}

type addSpeciesRequest struct {
	SpeciesID int64 `json:"species_id"`
}

// groupOwnerCheck fetches the group, writes 404/403 and returns false if the
// requesting user does not own it.
func (h *Handler) groupOwnerCheck(w http.ResponseWriter, r *http.Request, groupID, userID int64) bool {
	group, err := h.queries.GetGroup(r.Context(), groupID)
	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, "not found", http.StatusNotFound)
		return false
	}
	if err != nil {
		log.Printf("GetGroup error: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return false
	}
	if !group.OwnerID.Valid || group.OwnerID.Int64 != userID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return false
	}
	return true
}

func (h *Handler) listGroups(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromCtx(r.Context())
	groups, err := h.queries.ListUserGroups(r.Context(), userID)
	if err != nil {
		log.Printf("ListUserGroups error: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	if groups == nil {
		groups = []store.ListUserGroupsRow{}
	}
	writeJSON(w, http.StatusOK, groups)
}

func (h *Handler) createGroup(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromCtx(r.Context())

	var req createGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	group, err := h.queries.CreateGroup(r.Context(), store.CreateGroupParams{
		Name:    req.Name,
		OwnerID: pgtype.Int8{Int64: userID, Valid: true},
	})
	if err != nil {
		log.Printf("CreateGroup error: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, group)
}
```

- [ ] **Step 4: Run tests**

```bash
cd backend && go test ./internal/api/... -run 'TestListGroups|TestCreateGroup'
```

Expected: PASS (3 tests).

- [ ] **Step 5: Commit**

```bash
jj describe -m "feat: add listGroups and createGroup handlers"
jj new
```

---

### Task 4: Group CRUD handlers -- update and delete

**Files:**
- Modify: `backend/internal/api/groups.go`
- Modify: `backend/internal/api/groups_test.go`

- [ ] **Step 1: Write failing tests for `updateGroup` and `deleteGroup`**

Add to `backend/internal/api/groups_test.go`:

```go
func TestUpdateGroup_RenamesGroup(t *testing.T) {
	q := &groupStubQuerier{
		getGroup: func(_ context.Context, id int64) (store.Group, error) {
			return store.Group{ID: id, Name: "Old Name", OwnerID: ownerID(1)}, nil
		},
		updateGroupName: func(_ context.Context, arg store.UpdateGroupNameParams) (store.Group, error) {
			assert.Equal(t, int64(42), arg.ID)
			assert.Equal(t, "New Name", arg.Name)
			return store.Group{ID: 42, Name: "New Name", OwnerID: ownerID(1)}, nil
		},
	}
	h := makeHandler(q)
	body := `{"name":"New Name"}`
	r := httptest.NewRequest(http.MethodPatch, "/api/v1/groups/42", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.updateGroup(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var got store.Group
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	assert.Equal(t, "New Name", got.Name)
}

func TestUpdateGroup_WrongOwner_Returns403(t *testing.T) {
	q := &groupStubQuerier{
		getGroup: func(_ context.Context, id int64) (store.Group, error) {
			return store.Group{ID: id, OwnerID: ownerID(999)}, nil
		},
	}
	h := makeHandler(q)
	body := `{"name":"New Name"}`
	r := httptest.NewRequest(http.MethodPatch, "/api/v1/groups/42", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.updateGroup(w, r)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestDeleteGroup_DeletesGroup(t *testing.T) {
	deleted := false
	q := &groupStubQuerier{
		getGroup: func(_ context.Context, id int64) (store.Group, error) {
			return store.Group{ID: id, OwnerID: ownerID(1)}, nil
		},
		deleteGroup: func(_ context.Context, id int64) error {
			assert.Equal(t, int64(42), id)
			deleted = true
			return nil
		},
	}
	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodDelete, "/api/v1/groups/42", nil)
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.deleteGroup(w, r)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.True(t, deleted)
}

func TestDeleteGroup_NotFound_Returns404(t *testing.T) {
	q := &groupStubQuerier{
		getGroup: func(_ context.Context, id int64) (store.Group, error) {
			return store.Group{}, pgx.ErrNoRows
		},
	}
	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodDelete, "/api/v1/groups/99", nil)
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "99")
	w := httptest.NewRecorder()

	h.deleteGroup(w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
```

Also add missing import to the test file:
```go
"github.com/jackc/pgx/v5"
```
(Check if it's already imported from Task 3; add if not.)

- [ ] **Step 2: Run tests to confirm they fail**

```bash
cd backend && go test ./internal/api/... -run 'TestUpdateGroup|TestDeleteGroup' 2>&1 | head -20
```

Expected: compile error (updateGroup, deleteGroup not defined yet).

- [ ] **Step 3: Add `updateGroup` and `deleteGroup` to `groups.go`**

Append to `backend/internal/api/groups.go`:

```go
func (h *Handler) updateGroup(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromCtx(r.Context())

	groupID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid group id", http.StatusBadRequest)
		return
	}

	if !h.groupOwnerCheck(w, r, groupID, userID) {
		return
	}

	var req updateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	group, err := h.queries.UpdateGroupName(r.Context(), store.UpdateGroupNameParams{
		ID:   groupID,
		Name: req.Name,
	})
	if err != nil {
		log.Printf("UpdateGroupName error: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, group)
}

func (h *Handler) deleteGroup(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromCtx(r.Context())

	groupID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid group id", http.StatusBadRequest)
		return
	}

	if !h.groupOwnerCheck(w, r, groupID, userID) {
		return
	}

	if err := h.queries.DeleteGroup(r.Context(), groupID); err != nil {
		log.Printf("DeleteGroup error: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
```

- [ ] **Step 4: Run tests**

```bash
cd backend && go test ./internal/api/... -run 'TestUpdateGroup|TestDeleteGroup'
```

Expected: PASS (4 tests).

- [ ] **Step 5: Commit**

```bash
jj describe -m "feat: add updateGroup and deleteGroup handlers"
jj new
```

---

### Task 5: Group species handlers -- list, add, remove

**Files:**
- Modify: `backend/internal/api/groups.go`
- Modify: `backend/internal/api/groups_test.go`
- Modify: `backend/internal/api/quiz_test.go` -- add `withChiParams` helper

- [ ] **Step 1: Add `withChiParams` helper to `quiz_test.go`**

Open `backend/internal/api/quiz_test.go` and add after the existing `withChiParam` function:

```go
// withChiParams sets multiple chi URL params on the request in a single context.
func withChiParams(r *http.Request, params map[string]string) *http.Request {
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}
```

- [ ] **Step 2: Run existing tests to confirm helper compiles**

```bash
cd backend && go test ./internal/api/...
```

Expected: all existing tests pass.

- [ ] **Step 3: Write failing tests for group species handlers**

Add to `backend/internal/api/groups_test.go`:

```go
func TestListGroupSpecies_ReturnsList(t *testing.T) {
	q := &groupStubQuerier{
		getGroup: func(_ context.Context, id int64) (store.Group, error) {
			return store.Group{ID: id, OwnerID: ownerID(1)}, nil
		},
		listGroupSpecies: func(_ context.Context, groupID int64) ([]store.ListGroupSpeciesRow, error) {
			assert.Equal(t, int64(42), groupID)
			return []store.ListGroupSpeciesRow{
				{ID: 7, CommonName: "Song Sparrow", ScientificName: "Melospiza melodia", EbirdCode: "sonspa"},
			}, nil
		},
	}
	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/groups/42/species", nil)
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.listGroupSpecies(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var body []store.ListGroupSpeciesRow
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Len(t, body, 1)
	assert.Equal(t, "Song Sparrow", body[0].CommonName)
}

func TestAddSpeciesToGroup_InsertsAndUpsertsBothCards(t *testing.T) {
	upsertedLanes := []string{}
	q := &groupStubQuerier{
		getGroup: func(_ context.Context, id int64) (store.Group, error) {
			return store.Group{ID: id, OwnerID: ownerID(1)}, nil
		},
		addSpeciesToGroup: func(_ context.Context, arg store.AddSpeciesToGroupParams) error {
			assert.Equal(t, int64(42), arg.GroupID)
			assert.Equal(t, int64(7), arg.SpeciesID)
			return nil
		},
		upsertCard: func(_ context.Context, arg store.UpsertCardParams) error {
			upsertedLanes = append(upsertedLanes, arg.Lane)
			return nil
		},
	}
	h := makeHandler(q)
	body := `{"species_id":7}`
	r := httptest.NewRequest(http.MethodPost, "/api/v1/groups/42/species", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = injectUserID(r, 1)
	r = withChiParam(r, "id", "42")
	w := httptest.NewRecorder()

	h.addSpeciesToGroup(w, r)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.ElementsMatch(t, []string{"audio", "image"}, upsertedLanes)
}

func TestRemoveSpeciesFromGroup_RemovesEntry(t *testing.T) {
	removed := false
	q := &groupStubQuerier{
		getGroup: func(_ context.Context, id int64) (store.Group, error) {
			return store.Group{ID: id, OwnerID: ownerID(1)}, nil
		},
		removeSpeciesFromGroup: func(_ context.Context, arg store.RemoveSpeciesFromGroupParams) error {
			assert.Equal(t, int64(42), arg.GroupID)
			assert.Equal(t, int64(7), arg.SpeciesID)
			removed = true
			return nil
		},
	}
	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodDelete, "/api/v1/groups/42/species/7", nil)
	r = injectUserID(r, 1)
	r = withChiParams(r, map[string]string{"id": "42", "species_id": "7"})
	w := httptest.NewRecorder()

	h.removeSpeciesFromGroup(w, r)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.True(t, removed)
}
```

- [ ] **Step 4: Run tests to confirm they fail**

```bash
cd backend && go test ./internal/api/... -run 'TestListGroupSpecies|TestAddSpecies|TestRemoveSpecies' 2>&1 | head -20
```

Expected: compile error.

- [ ] **Step 5: Implement group species handlers in `groups.go`**

Append to `backend/internal/api/groups.go`:

```go
func (h *Handler) listGroupSpecies(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromCtx(r.Context())

	groupID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid group id", http.StatusBadRequest)
		return
	}

	if !h.groupOwnerCheck(w, r, groupID, userID) {
		return
	}

	species, err := h.queries.ListGroupSpecies(r.Context(), groupID)
	if err != nil {
		log.Printf("ListGroupSpecies error: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	if species == nil {
		species = []store.ListGroupSpeciesRow{}
	}
	writeJSON(w, http.StatusOK, species)
}

func (h *Handler) addSpeciesToGroup(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromCtx(r.Context())

	groupID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid group id", http.StatusBadRequest)
		return
	}

	if !h.groupOwnerCheck(w, r, groupID, userID) {
		return
	}

	var req addSpeciesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.queries.AddSpeciesToGroup(r.Context(), store.AddSpeciesToGroupParams{
		GroupID:   groupID,
		SpeciesID: req.SpeciesID,
	}); err != nil {
		log.Printf("AddSpeciesToGroup error: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	for _, lane := range []string{"audio", "image"} {
		if err := h.queries.UpsertCard(r.Context(), store.UpsertCardParams{
			UserID:    userID,
			SpeciesID: req.SpeciesID,
			Lane:      lane,
		}); err != nil {
			log.Printf("UpsertCard error: %v", err)
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) removeSpeciesFromGroup(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromCtx(r.Context())

	groupID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid group id", http.StatusBadRequest)
		return
	}

	speciesID, err := strconv.ParseInt(chi.URLParam(r, "species_id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid species id", http.StatusBadRequest)
		return
	}

	if !h.groupOwnerCheck(w, r, groupID, userID) {
		return
	}

	_ = userID // cards are not deleted (become dormant)
	if err := h.queries.RemoveSpeciesFromGroup(r.Context(), store.RemoveSpeciesFromGroupParams{
		GroupID:   groupID,
		SpeciesID: speciesID,
	}); err != nil {
		log.Printf("RemoveSpeciesFromGroup error: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
```

- [ ] **Step 6: Run tests**

```bash
cd backend && go test ./internal/api/...
```

Expected: all tests pass.

- [ ] **Step 7: Commit**

```bash
jj describe -m "feat: add group species handlers (list, add, remove)"
jj new
```

---

### Task 6: Species search handler

**Files:**
- Create: `backend/internal/api/species.go`
- Create: `backend/internal/api/species_test.go`

- [ ] **Step 1: Write failing test for `searchSpecies`**

Create `backend/internal/api/species_test.go`:

```go
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jameynakama/lifer/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type speciesStubQuerier struct {
	store.Querier
	searchSpecies func(ctx context.Context, query string) ([]store.SearchSpeciesRow, error)
}

func (s *speciesStubQuerier) SearchSpecies(ctx context.Context, query string) ([]store.SearchSpeciesRow, error) {
	return s.searchSpecies(ctx, query)
}

func TestSearchSpecies_ReturnsMatches(t *testing.T) {
	q := &speciesStubQuerier{
		searchSpecies: func(_ context.Context, query string) ([]store.SearchSpeciesRow, error) {
			assert.Equal(t, "sparrow", query)
			return []store.SearchSpeciesRow{
				{ID: 1, CommonName: "Song Sparrow", ScientificName: "Melospiza melodia", EbirdCode: "sonspa"},
				{ID: 2, CommonName: "Fox Sparrow", ScientificName: "Passerella iliaca", EbirdCode: "foxspa"},
			}, nil
		},
	}
	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/species?q=sparrow", nil)
	r = injectUserID(r, 1)
	w := httptest.NewRecorder()

	h.searchSpecies(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var body []store.SearchSpeciesRow
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Len(t, body, 2)
	assert.Equal(t, "Song Sparrow", body[0].CommonName)
}

func TestSearchSpecies_EmptyQuery_ReturnsEmpty(t *testing.T) {
	q := &speciesStubQuerier{
		searchSpecies: func(_ context.Context, query string) ([]store.SearchSpeciesRow, error) {
			return []store.SearchSpeciesRow{}, nil
		},
	}
	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/species?q=", nil)
	r = injectUserID(r, 1)
	w := httptest.NewRecorder()

	h.searchSpecies(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
}
```

- [ ] **Step 2: Run test to confirm it fails**

```bash
cd backend && go test ./internal/api/... -run TestSearchSpecies 2>&1 | head -20
```

Expected: compile error.

- [ ] **Step 3: Implement `searchSpecies`**

Create `backend/internal/api/species.go`:

```go
package api

import (
	"log"
	"net/http"

	"github.com/jameynakama/lifer/internal/store"
)

func (h *Handler) searchSpecies(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")

	results, err := h.queries.SearchSpecies(r.Context(), q)
	if err != nil {
		log.Printf("SearchSpecies error: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	if results == nil {
		results = []store.SearchSpeciesRow{}
	}
	writeJSON(w, http.StatusOK, results)
}
```

- [ ] **Step 4: Run tests**

```bash
cd backend && go test ./internal/api/...
```

Expected: all tests pass.

- [ ] **Step 5: Commit**

```bash
jj describe -m "feat: add species search handler"
jj new
```

---

### Task 7: Register all 8 new routes

**Files:**
- Modify: `backend/internal/api/router.go`

- [ ] **Step 1: Add routes to the auth-required group in `router.go`**

In `backend/internal/api/router.go`, replace the auth group block:

Old:
```go
r.Group(func(r chi.Router) {
    r.Use(auth.RequireAuth(cfg.JWTSecret))
    r.Get("/me", h.getMe)
    r.Get("/groups/{id}/next", h.getNextCard)
    r.Post("/groups/{id}/rate", h.rateCard)
    r.Put("/species/{id}/preferences", h.updatePreferences)
})
```

New:
```go
r.Group(func(r chi.Router) {
    r.Use(auth.RequireAuth(cfg.JWTSecret))
    r.Get("/me", h.getMe)

    r.Get("/groups", h.listGroups)
    r.Post("/groups", h.createGroup)
    r.Patch("/groups/{id}", h.updateGroup)
    r.Delete("/groups/{id}", h.deleteGroup)

    r.Get("/groups/{id}/species", h.listGroupSpecies)
    r.Post("/groups/{id}/species", h.addSpeciesToGroup)
    r.Delete("/groups/{id}/species/{species_id}", h.removeSpeciesFromGroup)

    r.Get("/groups/{id}/next", h.getNextCard)
    r.Post("/groups/{id}/rate", h.rateCard)

    r.Get("/species", h.searchSpecies)
    r.Put("/species/{id}/preferences", h.updatePreferences)
})
```

- [ ] **Step 2: Build and test**

```bash
just test
```

Expected: all tests pass, no compile errors.

- [ ] **Step 3: Commit**

```bash
jj describe -m "feat: register group and species search routes"
jj new
```

---

### Task 8: SvelteKit packages and config

**Files:**
- Modify: `frontend/package.json`
- Modify: `frontend/vite.config.ts`
- Create: `frontend/svelte.config.js`
- Create: `frontend/vitest.config.ts`
- Create: `frontend/src/app.html`
- Delete: `frontend/index.html`

- [ ] **Step 1: Install SvelteKit packages**

```bash
cd frontend && npm install -D @sveltejs/kit @sveltejs/adapter-static
```

Expected: packages installed, `package.json` and `package-lock.json` updated.

- [ ] **Step 2: Update `package.json` scripts**

In `frontend/package.json`, replace the `scripts` block:

Old:
```json
"scripts": {
  "dev": "vite",
  "build": "vite build",
  "preview": "vite preview",
  "check": "svelte-check --tsconfig ./tsconfig.app.json && tsc -p tsconfig.node.json",
  "check:watch": "svelte-check --tsconfig ./tsconfig.app.json --watch",
  "test": "vitest run",
  "test:watch": "vitest"
}
```

New:
```json
"scripts": {
  "dev": "vite dev",
  "build": "vite build",
  "preview": "vite preview",
  "check": "svelte-kit sync && svelte-check --tsconfig ./tsconfig.json",
  "test": "vitest run --config vitest.config.ts",
  "test:watch": "vitest --config vitest.config.ts"
}
```

- [ ] **Step 3: Create `svelte.config.js`**

```js
import adapter from '@sveltejs/adapter-static';

export default {
  kit: {
    adapter: adapter({ fallback: 'index.html' }),
    alias: {
      $stores: './src/stores',
      $components: './src/components',
      $lib: './src/lib',
    },
  },
};
```

- [ ] **Step 4: Update `vite.config.ts`**

Replace the entire file:

```typescript
import { defineConfig } from 'vite'
import { sveltekit } from '@sveltejs/kit/vite'

export default defineConfig({
  plugins: [sveltekit()],
  server: {
    proxy: {
      '/api': 'http://127.0.0.1:8080',
    },
  },
})
```

- [ ] **Step 5: Create `vitest.config.ts`**

```typescript
import { defineConfig } from 'vitest/config'
import { svelte } from '@sveltejs/vite-plugin-svelte'

export default defineConfig({
  plugins: [svelte({ hot: !process.env.VITEST })],
  resolve: {
    conditions: ['browser'],
    alias: {
      $stores: new URL('./src/stores', import.meta.url).pathname,
      $components: new URL('./src/components', import.meta.url).pathname,
      $lib: new URL('./src/lib', import.meta.url).pathname,
      '$app/navigation': new URL('./src/__mocks__/app-navigation.ts', import.meta.url).pathname,
      '$app/state': new URL('./src/__mocks__/app-state.ts', import.meta.url).pathname,
    },
  },
  test: {
    environment: 'jsdom',
    setupFiles: ['src/test-setup.ts'],
    globals: true,
  },
})
```

- [ ] **Step 6: Create SvelteKit mock modules for tests**

Create `frontend/src/__mocks__/app-navigation.ts`:

```typescript
import { vi } from 'vitest'

export const goto = vi.fn()
```

Create `frontend/src/__mocks__/app-state.ts`:

```typescript
export const page = {
  params: {} as Record<string, string>,
  url: new URL('http://localhost/'),
}
```

- [ ] **Step 7: Create `src/app.html`**

```html
<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    %sveltekit.head%
  </head>
  <body data-sveltekit-preload-data="hover">
    <div style="display: contents">%sveltekit.body%</div>
  </body>
</html>
```

- [ ] **Step 8: Delete `frontend/index.html`**

```bash
rm frontend/index.html
```

- [ ] **Step 9: Run `svelte-kit sync` to generate tsconfig**

```bash
cd frontend && npx svelte-kit sync
```

Expected: generates `frontend/.svelte-kit/` directory and `tsconfig.json`.

- [ ] **Step 10: Run existing frontend tests**

```bash
cd frontend && npm test
```

Expected: all existing tests pass (using the new vitest config).

- [ ] **Step 11: Commit**

```bash
jj describe -m "feat: configure SvelteKit with adapter-static and vitest split"
jj new
```

---

### Task 9: Root `+layout.svelte` -- auth check and app shell

**Files:**
- Create: `frontend/src/routes/+layout.svelte`
- Create: `frontend/src/routes/+layout.test.ts`
- Delete: `frontend/src/App.svelte`
- Delete: `frontend/src/App.test.ts`
- Delete: `frontend/src/main.ts`
- Modify: `frontend/src/stores/stores.test.ts` -- remove view/session tests

- [ ] **Step 1: Write failing test for `+layout.svelte`**

Create `frontend/src/routes/+layout.test.ts`:

```typescript
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen } from '@testing-library/svelte'
import { get } from 'svelte/store'
import { auth } from '$stores/auth'
import Layout from './+layout.svelte'

const mockMatchMedia = (prefersDark = true) => {
  vi.stubGlobal('matchMedia', vi.fn().mockImplementation((query: string) => ({
    matches: query === '(prefers-color-scheme: dark)' ? prefersDark : !prefersDark,
    media: query,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
  })))
}

beforeEach(() => {
  auth.set(null)
  document.documentElement.removeAttribute('data-theme')
  mockMatchMedia()
})

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe('Layout', () => {
  it('shows Login when /api/v1/me returns 401', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false }))
    render(Layout)
    await vi.waitFor(() => {
      expect(screen.getByRole('link', { name: /sign in with google/i })).toBeInTheDocument()
    })
  })

  it('shows app shell and sets auth when /api/v1/me returns 200', async () => {
    const user = { id: 1, email: 'test@example.com', name: 'Test User' }
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve(user) }))
    render(Layout)
    await vi.waitFor(() => {
      expect(screen.getByText('Lifer')).toBeInTheDocument()
    })
    expect(get(auth)).toEqual(user)
  })

  it('shows theme toggle when authenticated', async () => {
    const user = { id: 1, email: 'test@example.com', name: 'Test User' }
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve(user) }))
    render(Layout)
    await vi.waitFor(() => {
      expect(screen.getByRole('button', { name: /switch to .* mode/i })).toBeInTheDocument()
    })
  })
})
```

- [ ] **Step 2: Run test to confirm it fails**

```bash
cd frontend && npm test -- --reporter=verbose 2>&1 | grep -E 'FAIL|Error' | head -20
```

Expected: fail (Layout file doesn't exist yet).

- [ ] **Step 3: Create `src/routes/+layout.svelte`**

```svelte
<script lang="ts">
  import '../app.css'
  import { auth } from '$stores/auth'
  import { getCurrentTheme, toggleTheme } from '$lib/theme'
  import Login from '../views/Login.svelte'

  let { children } = $props()
  let checking = $state(true)
  let theme = $state(getCurrentTheme())

  $effect(() => {
    fetch('/api/v1/me')
      .then(async (res) => {
        if (res.ok) {
          $auth = await res.json()
        }
      })
      .catch(() => {})
      .finally(() => { checking = false })
  })

  function handleToggle() {
    toggleTheme()
    theme = getCurrentTheme()
  }
</script>

{#if checking}
  <div class="loading">
    <span class="spinner"></span>
  </div>
{:else if !$auth}
  <Login />
{:else}
  <header>
    <a href="/" class="wordmark">Lifer</a>
    <nav>
      <a href="/groups">Groups</a>
      <a href="/explore">Explore</a>
    </nav>
    <button
      onclick={handleToggle}
      aria-label={theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'}
    >
      {theme === 'dark' ? '☀️' : '🌙'}
    </button>
  </header>
  <main>
    {@render children()}
  </main>
{/if}

<style>
  .loading {
    display: flex;
    justify-content: center;
    align-items: center;
    min-height: 100vh;
  }
  .spinner {
    width: 32px;
    height: 32px;
    border: 3px solid var(--text-muted);
    border-top-color: var(--text-secondary);
    border-radius: 50%;
    animation: spin 0.7s linear infinite;
  }
  @keyframes spin {
    to { transform: rotate(360deg); }
  }
  header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 1rem 0 1.25rem;
  }
  .wordmark {
    font-size: 1.125rem;
    font-weight: 700;
    color: var(--text);
    letter-spacing: -0.02em;
    text-decoration: none;
  }
  nav {
    display: flex;
    gap: 1rem;
  }
  nav a {
    color: var(--text-secondary);
    text-decoration: none;
    font-size: 0.875rem;
    font-weight: 500;
  }
  nav a:hover {
    color: var(--text);
  }
  header button {
    background: var(--surface);
    border: 1px solid var(--border);
    color: var(--text-secondary);
    border-radius: 6px;
    padding: 0.25rem 0.5rem;
    font-size: 1rem;
    cursor: pointer;
    line-height: 1;
    box-shadow: var(--shadow);
  }
  main {
    padding-bottom: 2rem;
  }
</style>
```

- [ ] **Step 4: Run layout tests**

```bash
cd frontend && npm test -- --reporter=verbose 2>&1 | grep -E 'Layout|PASS|FAIL'
```

Expected: Layout tests pass.

- [ ] **Step 5: Update `stores.test.ts` to remove deleted stores**

Replace `frontend/src/stores/stores.test.ts`:

```typescript
import { describe, it, expect } from 'vitest'
import { get } from 'svelte/store'
import { auth } from './auth'

describe('auth store', () => {
  it('starts as null', () => {
    expect(get(auth)).toBe(null)
  })
})
```

- [ ] **Step 6: Delete old app files**

```bash
rm frontend/src/App.svelte frontend/src/App.test.ts frontend/src/main.ts
```

- [ ] **Step 7: Delete retired stores**

```bash
rm frontend/src/stores/view.ts frontend/src/stores/session.ts
```

- [ ] **Step 8: Run all frontend tests**

```bash
cd frontend && npm test
```

Expected: all tests pass (Dashboard.test.ts and Quiz.test.ts may fail now -- that's expected; they'll be replaced in later tasks).

- [ ] **Step 9: Commit**

```bash
jj describe -m "feat: add SvelteKit layout with auth check and app shell"
jj new
```

---

### Task 10: Dashboard `+page.svelte` with real API data

**Files:**
- Create: `frontend/src/routes/+page.svelte`
- Create: `frontend/src/routes/+page.test.ts`
- Delete: `frontend/src/views/Dashboard.svelte`
- Delete: `frontend/src/views/Dashboard.test.ts`

- [ ] **Step 1: Write failing test for Dashboard page**

Create `frontend/src/routes/+page.test.ts`:

```typescript
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/svelte'
import { goto } from '$app/navigation'
import Dashboard from './+page.svelte'

const groups = [
  { id: 1, name: 'Pacific Northwest', is_preset: false, audio_due: 8, image_due: 5 },
  { id: 2, name: 'My Warblers', is_preset: false, audio_due: 3, image_due: 0 },
]

beforeEach(() => {
  vi.mocked(goto).mockClear()
})

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe('Dashboard page', () => {
  it('renders group names from API', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(groups),
    }))
    render(Dashboard)
    await vi.waitFor(() => {
      expect(screen.getAllByText(/pacific northwest/i).length).toBeGreaterThan(0)
    })
    expect(screen.getAllByText(/my warblers/i).length).toBeGreaterThan(0)
  })

  it('shows empty state when no groups', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve([]),
    }))
    render(Dashboard)
    await vi.waitFor(() => {
      expect(screen.getByText(/no groups yet/i)).toBeInTheDocument()
    })
  })

  it('navigates to quiz when Audio button clicked', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(groups),
    }))
    render(Dashboard)
    await vi.waitFor(() => screen.getAllByRole('button', { name: /audio/i }))
    await fireEvent.click(screen.getAllByRole('button', { name: /audio/i })[0])
    expect(goto).toHaveBeenCalledWith('/groups/1/quiz?lane=audio')
  })

  it('navigates to quiz when Image button clicked', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(groups),
    }))
    render(Dashboard)
    await vi.waitFor(() => screen.getAllByRole('button', { name: /image/i }))
    await fireEvent.click(screen.getAllByRole('button', { name: /image/i })[0])
    expect(goto).toHaveBeenCalledWith('/groups/1/quiz?lane=image')
  })
})
```

- [ ] **Step 2: Run test to confirm it fails**

```bash
cd frontend && npm test -- src/routes/+page.test.ts 2>&1 | head -30
```

Expected: fail (file doesn't exist yet).

- [ ] **Step 3: Create `src/routes/+page.svelte`**

```svelte
<script lang="ts">
  import { goto } from '$app/navigation'
  import type { Group } from '../types'
  import StatsBar from '$components/StatsBar.svelte'
  import GroupList from '$components/GroupList.svelte'

  let groups: Group[] = $state([])
  let loading = $state(true)

  $effect(() => {
    fetch('/api/v1/groups')
      .then(async (res) => {
        if (res.ok) groups = await res.json()
      })
      .finally(() => { loading = false })
  })

  const totalDue = $derived(groups.reduce((sum, g) => sum + g.audio_due + g.image_due, 0))

  const stats = $derived([
    { label: 'Due today', value: totalDue },
  ])

  function startPractice(group: Group, lane: 'audio' | 'image') {
    goto(`/groups/${group.id}/quiz?lane=${lane}`)
  }
</script>

<div class="dashboard">
  {#if loading}
    <p class="status">Loading...</p>
  {:else if groups.length === 0}
    <p class="empty">No groups yet. <a href="/groups">Create one</a> to get started.</p>
  {:else}
    <StatsBar {stats} />
    <GroupList {groups} onPractice={startPractice} />
  {/if}
</div>

<style>
  .dashboard {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }
  .status, .empty {
    text-align: center;
    color: var(--text-muted);
    padding: 2rem 0;
  }
  .empty a {
    color: var(--accent);
  }
</style>
```

- [ ] **Step 4: Run tests**

```bash
cd frontend && npm test -- src/routes/+page.test.ts
```

Expected: all 4 tests pass.

- [ ] **Step 5: Delete old dashboard files**

```bash
rm frontend/src/views/Dashboard.svelte frontend/src/views/Dashboard.test.ts
```

- [ ] **Step 6: Run all frontend tests**

```bash
cd frontend && npm test
```

Expected: all tests pass.

- [ ] **Step 7: Commit**

```bash
jj describe -m "feat: dashboard page fetches real group data from API"
jj new
```

---

### Task 11: `/groups` page -- CRUD list

**Files:**
- Create: `frontend/src/routes/groups/+page.svelte`
- Create: `frontend/src/routes/groups/+page.test.ts`

- [ ] **Step 1: Write failing tests for groups list page**

Create `frontend/src/routes/groups/+page.test.ts`:

```typescript
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/svelte'
import { goto } from '$app/navigation'
import GroupsPage from './+page.svelte'

const groups = [
  { id: 1, name: 'My Warblers', is_preset: false, audio_due: 2, image_due: 0 },
]

beforeEach(() => {
  vi.mocked(goto).mockClear()
})

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe('Groups page', () => {
  it('renders group list from API', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true, json: () => Promise.resolve(groups),
    }))
    render(GroupsPage)
    await vi.waitFor(() => {
      expect(screen.getAllByText(/my warblers/i).length).toBeGreaterThan(0)
    })
  })

  it('creates a group when form submitted', async () => {
    let postCalled = false
    vi.stubGlobal('fetch', vi.fn().mockImplementation((url: string, opts?: RequestInit) => {
      if (opts?.method === 'POST') {
        postCalled = true
        return Promise.resolve({ ok: true, json: () => Promise.resolve({ id: 2, name: 'New Group', is_preset: false, audio_due: 0, image_due: 0 }) })
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve(groups) })
    }))
    render(GroupsPage)
    await vi.waitFor(() => screen.getByPlaceholderText(/group name/i))
    await fireEvent.input(screen.getByPlaceholderText(/group name/i), { target: { value: 'New Group' } })
    await fireEvent.click(screen.getByRole('button', { name: /create/i }))
    await vi.waitFor(() => { expect(postCalled).toBe(true) })
  })

  it('navigates to group detail when name clicked', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true, json: () => Promise.resolve(groups),
    }))
    render(GroupsPage)
    await vi.waitFor(() => screen.getByRole('link', { name: /my warblers/i }))
    // Link navigates via href, not goto -- just verify the link exists with correct href
    const link = screen.getByRole('link', { name: /my warblers/i })
    expect(link).toHaveAttribute('href', '/groups/1')
  })

  it('deletes a group when delete button clicked', async () => {
    let deleteCalled = false
    vi.stubGlobal('fetch', vi.fn().mockImplementation((url: string, opts?: RequestInit) => {
      if (opts?.method === 'DELETE') {
        deleteCalled = true
        return Promise.resolve({ ok: true })
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve(groups) })
    }))
    render(GroupsPage)
    await vi.waitFor(() => screen.getByRole('button', { name: /delete/i }))
    await fireEvent.click(screen.getByRole('button', { name: /delete/i }))
    await vi.waitFor(() => { expect(deleteCalled).toBe(true) })
  })
})
```

- [ ] **Step 2: Run test to confirm it fails**

```bash
cd frontend && npm test -- src/routes/groups/+page.test.ts 2>&1 | head -30
```

Expected: fail (file doesn't exist).

- [ ] **Step 3: Create `src/routes/groups/+page.svelte`**

```svelte
<script lang="ts">
  import type { Group } from '../../types'

  let groups: Group[] = $state([])
  let loading = $state(true)
  let newName = $state('')
  let creating = $state(false)

  async function loadGroups() {
    const res = await fetch('/api/v1/groups')
    if (res.ok) groups = await res.json()
    loading = false
  }

  async function createGroup() {
    if (!newName.trim()) return
    creating = true
    try {
      const res = await fetch('/api/v1/groups', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: newName.trim() }),
      })
      if (res.ok) {
        const created = await res.json()
        groups = [...groups, { ...created, audio_due: 0, image_due: 0 }]
        newName = ''
      }
    } finally {
      creating = false
    }
  }

  async function deleteGroup(id: number) {
    const res = await fetch(`/api/v1/groups/${id}`, { method: 'DELETE' })
    if (res.ok) {
      groups = groups.filter((g) => g.id !== id)
    }
  }

  loadGroups()
</script>

<div class="groups-page">
  <h1>Groups</h1>

  <form class="create-form" onsubmit={(e) => { e.preventDefault(); createGroup() }}>
    <input
      type="text"
      bind:value={newName}
      placeholder="Group name"
      disabled={creating}
    />
    <button type="submit" disabled={creating || !newName.trim()}>Create</button>
  </form>

  {#if loading}
    <p class="status">Loading...</p>
  {:else if groups.length === 0}
    <p class="empty">No groups yet. Create your first one above.</p>
  {:else}
    <ul class="group-list">
      {#each groups as group (group.id)}
        <li class="group-row">
          <a href="/groups/{group.id}">{group.name}</a>
          <button class="btn-delete" onclick={() => deleteGroup(group.id as unknown as number)}>Delete</button>
        </li>
      {/each}
    </ul>
  {/if}
</div>

<style>
  .groups-page {
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
  }
  h1 {
    font-size: 1.25rem;
    font-weight: 700;
    color: var(--text);
    margin: 0;
  }
  .create-form {
    display: flex;
    gap: 0.5rem;
  }
  .create-form input {
    flex: 1;
    background: var(--surface);
    border: 1px solid var(--border);
    color: var(--text);
    border-radius: 8px;
    padding: 0.5rem 0.75rem;
    font-size: 0.9375rem;
    font-family: inherit;
  }
  .create-form button {
    background: var(--accent);
    color: #fff;
    border: none;
    border-radius: 8px;
    padding: 0.5rem 1rem;
    font-size: 0.875rem;
    font-weight: 600;
    cursor: pointer;
    font-family: inherit;
  }
  .create-form button:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
  .group-list {
    list-style: none;
    padding: 0;
    margin: 0;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }
  .group-row {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: 10px;
    padding: 0.875rem 1rem;
    display: flex;
    justify-content: space-between;
    align-items: center;
    box-shadow: var(--shadow);
  }
  .group-row a {
    color: var(--text);
    font-weight: 600;
    text-decoration: none;
    font-size: 0.9375rem;
  }
  .btn-delete {
    background: transparent;
    border: 1px solid var(--border);
    color: var(--text-muted);
    border-radius: 6px;
    padding: 0.25rem 0.5rem;
    font-size: 0.75rem;
    cursor: pointer;
    font-family: inherit;
  }
  .status, .empty {
    text-align: center;
    color: var(--text-muted);
    padding: 2rem 0;
  }
</style>
```

- [ ] **Step 4: Run tests**

```bash
cd frontend && npm test -- src/routes/groups/+page.test.ts
```

Expected: all 4 tests pass.

- [ ] **Step 5: Commit**

```bash
jj describe -m "feat: groups list page with create and delete"
jj new
```

---

### Task 12: `/groups/[id]` page -- group detail and species search

**Files:**
- Create: `frontend/src/routes/groups/[id]/+page.svelte`
- Create: `frontend/src/routes/groups/[id]/+page.test.ts`

- [ ] **Step 1: Write failing tests for group detail page**

Create `frontend/src/routes/groups/[id]/+page.test.ts`:

```typescript
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/svelte'
import { goto } from '$app/navigation'
import { page } from '$app/state'
import GroupDetailPage from './+page.svelte'

const species = [
  { id: 7, common_name: 'Song Sparrow', scientific_name: 'Melospiza melodia', ebird_code: 'sonspa' },
]

beforeEach(() => {
  page.params = { id: '42' }
  vi.mocked(goto).mockClear()
})

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe('Group detail page', () => {
  it('renders species list for the group', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true, json: () => Promise.resolve(species),
    }))
    render(GroupDetailPage)
    await vi.waitFor(() => {
      expect(screen.getAllByText(/song sparrow/i).length).toBeGreaterThan(0)
    })
  })

  it('navigates to audio quiz on Practice Audio click', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true, json: () => Promise.resolve(species),
    }))
    render(GroupDetailPage)
    await vi.waitFor(() => screen.getByRole('button', { name: /practice audio/i }))
    await fireEvent.click(screen.getByRole('button', { name: /practice audio/i }))
    expect(goto).toHaveBeenCalledWith('/groups/42/quiz?lane=audio')
  })

  it('searches species and shows results', async () => {
    vi.stubGlobal('fetch', vi.fn().mockImplementation((url: string) => {
      if (url.includes('/api/v1/species')) {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve([
            { id: 8, common_name: 'Fox Sparrow', scientific_name: 'Passerella iliaca', ebird_code: 'foxspa' },
          ]),
        })
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve(species) })
    }))
    render(GroupDetailPage)
    await vi.waitFor(() => screen.getByPlaceholderText(/search species/i))
    await fireEvent.input(screen.getByPlaceholderText(/search species/i), {
      target: { value: 'fox' },
    })
    await vi.waitFor(() => {
      expect(screen.getAllByText(/fox sparrow/i).length).toBeGreaterThan(0)
    })
  })

  it('removes species from group on Remove click', async () => {
    let deleteCalled = false
    vi.stubGlobal('fetch', vi.fn().mockImplementation((url: string, opts?: RequestInit) => {
      if (opts?.method === 'DELETE') {
        deleteCalled = true
        return Promise.resolve({ ok: true })
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve(species) })
    }))
    render(GroupDetailPage)
    await vi.waitFor(() => screen.getByRole('button', { name: /remove/i }))
    await fireEvent.click(screen.getByRole('button', { name: /remove/i }))
    await vi.waitFor(() => { expect(deleteCalled).toBe(true) })
  })
})
```

- [ ] **Step 2: Run test to confirm it fails**

```bash
cd frontend && npm test -- 'src/routes/groups/\[id\]/+page.test.ts' 2>&1 | head -30
```

Expected: fail.

- [ ] **Step 3: Create `src/routes/groups/[id]/+page.svelte`**

```svelte
<script lang="ts">
  import { goto } from '$app/navigation'
  import { page } from '$app/state'

  interface Species {
    id: number
    common_name: string
    scientific_name: string
    ebird_code: string
  }

  let groupId = $derived(page.params.id)
  let groupSpecies: Species[] = $state([])
  let searchQuery = $state('')
  let searchResults: Species[] = $state([])
  let loading = $state(true)
  let searchTimer: ReturnType<typeof setTimeout> | null = null

  async function loadSpecies() {
    const res = await fetch(`/api/v1/groups/${groupId}/species`)
    if (res.ok) groupSpecies = await res.json()
    loading = false
  }

  function onSearchInput() {
    if (searchTimer) clearTimeout(searchTimer)
    if (!searchQuery.trim()) { searchResults = []; return }
    searchTimer = setTimeout(async () => {
      const res = await fetch(`/api/v1/species?q=${encodeURIComponent(searchQuery)}`)
      if (res.ok) searchResults = await res.json()
    }, 300)
  }

  async function addSpecies(speciesId: number) {
    const res = await fetch(`/api/v1/groups/${groupId}/species`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ species_id: speciesId }),
    })
    if (res.ok) {
      const added = searchResults.find((s) => s.id === speciesId)
      if (added) groupSpecies = [...groupSpecies, added]
      searchQuery = ''
      searchResults = []
    }
  }

  async function removeSpecies(speciesId: number) {
    const res = await fetch(`/api/v1/groups/${groupId}/species/${speciesId}`, {
      method: 'DELETE',
    })
    if (res.ok) {
      groupSpecies = groupSpecies.filter((s) => s.id !== speciesId)
    }
  }

  $effect(() => {
    if (groupId) loadSpecies()
  })
</script>

<div class="group-detail">
  <div class="actions">
    <button class="btn-practice" onclick={() => goto(`/groups/${groupId}/quiz?lane=audio`)}>
      Practice Audio
    </button>
    <button class="btn-practice" onclick={() => goto(`/groups/${groupId}/quiz?lane=image`)}>
      Practice Image
    </button>
  </div>

  {#if loading}
    <p class="status">Loading...</p>
  {:else if groupSpecies.length === 0}
    <p class="empty">No species yet. Search below to add some.</p>
  {:else}
    <ul class="species-list">
      {#each groupSpecies as s (s.id)}
        <li class="species-row">
          <span>
            <strong>{s.common_name}</strong>
            <em>{s.scientific_name}</em>
          </span>
          <button class="btn-remove" onclick={() => removeSpecies(s.id)}>Remove</button>
        </li>
      {/each}
    </ul>
  {/if}

  <div class="search-section">
    <input
      type="text"
      placeholder="Search species to add..."
      bind:value={searchQuery}
      oninput={onSearchInput}
    />
    {#if searchResults.length > 0}
      <ul class="search-results">
        {#each searchResults as s (s.id)}
          <li class="search-row">
            <span>
              <strong>{s.common_name}</strong>
              <em>{s.scientific_name}</em>
            </span>
            <button class="btn-add" onclick={() => addSpecies(s.id)}>Add</button>
          </li>
        {/each}
      </ul>
    {/if}
  </div>
</div>

<style>
  .group-detail {
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
  }
  .actions {
    display: flex;
    gap: 0.75rem;
  }
  .btn-practice {
    background: var(--accent);
    color: #fff;
    border: none;
    border-radius: 10px;
    padding: 0.625rem 1.25rem;
    font-size: 0.875rem;
    font-weight: 600;
    cursor: pointer;
    font-family: inherit;
  }
  .species-list, .search-results {
    list-style: none;
    padding: 0;
    margin: 0;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }
  .species-row, .search-row {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: 10px;
    padding: 0.75rem 1rem;
    display: flex;
    justify-content: space-between;
    align-items: center;
    box-shadow: var(--shadow);
  }
  .species-row span, .search-row span {
    display: flex;
    flex-direction: column;
    gap: 0.125rem;
  }
  .species-row strong, .search-row strong {
    font-size: 0.9375rem;
    color: var(--text);
  }
  .species-row em, .search-row em {
    font-size: 0.8125rem;
    color: var(--text-muted);
    font-style: italic;
  }
  .btn-remove {
    background: transparent;
    border: 1px solid var(--border);
    color: var(--text-muted);
    border-radius: 6px;
    padding: 0.25rem 0.5rem;
    font-size: 0.75rem;
    cursor: pointer;
    font-family: inherit;
  }
  .btn-add {
    background: var(--surface);
    border: 1px solid var(--accent);
    color: var(--accent);
    border-radius: 6px;
    padding: 0.25rem 0.625rem;
    font-size: 0.75rem;
    font-weight: 600;
    cursor: pointer;
    font-family: inherit;
  }
  .search-section {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }
  .search-section input {
    background: var(--surface);
    border: 1px solid var(--border);
    color: var(--text);
    border-radius: 8px;
    padding: 0.5rem 0.75rem;
    font-size: 0.9375rem;
    font-family: inherit;
    width: 100%;
    box-sizing: border-box;
  }
  .status, .empty {
    text-align: center;
    color: var(--text-muted);
    padding: 2rem 0;
  }
</style>
```

- [ ] **Step 4: Run tests**

```bash
cd frontend && npm test -- 'src/routes/groups/\[id\]/+page.test.ts'
```

Expected: all 4 tests pass.

- [ ] **Step 5: Commit**

```bash
jj describe -m "feat: group detail page with species search and add/remove"
jj new
```

---

### Task 13: `/groups/[id]/quiz` page and cleanup old quiz files

**Files:**
- Create: `frontend/src/routes/groups/[id]/quiz/+page.svelte`
- Create: `frontend/src/routes/groups/[id]/quiz/+page.test.ts`
- Delete: `frontend/src/views/Quiz.svelte`
- Delete: `frontend/src/views/Quiz.test.ts`

- [ ] **Step 1: Write failing tests for quiz page**

Create `frontend/src/routes/groups/[id]/quiz/+page.test.ts`:

```typescript
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/svelte'
import { goto } from '$app/navigation'
import { page } from '$app/state'
import QuizPage from './+page.svelte'

const card = {
  species_id: 99,
  common_name: 'Song Sparrow',
  scientific_name: 'Melospiza melodia',
  media_url: '/recordings/song-sparrow.mp3',
  photo_url: '/photos/song-sparrow.jpg',
  lane: 'audio',
}

beforeEach(() => {
  page.params = { id: '42' }
  page.url = new URL('http://localhost/groups/42/quiz?lane=audio')
  vi.mocked(goto).mockClear()
})

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe('Quiz page', () => {
  it('shows loading initially', () => {
    vi.stubGlobal('fetch', vi.fn().mockReturnValue(new Promise(() => {})))
    render(QuizPage)
    expect(screen.getByText(/loading/i)).toBeInTheDocument()
  })

  it('shows QuizCard when a card is returned', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true, status: 200, json: () => Promise.resolve(card),
    }))
    render(QuizPage)
    await vi.waitFor(() => {
      expect(document.querySelector('audio')).not.toBeNull()
    })
  })

  it('shows All done when 204 is returned', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, status: 204 }))
    render(QuizPage)
    await vi.waitFor(() => {
      expect(screen.getByText(/all done/i)).toBeInTheDocument()
    })
  })

  it('navigates to group detail when All done button clicked', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, status: 204 }))
    render(QuizPage)
    await vi.waitFor(() => screen.getByRole('button', { name: /back to group/i }))
    await fireEvent.click(screen.getByRole('button', { name: /back to group/i }))
    expect(goto).toHaveBeenCalledWith('/groups/42')
  })
})
```

- [ ] **Step 2: Run test to confirm it fails**

```bash
cd frontend && npm test -- 'src/routes/groups/\[id\]/quiz/+page.test.ts' 2>&1 | head -30
```

Expected: fail.

- [ ] **Step 3: Create `src/routes/groups/[id]/quiz/+page.svelte`**

```svelte
<script lang="ts">
  import { goto } from '$app/navigation'
  import { page } from '$app/state'
  import type { BirdCard } from '../../../../types'
  import QuizCard from '$components/QuizCard.svelte'
  import ImageQuizCard from '$components/ImageQuizCard.svelte'
  import RevealCard from '$components/RevealCard.svelte'
  import StatsBar from '$components/StatsBar.svelte'

  let groupId = $derived(page.params.id)
  let lane = $derived(page.url.searchParams.get('lane') as 'audio' | 'image' ?? 'audio')

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
    } catch {
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
      // non-fatal
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

  $effect(() => {
    if (groupId) fetchNext()
  })
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
      <button onclick={() => goto(`/groups/${groupId}`)}>Back to group</button>
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

- [ ] **Step 4: Run tests**

```bash
cd frontend && npm test -- 'src/routes/groups/\[id\]/quiz/+page.test.ts'
```

Expected: all 4 tests pass.

- [ ] **Step 5: Delete old quiz files**

```bash
rm frontend/src/views/Quiz.svelte frontend/src/views/Quiz.test.ts
```

- [ ] **Step 6: Run all frontend tests**

```bash
cd frontend && npm test
```

Expected: all tests pass.

- [ ] **Step 7: Commit**

```bash
jj describe -m "feat: quiz page reads groupId and lane from URL params"
jj new
```

---

### Task 14: Explore stub and final cleanup

**Files:**
- Create: `frontend/src/routes/explore/+page.svelte`
- Delete remaining old files

- [ ] **Step 1: Create `/explore` stub**

Create `frontend/src/routes/explore/+page.svelte`:

```svelte
<div class="explore-stub">
  <p>Explore is coming soon.</p>
</div>

<style>
  .explore-stub {
    display: flex;
    justify-content: center;
    align-items: center;
    padding: 4rem 0;
    color: var(--text-muted);
    font-size: 1rem;
  }
</style>
```

- [ ] **Step 2: Delete the Login test (if it tests view-specific behaviour)**

Check `frontend/src/views/Login.test.ts` -- if it still passes and tests only the Login component UI (sign-in button), leave it. If it imports deleted stores (`view`, `session`), update it to remove those imports.

Run:
```bash
cd frontend && npm test -- src/views/Login.test.ts
```

If it fails, open the file, remove any references to `view` or `session` stores, and rerun.

- [ ] **Step 3: Run all tests -- backend and frontend**

```bash
just test
cd frontend && npm test
```

Expected: all tests pass.

- [ ] **Step 4: Verify the SvelteKit build compiles**

```bash
cd frontend && npx svelte-kit sync && npm run build
```

Expected: no errors. The `build/` directory is produced with `index.html` at root (the SPA fallback).

- [ ] **Step 5: Commit**

```bash
jj describe -m "feat: explore stub, complete SvelteKit migration"
jj new
```

---

## Self-Review Checklist

**Spec coverage:**
- [x] GET /api/v1/groups -- Task 3 (`listGroups`)
- [x] POST /api/v1/groups -- Task 3 (`createGroup`)
- [x] PATCH /api/v1/groups/:id -- Task 4 (`updateGroup`)
- [x] DELETE /api/v1/groups/:id -- Task 4 (`deleteGroup`)
- [x] GET /api/v1/groups/:id/species -- Task 5 (`listGroupSpecies`)
- [x] POST /api/v1/groups/:id/species -- Task 5 (`addSpeciesToGroup`, upserts cards for both lanes)
- [x] DELETE /api/v1/groups/:id/species/:species_id -- Task 5 (`removeSpeciesFromGroup`)
- [x] GET /api/v1/species?q= -- Task 6 (`searchSpecies`)
- [x] Auth check: 403 if wrong owner -- covered in Task 4 tests (`TestUpdateGroup_WrongOwner_Returns403`)
- [x] Add species auto-creates cards -- Task 5 tests assert both lanes upserted
- [x] SvelteKit SPA mode -- Task 8 (`adapter-static`, `fallback: 'index.html'`)
- [x] Auth in layout -- Task 9 (`+layout.svelte` checks `/api/v1/me`)
- [x] Dashboard real API -- Task 10
- [x] `/groups` CRUD -- Task 11
- [x] `/groups/[id]` detail + search -- Task 12
- [x] `/groups/[id]/quiz` URL params -- Task 13
- [x] `/explore` stub -- Task 14

**`withChiParams` added:** Task 5 Step 1 adds the multi-param helper to `quiz_test.go`.

**`groupOwnerCheck` uses `pgtype.Int8`:** `!group.OwnerID.Valid || group.OwnerID.Int64 != userID` -- handles nullable column correctly.

**Empty slice vs nil:** `listGroups` and `listGroupSpecies` coerce `nil` to `[]T{}` so the JSON response is `[]` not `null`.

**Cards not deleted on species removal:** `removeSpeciesFromGroup` only deletes from `group_species`, not `cards` -- matches spec.
