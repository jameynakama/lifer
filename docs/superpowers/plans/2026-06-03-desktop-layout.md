# Desktop Layout Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix the 480px mobile-only container, add a 4-tile dashboard stats bar with a live "next due in" countdown, and make the deck list a 2-column grid on desktop.

**Architecture:** The global `.app-container` max-width changes from 480px to 900px; quiz/practice pages add a local 560px inner wrapper. The `GET /api/v1/decks` response changes from a bare array to `{ decks, next_due_at }`. A new `DashboardStats` component handles the 4-tile countdown (StatsBar is unchanged -- it's still used by quiz and practice with its `Stat[]` interface). The mobile nav header becomes two rows below 640px via CSS only.

**Tech Stack:** Go + sqlc + pgx, SvelteKit (Svelte 5 runes), Vitest + Testing Library, TypeScript

---

## File map

| File | Action | What changes |
|---|---|---|
| `backend/internal/store/queries/cards.sql` | Modify | Add `GetNextDueAt` query |
| `backend/internal/store/cards.sql.go` | Regenerate | sqlc output for new query |
| `backend/internal/store/querier.go` | Regenerate | New method in interface |
| `backend/internal/api/decks.go` | Modify | New response struct + updated `listDecks` |
| `backend/internal/api/decks_test.go` | Modify | Update stub + existing test + 2 new tests |
| `frontend/src/types.ts` | Modify | Add `DecksResponse` interface |
| `frontend/src/routes/+layout.svelte` | Modify | 900px container, two-row mobile nav |
| `frontend/src/components/DashboardStats.svelte` | Create | 4-tile stats + countdown |
| `frontend/src/components/DashboardStats.test.ts` | Create | Tests for all 3 countdown states |
| `frontend/src/components/DeckList.svelte` | Modify | 2-column CSS grid |
| `frontend/src/routes/+page.svelte` | Modify | New response shape, use DashboardStats |
| `frontend/src/routes/page.test.ts` | Modify | Update fetch mock to new response shape |
| `frontend/src/routes/decks/+page.svelte` | Modify | Handle new response shape |
| `frontend/src/routes/decks/page.test.ts` | Modify | Update fetch mock to new response shape |
| `frontend/src/routes/decks/[id]/quiz/+page.svelte` | Modify | Add 560px inner wrapper |
| `frontend/src/routes/decks/[id]/practice/+page.svelte` | Modify | Add 560px inner wrapper |

---

## Task 1: Add GetNextDueAt SQL query

**Files:**
- Modify: `backend/internal/store/queries/cards.sql`

- [ ] **Step 1: Add the query**

Append to the end of `backend/internal/store/queries/cards.sql`:

```sql
-- name: GetNextDueAt :one
SELECT MIN(due) AS next_due_at
FROM cards
WHERE user_id = $1
  AND due > NOW();
```

- [ ] **Step 2: Regenerate sqlc**

```bash
just generate
```

Expected: exits 0. Three files update: `cards.sql.go`, `querier.go`, and possibly `db.go`. Verify `querier.go` now contains:

```go
GetNextDueAt(ctx context.Context, userID int64) (pgtype.Timestamptz, error)
```

And `cards.sql.go` contains a `GetNextDueAt` function.

- [ ] **Step 3: Verify backend still compiles**

```bash
cd backend && go build ./...
```

Expected: exits 0, no output.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/store/queries/cards.sql backend/internal/store/cards.sql.go backend/internal/store/querier.go
git commit -m "feat: add GetNextDueAt SQL query"
```

---

## Task 2: Update listDecks handler to return `{ decks, next_due_at }`

**Files:**
- Modify: `backend/internal/api/decks.go`
- Modify: `backend/internal/api/decks_test.go`

- [ ] **Step 1: Write the failing tests**

In `backend/internal/api/decks_test.go`:

1. Add `getNextDueAt` to `deckStubQuerier` (struct field + method):

```go
type deckStubQuerier struct {
    store.Querier
    listUserDecks            func(ctx context.Context, userID int64) ([]store.ListUserDecksRow, error)
    createDeck               func(ctx context.Context, arg store.CreateDeckParams) (store.Deck, error)
    getDeck                  func(ctx context.Context, id int64) (store.Deck, error)
    updateDeckName           func(ctx context.Context, arg store.UpdateDeckNameParams) (store.Deck, error)
    deleteDeck               func(ctx context.Context, id int64) error
    listDeckSpeciesWithPrefs func(ctx context.Context, arg store.ListDeckSpeciesWithPrefsParams) ([]store.ListDeckSpeciesWithPrefsRow, error)
    addSpeciesToDeck         func(ctx context.Context, arg store.AddSpeciesToDeckParams) error
    removeSpeciesFromDeck    func(ctx context.Context, arg store.RemoveSpeciesFromDeckParams) error
    upsertCard               func(ctx context.Context, arg store.UpsertCardParams) error
    getNextDueAt             func(ctx context.Context, userID int64) (pgtype.Timestamptz, error)
}

func (s *deckStubQuerier) GetNextDueAt(ctx context.Context, userID int64) (pgtype.Timestamptz, error) {
    if s.getNextDueAt != nil {
        return s.getNextDueAt(ctx, userID)
    }
    return pgtype.Timestamptz{}, nil
}
```

2. Update `TestListDecks_ReturnsList` to use the new response shape:

```go
func TestListDecks_ReturnsList(t *testing.T) {
    q := &deckStubQuerier{
        listUserDecks: func(_ context.Context, userID int64) ([]store.ListUserDecksRow, error) {
            assert.Equal(t, int64(1), userID)
            return []store.ListUserDecksRow{
                {ID: 1, Name: "My Warblers", OwnerID: ownerID(1), AudioDue: 3, ImageDue: 1},
            }, nil
        },
    }
    h := makeHandler(q)
    r := httptest.NewRequest(http.MethodGet, "/api/v1/decks", nil)
    r = injectUserID(r, 1)
    w := httptest.NewRecorder()

    h.listDecks(w, r)

    assert.Equal(t, http.StatusOK, w.Code)
    var body listDecksResponse
    require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
    assert.Len(t, body.Decks, 1)
    assert.Equal(t, "My Warblers", body.Decks[0].Name)
    assert.Equal(t, int64(3), body.Decks[0].AudioDue)
    assert.Nil(t, body.NextDueAt)
}
```

3. Add two new tests after `TestListDecks_ReturnsList`:

```go
func TestListDecks_NextDueAtSetWhenFutureCardExists(t *testing.T) {
    futureTime := time.Now().Add(2 * time.Hour)
    q := &deckStubQuerier{
        listUserDecks: func(_ context.Context, _ int64) ([]store.ListUserDecksRow, error) {
            return []store.ListUserDecksRow{}, nil
        },
        getNextDueAt: func(_ context.Context, _ int64) (pgtype.Timestamptz, error) {
            return pgtype.Timestamptz{Time: futureTime, Valid: true}, nil
        },
    }
    h := makeHandler(q)
    r := httptest.NewRequest(http.MethodGet, "/api/v1/decks", nil)
    r = injectUserID(r, 1)
    w := httptest.NewRecorder()

    h.listDecks(w, r)

    assert.Equal(t, http.StatusOK, w.Code)
    var body listDecksResponse
    require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
    require.NotNil(t, body.NextDueAt)
    parsed, err := time.Parse(time.RFC3339, *body.NextDueAt)
    require.NoError(t, err)
    assert.WithinDuration(t, futureTime, parsed, time.Second)
}

func TestListDecks_NextDueAtNullWhenNofutureCards(t *testing.T) {
    q := &deckStubQuerier{
        listUserDecks: func(_ context.Context, _ int64) ([]store.ListUserDecksRow, error) {
            return []store.ListUserDecksRow{}, nil
        },
        getNextDueAt: func(_ context.Context, _ int64) (pgtype.Timestamptz, error) {
            return pgtype.Timestamptz{Valid: false}, nil
        },
    }
    h := makeHandler(q)
    r := httptest.NewRequest(http.MethodGet, "/api/v1/decks", nil)
    r = injectUserID(r, 1)
    w := httptest.NewRecorder()

    h.listDecks(w, r)

    assert.Equal(t, http.StatusOK, w.Code)
    var body listDecksResponse
    require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
    assert.Nil(t, body.NextDueAt)
}
```

Make sure `"time"` is imported in `decks_test.go`.

- [ ] **Step 2: Run tests to confirm they fail**

```bash
just test
```

Expected: compile error (`listDecksResponse` undefined) or test failures. Tests failing or not compiling confirms the right direction.

- [ ] **Step 3: Implement the handler change**

In `backend/internal/api/decks.go`, add `"time"` to the import block. Then add the response struct and update `listDecks`:

```go
type listDecksResponse struct {
    Decks     []store.ListUserDecksRow `json:"decks"`
    NextDueAt *string                  `json:"next_due_at"`
}

func (h *Handler) listDecks(w http.ResponseWriter, r *http.Request) {
    userID := auth.UserIDFromCtx(r.Context())
    decks, err := h.queries.ListUserDecks(r.Context(), userID)
    if err != nil {
        log.Printf("ListUserDecks error: %v", err)
        http.Error(w, "server error", http.StatusInternalServerError)
        return
    }
    if decks == nil {
        decks = []store.ListUserDecksRow{}
    }
    nextDue, err := h.queries.GetNextDueAt(r.Context(), userID)
    if err != nil {
        log.Printf("GetNextDueAt error: %v", err)
        http.Error(w, "server error", http.StatusInternalServerError)
        return
    }
    resp := listDecksResponse{Decks: decks}
    if nextDue.Valid {
        t := nextDue.Time.UTC().Format(time.RFC3339)
        resp.NextDueAt = &t
    }
    writeJSON(w, http.StatusOK, resp)
}
```

- [ ] **Step 4: Run tests to confirm they pass**

```bash
just test
```

Expected: all tests pass, no failures.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/api/decks.go backend/internal/api/decks_test.go
git commit -m "feat: listDecks returns next_due_at alongside deck list"
```

---

## Task 3: Frontend types

**Files:**
- Modify: `frontend/src/types.ts`

- [ ] **Step 1: Add DecksResponse interface**

In `frontend/src/types.ts`, add after the `Deck` interface:

```typescript
export interface DecksResponse {
    decks: Deck[];
    next_due_at: string | null;
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/types.ts
git commit -m "feat: add DecksResponse type for new listDecks shape"
```

---

## Task 4: Layout -- widen container and add two-row mobile nav

**Files:**
- Modify: `frontend/src/routes/+layout.svelte`

The layout test (`layout.test.ts`) doesn't test CSS structure, so no test changes are needed.

- [ ] **Step 1: Update +layout.svelte**

Replace the entire `<style>` block in `frontend/src/routes/+layout.svelte` with:

```svelte
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
  .app-container {
    max-width: 900px;
    margin: 0 auto;
    padding: 0 1.5rem 2rem;
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
  .header-actions {
    display: flex;
    gap: 0.5rem;
    align-items: center;
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
  .logout {
    font-size: 0.75rem !important;
  }
  main {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  /* Mobile: two-row header */
  @media (max-width: 639px) {
    .app-container {
      padding: 0 1rem 2rem;
    }
    header {
      flex-wrap: wrap;
      padding: 0.75rem 0 0;
      gap: 0.25rem 0;
    }
    .wordmark {
      flex: 1;
    }
    nav {
      order: 3;
      width: 100%;
      padding: 0.375rem 0 0.625rem;
      border-top: 1px solid var(--border);
      margin-top: 0.25rem;
    }
  }
</style>
```

- [ ] **Step 2: Run frontend tests**

```bash
cd frontend && npm test
```

Expected: all tests pass (layout tests check text content and auth state, not CSS structure).

- [ ] **Step 3: Commit**

```bash
git add frontend/src/routes/+layout.svelte
git commit -m "feat: widen layout to 900px, add two-row mobile nav"
```

---

## Task 5: Create DashboardStats component

**Files:**
- Create: `frontend/src/components/DashboardStats.svelte`
- Create: `frontend/src/components/DashboardStats.test.ts`

The existing `StatsBar` component is **not changed** -- it's still used by the quiz and practice pages with its `Stat[]` interface.

- [ ] **Step 1: Write the failing tests**

Create `frontend/src/components/DashboardStats.test.ts`:

```typescript
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen } from '@testing-library/svelte'
import DashboardStats from './DashboardStats.svelte'

afterEach(() => {
  vi.useRealTimers()
})

describe('DashboardStats', () => {
  it('renders all four tile labels', () => {
    render(DashboardStats, { props: { audioDue: 0, imageDue: 0, nextDueAt: null } })
    expect(screen.getByText('DUE TODAY')).toBeInTheDocument()
    expect(screen.getByText('AUDIO DUE')).toBeInTheDocument()
    expect(screen.getByText('IMAGE DUE')).toBeInTheDocument()
    expect(screen.getByText('NEXT DUE IN')).toBeInTheDocument()
  })

  it('shows totals in the first three tiles', () => {
    render(DashboardStats, { props: { audioDue: 3, imageDue: 5, nextDueAt: null } })
    expect(screen.getByText('8')).toBeInTheDocument()  // total = 3 + 5
    expect(screen.getByText('3')).toBeInTheDocument()  // audio
    expect(screen.getByText('5')).toBeInTheDocument()  // image
  })

  it('shows "Now" in accent color when cards are due', () => {
    render(DashboardStats, { props: { audioDue: 2, imageDue: 1, nextDueAt: '2099-01-01T00:00:00Z' } })
    const el = screen.getByText('Now')
    expect(el).toBeInTheDocument()
    expect(el).toHaveStyle({ color: 'var(--accent)' })
  })

  it('shows "--" when caught up and no future cards', () => {
    render(DashboardStats, { props: { audioDue: 0, imageDue: 0, nextDueAt: null } })
    expect(screen.getByText('--')).toBeInTheDocument()
  })

  it('shows countdown when caught up and nextDueAt is in the future', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-06-03T10:00:00Z'))
    const nextDueAt = '2026-06-03T12:30:00Z'  // 2h 30m in the future
    render(DashboardStats, { props: { audioDue: 0, imageDue: 0, nextDueAt } })
    expect(screen.getByText('2h 30m')).toBeInTheDocument()
  })

  it('shows minutes only when under 1 hour', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-06-03T10:00:00Z'))
    const nextDueAt = '2026-06-03T10:45:00Z'  // 45m in the future
    render(DashboardStats, { props: { audioDue: 0, imageDue: 0, nextDueAt } })
    expect(screen.getByText('45m')).toBeInTheDocument()
  })
})
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
cd frontend && npm test -- --reporter=verbose DashboardStats
```

Expected: fails with "Cannot find module './DashboardStats.svelte'".

- [ ] **Step 3: Create the component**

Create `frontend/src/components/DashboardStats.svelte`:

```svelte
<script lang="ts">
  let {
    audioDue,
    imageDue,
    nextDueAt,
  }: {
    audioDue: number;
    imageDue: number;
    nextDueAt: string | null;
  } = $props()

  const totalDue = $derived(audioDue + imageDue)

  function formatCountdown(target: Date): string {
    const ms = target.getTime() - Date.now()
    if (ms <= 0) return 'Now'
    const totalMinutes = Math.floor(ms / 60_000)
    const hours = Math.floor(totalMinutes / 60)
    const minutes = totalMinutes % 60
    if (hours > 0) return `${hours}h ${minutes}m`
    if (minutes > 0) return `${minutes}m`
    return '<1m'
  }

  let countdown = $state<string>('--')
  let interval: ReturnType<typeof setInterval> | null = null

  $effect(() => {
    if (interval) {
      clearInterval(interval)
      interval = null
    }

    if (totalDue > 0) {
      countdown = 'Now'
      return
    }

    if (!nextDueAt) {
      countdown = '--'
      return
    }

    const target = new Date(nextDueAt)
    countdown = formatCountdown(target)
    interval = setInterval(() => {
      countdown = formatCountdown(target)
    }, 30_000)

    return () => {
      if (interval) {
        clearInterval(interval)
        interval = null
      }
    }
  })
</script>

<div class="stats-bar">
  <div class="stat">
    <span class="value">{totalDue}</span>
    <span class="label">DUE TODAY</span>
  </div>
  <div class="stat">
    <span class="value">{audioDue}</span>
    <span class="label">AUDIO DUE</span>
  </div>
  <div class="stat">
    <span class="value">{imageDue}</span>
    <span class="label">IMAGE DUE</span>
  </div>
  <div class="stat">
    <span class="value" class:now={totalDue > 0}>{countdown}</span>
    <span class="label">NEXT DUE IN</span>
  </div>
</div>

<style>
  .stats-bar {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 0.625rem;
  }
  .stat {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: 10px;
    padding: 0.875rem 1rem;
    display: flex;
    flex-direction: column;
    box-shadow: var(--shadow);
  }
  .value {
    color: var(--text);
    font-size: 1.5rem;
    font-weight: 700;
    line-height: 1;
  }
  .value.now {
    color: var(--accent);
  }
  .label {
    color: var(--text-muted);
    font-size: 0.5625rem;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    margin-top: 0.3rem;
  }

  @media (max-width: 639px) {
    .stats-bar {
      grid-template-columns: repeat(2, 1fr);
    }
  }
</style>
```

- [ ] **Step 4: Run tests to confirm they pass**

```bash
cd frontend && npm test -- --reporter=verbose DashboardStats
```

Expected: all 6 tests pass.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/components/DashboardStats.svelte frontend/src/components/DashboardStats.test.ts
git commit -m "feat: add DashboardStats component with 4-tile countdown"
```

---

## Task 6: DeckList -- 2-column grid

**Files:**
- Modify: `frontend/src/components/DeckList.svelte`

The existing `DeckList.test.ts` checks rendered text and click behavior -- no test changes needed.

- [ ] **Step 1: Update the CSS**

In `frontend/src/components/DeckList.svelte`, replace the `.deck-list` CSS rule:

```css
/* old */
.deck-list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}
```

with:

```css
.deck-list {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 0.625rem;
}

@media (max-width: 639px) {
  .deck-list {
    grid-template-columns: 1fr;
  }
}
```

- [ ] **Step 2: Run frontend tests**

```bash
cd frontend && npm test
```

Expected: all tests pass.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/DeckList.svelte
git commit -m "feat: DeckList 2-column grid on desktop"
```

---

## Task 7: Dashboard page -- new response shape and DashboardStats

**Files:**
- Modify: `frontend/src/routes/+page.svelte`
- Modify: `frontend/src/routes/page.test.ts`

- [ ] **Step 1: Update the dashboard page test**

In `frontend/src/routes/page.test.ts`, keep the existing `decks` const as-is. Change every mock that returns it to wrap it in the new shape. Replace every `Promise.resolve(decks)` with `Promise.resolve({ decks, next_due_at: null })`. Also update the empty-state test:

```typescript
// Before
json: () => Promise.resolve(decks)

// After
json: () => Promise.resolve({ decks, next_due_at: null })
```

```typescript
// Empty state: before
json: () => Promise.resolve([])

// After
json: () => Promise.resolve({ decks: [], next_due_at: null })
```

Apply this to all four tests in the file.

- [ ] **Step 2: Run tests to confirm they fail**

```bash
cd frontend && npm test -- --reporter=verbose page
```

Expected: the deck-rendering tests fail because the dashboard page still tries to assign `res.json()` directly to `decks` (a bare array vs. `{ decks, next_due_at }`).

- [ ] **Step 3: Update +page.svelte**

Replace the entire contents of `frontend/src/routes/+page.svelte` with:

```svelte
<script lang="ts">
  import { goto } from '$app/navigation'
  import type { Deck, DecksResponse } from '../types'
  import DashboardStats from '$components/DashboardStats.svelte'
  import DeckList from '$components/DeckList.svelte'

  let decks: Deck[] = $state([])
  let nextDueAt: string | null = $state(null)
  let loading = $state(true)

  $effect(() => {
    fetch('/api/v1/decks')
      .then(async (res) => {
        if (res.ok) {
          const data: DecksResponse = await res.json()
          decks = data.decks
          nextDueAt = data.next_due_at
        }
      })
      .finally(() => { loading = false })
  })

  const audioDue = $derived(decks.reduce((sum, d) => sum + d.audio_due, 0))
  const imageDue = $derived(decks.reduce((sum, d) => sum + d.image_due, 0))

  function startPractice(deck: Deck, lane: 'audio' | 'image') {
    goto(`/decks/${deck.id}/quiz?lane=${lane}`)
  }
</script>

<div class="dashboard">
  {#if loading}
    <p class="status">Loading...</p>
  {:else if decks.length === 0}
    <p class="empty">No decks yet. <a href="/decks">Create one</a> to get started.</p>
  {:else}
    <DashboardStats {audioDue} {imageDue} {nextDueAt} />
    <DeckList {decks} onPractice={startPractice} />
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

- [ ] **Step 4: Run tests to confirm they pass**

```bash
cd frontend && npm test -- --reporter=verbose page
```

Expected: all tests pass.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/routes/+page.svelte frontend/src/routes/page.test.ts
git commit -m "feat: dashboard uses DashboardStats, handles new decks response shape"
```

---

## Task 8: /decks page -- handle new response shape

**Files:**
- Modify: `frontend/src/routes/decks/+page.svelte`
- Modify: `frontend/src/routes/decks/page.test.ts`

- [ ] **Step 1: Update the decks page test**

In `frontend/src/routes/decks/page.test.ts`, wrap the bare `decks` array in the new response shape everywhere fetch is mocked to return it. The `decks` const at the top of the file stays the same; update each mock that resolves with it.

Replace every `Promise.resolve(decks)` in a GET mock with `Promise.resolve({ decks, next_due_at: null })`. For example:

```typescript
// Before
vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
  ok: true, json: () => Promise.resolve(decks),
}))

// After
vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
  ok: true, json: () => Promise.resolve({ decks, next_due_at: null }),
}))
```

Apply this change to all tests that do a bare GET mock (the tests for renders, toggle, and delete). The `createDeck` test uses a `mockImplementation` that checks the method -- only replace the GET fallback branch. The `deckWithDue` test that uses a different deck array needs the same wrapping.

- [ ] **Step 2: Run tests to confirm they fail**

```bash
cd frontend && npm test -- --reporter=verbose decks/page
```

Expected: tests fail because the page still tries to assign `res.json()` directly to `decks`.

- [ ] **Step 3: Update decks/+page.svelte**

In `frontend/src/routes/decks/+page.svelte`, update `loadDecks` to destructure the response:

```typescript
async function loadDecks() {
  try {
    const res = await fetch('/api/v1/decks')
    if (res.ok) {
      const data = await res.json()
      decks = data.decks
    }
  } catch {
    // network error, loading still ends
  } finally {
    loading = false
  }
}
```

Add `import type { DecksResponse } from '../../../types'` at the top of the script block, and type the data:

```typescript
const data: DecksResponse = await res.json()
```

- [ ] **Step 4: Run tests to confirm they pass**

```bash
cd frontend && npm test -- --reporter=verbose decks/page
```

Expected: all tests pass.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/routes/decks/+page.svelte frontend/src/routes/decks/page.test.ts
git commit -m "fix: decks page handles new { decks, next_due_at } response shape"
```

---

## Task 9: Narrow quiz and practice pages to 560px

**Files:**
- Modify: `frontend/src/routes/decks/[id]/quiz/+page.svelte`
- Modify: `frontend/src/routes/decks/[id]/practice/+page.svelte`

These pages render within the 900px `.app-container`. Adding a local 560px inner wrapper keeps them focused without changing the header width.

- [ ] **Step 1: Update quiz page**

In `frontend/src/routes/decks/[id]/quiz/+page.svelte`, wrap the outermost `<div class="quiz">` in a centering container. Replace:

```svelte
<div class="quiz">
```

with:

```svelte
<div class="page-content">
<div class="quiz">
```

and at the end close it:

```svelte
</div>
</div>
```

Add to the `<style>` block:

```css
.page-content {
  max-width: 560px;
  margin: 0 auto;
}
```

- [ ] **Step 2: Update practice page**

Apply the same wrapper pattern to `frontend/src/routes/decks/[id]/practice/+page.svelte` (find the outermost div, wrap it in `.page-content` with the same CSS).

- [ ] **Step 3: Run all frontend tests**

```bash
cd frontend && npm test
```

Expected: all tests pass.

- [ ] **Step 4: Run all backend tests**

```bash
just test
```

Expected: all tests pass.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/routes/decks/[id]/quiz/+page.svelte frontend/src/routes/decks/[id]/practice/+page.svelte
git commit -m "feat: narrow quiz and practice pages to 560px"
```

---

## Done

All tasks complete. Verify the full test suite one final time:

```bash
just test && cd frontend && npm test
```

Then run the app locally to visually confirm:
```bash
just run   # backend on :8080
just frontend   # frontend dev server
```

Check:
- Dashboard: 4-tile stats bar, 2-column deck grid, wider layout
- Nav: two-row on narrow window, single-row on wide
- Quiz/practice: still narrow and focused
- /decks management page: unchanged behavior
