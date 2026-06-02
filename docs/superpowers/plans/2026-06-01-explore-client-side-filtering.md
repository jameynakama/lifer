# Explore Page: Client-Side Filtering Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the per-keystroke explore page search with a single all-species fetch + client-side filtering, eliminating backend hammering on every keystroke.

**Architecture:** New `GET /api/v1/species/all` endpoint returns the full catalog as a flat JSON array. The explore page fetches it once per session (`staleTime: Infinity` via TanStack Query) and filters/paginates entirely in-browser using Svelte 5 `$derived`. No debounce logic needed -- `$derived` is synchronous and instant at this data scale (~600 PNW species, at most ~2000 for all of US).

**Tech Stack:** Go + chi + sqlc (backend), SvelteKit + TanStack Query v6 + Svelte 5 runes (frontend), vitest + @testing-library/svelte (frontend tests), gotestsum / `just test` (backend tests), jj (version control)

---

### Task 1: Add `ListAllSpecies` SQL query and regenerate

**Files:**
- Modify: `backend/internal/store/queries/species.sql`
- Generated (auto): `backend/internal/store/species.sql.go`, `backend/internal/store/querier.go`

- [ ] **Step 1: Add query to species.sql**

Append to `backend/internal/store/queries/species.sql`:

```sql
-- name: ListAllSpecies :many
SELECT
    ebird_code,
    common_name,
    scientific_name,
    (SELECT file_path FROM species_images WHERE species_code = species.ebird_code LIMIT 1) AS image_url
FROM species
ORDER BY common_name;
```

- [ ] **Step 2: Regenerate sqlc types**

```bash
just generate
```

Expected: no errors. `backend/internal/store/species.sql.go` gains a `ListAllSpeciesRow` struct and `ListAllSpecies` method. `backend/internal/store/querier.go` gains `ListAllSpecies(ctx context.Context) ([]ListAllSpeciesRow, error)` in the `Querier` interface.

- [ ] **Step 3: Verify the build compiles**

```bash
cd backend && go build ./...
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
jj commit -m "feat(store): add ListAllSpecies query"
```

---

### Task 2: Add `listAllSpecies` handler (TDD)

**Files:**
- Modify: `backend/internal/api/species_test.go`
- Modify: `backend/internal/api/species.go`

Test helpers (`makeHandler`, `injectUserID`) are defined in `backend/internal/api/quiz_test.go` -- available to all `package api` test files automatically.

- [ ] **Step 1: Write the failing tests**

First, add `"errors"` to the import block at the top of `backend/internal/api/species_test.go`:

```go
import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jameynakama/lifer/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)
```

Then add to the bottom of `backend/internal/api/species_test.go`:

```go
// allSpeciesStubQuerier stubs ListAllSpecies.
type allSpeciesStubQuerier struct {
	store.Querier
	listAllSpecies func(ctx context.Context) ([]store.ListAllSpeciesRow, error)
}

func (s *allSpeciesStubQuerier) ListAllSpecies(ctx context.Context) ([]store.ListAllSpeciesRow, error) {
	return s.listAllSpecies(ctx)
}

func TestListAllSpecies_ReturnsFullCatalog(t *testing.T) {
	q := &allSpeciesStubQuerier{
		listAllSpecies: func(_ context.Context) ([]store.ListAllSpeciesRow, error) {
			return []store.ListAllSpeciesRow{
				{EbirdCode: "amro", CommonName: "American Robin", ScientificName: "Turdus migratorius", ImageUrl: "https://r2.example.com/amro.jpg"},
				{EbirdCode: "bcch", CommonName: "Black-capped Chickadee", ScientificName: "Poecile atricapillus", ImageUrl: ""},
			}, nil
		},
	}
	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/species/all", nil)
	r = injectUserID(r, 1)
	w := httptest.NewRecorder()

	h.listAllSpecies(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var body []SpeciesItem
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Len(t, body, 2)
	assert.Equal(t, "American Robin", body[0].CommonName)
	require.NotNil(t, body[0].ImageURL)
	assert.Equal(t, "https://r2.example.com/amro.jpg", *body[0].ImageURL)
	assert.Equal(t, "Black-capped Chickadee", body[1].CommonName)
	assert.Nil(t, body[1].ImageURL)
}

func TestListAllSpecies_EmptyDB_ReturnsEmptyArray(t *testing.T) {
	q := &allSpeciesStubQuerier{
		listAllSpecies: func(_ context.Context) ([]store.ListAllSpeciesRow, error) {
			return []store.ListAllSpeciesRow{}, nil
		},
	}
	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/species/all", nil)
	r = injectUserID(r, 1)
	w := httptest.NewRecorder()

	h.listAllSpecies(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var body []SpeciesItem
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.NotNil(t, body)
	assert.Empty(t, body)
}

func TestListAllSpecies_DBError_Returns500(t *testing.T) {
	q := &allSpeciesStubQuerier{
		listAllSpecies: func(_ context.Context) ([]store.ListAllSpeciesRow, error) {
			return nil, errors.New("db down")
		},
	}
	h := makeHandler(q)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/species/all", nil)
	r = injectUserID(r, 1)
	w := httptest.NewRecorder()

	h.listAllSpecies(w, r)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd backend && go test ./internal/api/ -run TestListAllSpecies -v
```

Expected: compile error -- `h.listAllSpecies undefined`.

- [ ] **Step 3: Implement the handler**

Add to `backend/internal/api/species.go` (after `listSpecies`):

```go
// listAllSpecies handles GET /api/v1/species/all.
// Returns the full species catalog as a flat array with no pagination.
func (h *Handler) listAllSpecies(w http.ResponseWriter, r *http.Request) {
	rows, err := h.queries.ListAllSpecies(r.Context())
	if err != nil {
		log.Printf("ListAllSpecies error: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	results := make([]SpeciesItem, len(rows))
	for i, row := range rows {
		var imageURL *string
		if row.ImageUrl != "" {
			imageURL = &row.ImageUrl
		}
		results[i] = SpeciesItem{
			EbirdCode:      row.EbirdCode,
			CommonName:     row.CommonName,
			ScientificName: row.ScientificName,
			ImageURL:       imageURL,
		}
	}
	writeJSON(w, http.StatusOK, results)
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd backend && go test ./internal/api/ -run TestListAllSpecies -v
```

Expected: `TestListAllSpecies_ReturnsFullCatalog`, `TestListAllSpecies_EmptyDB_ReturnsEmptyArray`, `TestListAllSpecies_DBError_Returns500` all PASS.

- [ ] **Step 5: Run full backend suite**

```bash
just test
```

Expected: all tests PASS.

- [ ] **Step 6: Commit**

```bash
jj commit -m "feat(api): add listAllSpecies handler"
```

---

### Task 3: Register the route

**Files:**
- Modify: `backend/internal/api/router.go`

- [ ] **Step 1: Add the route**

In `backend/internal/api/router.go`, inside the `RequireAuth` group, add `/species/all` **before** `/species/{ebird_code}` so chi's static-before-param precedence applies:

```go
r.Get("/species", h.listSpecies)
r.Get("/species/all", h.listAllSpecies)
r.Get("/species/{ebird_code}", h.getSpeciesDetail)
r.Get("/species/{ebird_code}/groups", h.getSpeciesGroups)
r.Put("/species/{ebird_code}/preferences", h.updatePreferences)
```

- [ ] **Step 2: Verify tests still pass**

```bash
just test
```

Expected: all tests PASS.

- [ ] **Step 3: Commit**

```bash
jj commit -m "feat(router): register GET /api/v1/species/all"
```

---

### Task 4: Add `SpeciesListItem` type

**Files:**
- Modify: `frontend/src/types.ts`

- [ ] **Step 1: Add the interface**

In `frontend/src/types.ts`, append after the `Species` interface:

```ts
export interface SpeciesListItem {
    ebird_code: string;
    common_name: string;
    scientific_name: string;
    image_url: string | null;
}
```

- [ ] **Step 2: Verify frontend build**

```bash
cd frontend && npm run test
```

Expected: all existing tests PASS (new type is unused so far).

- [ ] **Step 3: Commit**

```bash
jj commit -m "feat(types): add SpeciesListItem interface"
```

---

### Task 5: Update explore page tests and rewrite the page (TDD)

**Files:**
- Modify: `frontend/src/routes/explore/page.test.ts`
- Modify: `frontend/src/routes/explore/+page.svelte`

- [ ] **Step 1: Update the tests**

Replace the full contents of `frontend/src/routes/explore/page.test.ts`:

```ts
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/svelte'
import ExplorePage from './+page.svelte'

vi.mock('@tanstack/svelte-query', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/svelte-query')>()
  return {
    ...actual,
    createQuery: vi.fn(),
  }
})

const mockSpecies = [
  { ebird_code: 'amro', common_name: 'American Robin', scientific_name: 'Turdus migratorius', image_url: null },
  { ebird_code: 'bcch', common_name: 'Black-capped Chickadee', scientific_name: 'Poecile atricapillus', image_url: null },
]

beforeEach(async () => {
  const { createQuery } = await import('@tanstack/svelte-query')
  vi.mocked(createQuery).mockReturnValue({
    data: mockSpecies,
    isPending: false,
    isError: false,
  } as any)
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe('Explore page', () => {
  it('renders species rows from API response', async () => {
    render(ExplorePage)
    await vi.waitFor(() => {
      expect(screen.getByText(/american robin/i)).toBeTruthy()
      expect(screen.getByText(/black-capped chickadee/i)).toBeTruthy()
    })
  })

  it('shows pagination when species count exceeds limit', async () => {
    const { createQuery } = await import('@tanstack/svelte-query')
    const manySpecies = Array.from({ length: 25 }, (_, i) => ({
      ebird_code: `sp${i}`,
      common_name: `Species ${i}`,
      scientific_name: `Genus species${i}`,
      image_url: null,
    }))
    vi.mocked(createQuery).mockReturnValue({
      data: manySpecies,
      isPending: false,
      isError: false,
    } as any)
    render(ExplorePage)
    await vi.waitFor(() => {
      expect(screen.getByRole('navigation', { name: /pagination/i })).toBeTruthy()
    })
  })

  it('filters species by common name when searching', async () => {
    render(ExplorePage)
    const input = screen.getByPlaceholderText(/search species/i)
    await fireEvent.input(input, { target: { value: 'robin' } })
    await vi.waitFor(() => {
      expect(screen.getByText(/american robin/i)).toBeTruthy()
      expect(screen.queryByText(/black-capped chickadee/i)).toBeNull()
    })
  })

  it('filters species by scientific name when searching', async () => {
    render(ExplorePage)
    const input = screen.getByPlaceholderText(/search species/i)
    await fireEvent.input(input, { target: { value: 'poecile' } })
    await vi.waitFor(() => {
      expect(screen.getByText(/black-capped chickadee/i)).toBeTruthy()
      expect(screen.queryByText(/american robin/i)).toBeNull()
    })
  })

  it('shows all species when search query is shorter than 2 chars', async () => {
    render(ExplorePage)
    const input = screen.getByPlaceholderText(/search species/i)
    await fireEvent.input(input, { target: { value: 'r' } })
    await vi.waitFor(() => {
      expect(screen.getByText(/american robin/i)).toBeTruthy()
      expect(screen.getByText(/black-capped chickadee/i)).toBeTruthy()
    })
  })

  it('shows loading state', async () => {
    const { createQuery } = await import('@tanstack/svelte-query')
    vi.mocked(createQuery).mockReturnValue({
      data: undefined,
      isPending: true,
      isError: false,
    } as any)
    render(ExplorePage)
    expect(screen.getByText(/loading/i)).toBeTruthy()
  })

  it('shows error state', async () => {
    const { createQuery } = await import('@tanstack/svelte-query')
    vi.mocked(createQuery).mockReturnValue({
      data: undefined,
      isPending: false,
      isError: true,
    } as any)
    render(ExplorePage)
    expect(screen.getByText(/couldn't load species/i)).toBeTruthy()
  })
})
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd frontend && npm run test -- --reporter=verbose src/routes/explore/page.test.ts
```

Expected: "filters" and "shows all species" tests FAIL (page still uses old paginated shape).

- [ ] **Step 3: Rewrite the explore page**

Replace the full contents of `frontend/src/routes/explore/+page.svelte`:

```svelte
<script lang="ts">
  import { createQuery } from '@tanstack/svelte-query'
  import type { SpeciesListItem } from '../../types'
  import SpeciesRow from '$components/SpeciesRow.svelte'
  import PaginationBar from '$components/PaginationBar.svelte'

  const defaultLimit = 20

  let q = $state('')
  let offset = $state(0)

  const allSpeciesQuery = createQuery(() => ({
    queryKey: ['species', 'all'],
    queryFn: (): Promise<SpeciesListItem[]> =>
      fetch('/api/v1/species/all').then((r) => r.json()),
    staleTime: Infinity,
  }))

  const allSpecies = $derived(allSpeciesQuery.data ?? [])

  const filtered = $derived(
    q.length < 2
      ? allSpecies
      : allSpecies.filter((s) => {
          const lq = q.toLowerCase()
          return (
            s.common_name.toLowerCase().includes(lq) ||
            s.scientific_name.toLowerCase().includes(lq)
          )
        })
  )

  const displayed = $derived(filtered.slice(offset, offset + defaultLimit))

  function onPageChange(newOffset: number) {
    offset = newOffset
  }
</script>

<div class="explore">
  <input
    type="text"
    placeholder="Search species…"
    bind:value={q}
    oninput={() => { offset = 0 }}
  />

  {#if allSpeciesQuery.isPending}
    <p class="status">Loading…</p>
  {:else if allSpeciesQuery.isError}
    <p class="status error">Couldn't load species.</p>
  {:else}
    <ul class="species-list">
      {#each displayed as s (s.ebird_code)}
        <li>
          <SpeciesRow
            ebird_code={s.ebird_code}
            common_name={s.common_name}
            scientific_name={s.scientific_name}
            image_url={s.image_url ?? null}
          />
        </li>
      {/each}
    </ul>

    <PaginationBar
      total={filtered.length}
      {offset}
      limit={defaultLimit}
      {onPageChange}
    />
  {/if}
</div>

<style>
  .explore {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  input[type='text'] {
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

  input[type='text']:focus {
    outline: none;
    border-color: var(--accent);
  }

  .species-list {
    list-style: none;
    padding: 0;
    margin: 0;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .status {
    text-align: center;
    color: var(--text-muted);
    padding: 2rem 0;
  }
</style>
```

- [ ] **Step 4: Run explore tests to verify they pass**

```bash
cd frontend && npm run test -- --reporter=verbose src/routes/explore/page.test.ts
```

Expected: all 7 tests PASS.

- [ ] **Step 5: Run full frontend test suite**

```bash
cd frontend && npm run test
```

Expected: all tests PASS.

- [ ] **Step 6: Commit**

```bash
jj commit -m "feat(explore): client-side filtering via /api/v1/species/all"
```
