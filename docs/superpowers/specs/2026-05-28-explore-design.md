# Explore Feature Design

**Date:** 2026-05-28
**Status:** Approved

## Scope

Species catalog browse + detail pages, accessible via the "Explore" nav link (already wired to `/explore`). Region filter (eBird proxy) is explicitly out of scope -- separate task.

Users can:
- Browse all species in the catalog, paginated, sorted alphabetically by common name
- Search by common or scientific name (filters list, no pagination in search mode)
- View a species detail page (recordings, photos)
- Add/remove a species to/from any of their groups from the list row or the detail page
- Create a new group inline from the add-to-group dropdown

---

## Backend

### Constant

```go
// internal/api/species.go (or a shared constants file)
const defaultPageSize = 20
```

### Pagination response shape

All paginated endpoints return:

```json
{
  "count": 247,
  "next": "http://localhost:8080/api/v1/species?limit=20&offset=40",
  "previous": null,
  "results": [...]
}
```

- `next` / `previous`: absolute URL strings or `null`. Constructed server-side from `r.Host` + scheme (sniffed from `X-Forwarded-Proto` header, defaulting to `"http"`) + `r.URL.Path` + updated query params.
- `count`: total matching rows (from `COUNT(*) OVER()` window function -- no second query).
- `results`: array of items for this page.

### New sqlc queries

**`species.sql` additions:**

```sql
-- name: ListSpecies :many
SELECT
    ebird_code,
    common_name,
    scientific_name,
    COUNT(*) OVER() AS total_count
FROM species
ORDER BY common_name
LIMIT $1 OFFSET $2;

-- name: GetSpeciesByCode :one
SELECT ebird_code, common_name, scientific_name
FROM species
WHERE ebird_code = $1;

-- name: GetSpeciesRecordings :many
SELECT xeno_canto_id, file_path, quality, type
FROM species_recordings
WHERE species_code = $1
ORDER BY quality, type;

-- name: GetSpeciesImages :many
SELECT macaulay_id, file_path, credit
FROM species_images
WHERE species_code = $1;
```

```sql
-- name: GetGroupsForSpecies :many
SELECT group_id
FROM group_species
WHERE species_code = $1
  AND group_id IN (SELECT id FROM groups WHERE owner_id = $2);
```

Existing `SearchSpecies` stays as-is (used by the group detail typeahead, different shape/limit).

### Modified endpoint: `GET /api/v1/species`

**Query params:**
- `q` (optional): filter by common or scientific name (case-insensitive ILIKE)
- `limit` (optional): defaults to `defaultPageSize`, max 100
- `offset` (optional): defaults to 0

**Behaviour:**
- `q` empty → `ListSpecies(limit, offset)` → paginated response with `next`/`previous`
- `q` non-empty → existing `SearchSpecies` logic but with limit raised to 50; `next` and `previous` are always `null` (search results are not paginated)

**Response:** `PaginatedSpecies` struct:
```go
type PaginatedSpecies struct {
    Count    int64         `json:"count"`
    Next     *string       `json:"next"`
    Previous *string       `json:"previous"`
    Results  []SpeciesItem `json:"results"`
}

type SpeciesItem struct {
    EbirdCode      string `json:"ebird_code"`
    CommonName     string `json:"common_name"`
    ScientificName string `json:"scientific_name"`
}
```

### New endpoint: `GET /api/v1/species/:ebird_code`

Returns species detail: species fields + all recordings + all images. Three queries, assembled in the handler.

```go
type SpeciesDetail struct {
    EbirdCode      string      `json:"ebird_code"`
    CommonName     string      `json:"common_name"`
    ScientificName string      `json:"scientific_name"`
    Recordings     []Recording `json:"recordings"`
    Images         []Image     `json:"images"`
}

type Recording struct {
    XenoCantoID string `json:"xeno_canto_id"`
    FilePath    string `json:"file_path"`
    Quality     string `json:"quality"`
    Type        string `json:"type"`
}

type Image struct {
    MacaulayID string `json:"macaulay_id"`
    FilePath   string `json:"file_path"`
    Credit     string `json:"credit"`
}
```

404 if `ebird_code` not found.

New endpoint for dropdown checked state:

`GET /api/v1/species/:ebird_code/groups` -- returns `{ "group_ids": [1, 3] }` (IDs of the current user's groups that contain this species). Uses `GetGroupsForSpecies(ebirdCode, userID)`. TanStack query key: `['species', ebird_code, 'groups']`.

Router additions (inside the `RequireAuth` group):
```go
r.Get("/species/{ebird_code}", h.getSpeciesDetail)
r.Get("/species/{ebird_code}/groups", h.getSpeciesGroups)
```

---

## Frontend

### Dependency

Add `@tanstack/svelte-query`. Initialize `QueryClient` in `+layout.svelte`:

```svelte
<script lang="ts">
  import { QueryClient, QueryClientProvider } from '@tanstack/svelte-query'
  const queryClient = new QueryClient()
  // ...
</script>

<QueryClientProvider client={queryClient}>
  <!-- existing layout markup -->
</QueryClientProvider>
```

### Routes

| Route | File | Purpose |
|---|---|---|
| `/explore` | `src/routes/explore/+page.svelte` | List page (already stubbed) |
| `/explore/[ebird_code]` | `src/routes/explore/[ebird_code]/+page.svelte` | Detail page (new) |

### Components

All new components live in `src/components/`.

---

#### `SpeciesRow.svelte`

Thumbnail + names + group button. The whole row (except the group button) is a link to `/explore/[ebird_code]`.

**Exact styles:**
```css
.species-row {
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 0.625rem 1rem;
  display: flex;
  align-items: center;
  gap: 0.75rem;
  box-shadow: var(--shadow);
  text-decoration: none;
}

.thumbnail {
  width: 44px;
  height: 44px;
  border-radius: 8px;
  object-fit: cover;
  flex-shrink: 0;
  background: var(--border); /* shown while image loads */
}

.names {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
}

.common-name {
  font-size: 0.9375rem;
  font-weight: 600;
  color: var(--text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.scientific-name {
  font-size: 0.8125rem;
  color: var(--text-muted);
  font-style: italic;
}

.group-btn {
  background: var(--surface);
  border: 1px solid var(--border);
  color: var(--text-secondary);
  border-radius: 6px;
  padding: 0.25rem 0.5rem;
  font-size: 0.75rem;
  font-weight: 500;
  cursor: pointer;
  font-family: inherit;
  flex-shrink: 0;
  white-space: nowrap;
}

.group-btn:hover {
  border-color: var(--accent);
  color: var(--accent);
}
```

Props: `ebird_code: string`, `common_name: string`, `scientific_name: string`, `image_url: string | null`

---

#### `GroupDropdown.svelte`

Positioned dropdown anchored below the `+ Group` button. Closes on outside click (use `clickoutside` action or document listener).

**Exact styles:**
```css
.dropdown {
  position: absolute;
  top: calc(100% + 4px);
  right: 0;
  width: 220px;
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 10px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  z-index: 100;
  overflow: hidden;
}

.create-section {
  padding: 0.625rem 0.75rem;
  border-bottom: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
}

.create-input {
  background: var(--bg);
  border: 1px solid var(--border);
  color: var(--text);
  border-radius: 6px;
  padding: 0.375rem 0.5rem;
  font-size: 0.8125rem;
  font-family: inherit;
  width: 100%;
  box-sizing: border-box;
}

.create-btn {
  background: var(--accent);
  color: #fff;
  border: none;
  border-radius: 6px;
  padding: 0.3125rem 0.625rem;
  font-size: 0.75rem;
  font-weight: 600;
  cursor: pointer;
  font-family: inherit;
  width: 100%;
}

.groups-list {
  max-height: 200px;
  overflow-y: auto;
  padding: 0.375rem 0;
}

.group-item {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 0.75rem;
  cursor: pointer;
  font-size: 0.875rem;
  color: var(--text);
}

.group-item:hover {
  background: var(--bg);
}

.group-item input[type="checkbox"] {
  accent-color: var(--accent);
  flex-shrink: 0;
}
```

**Behaviour:**
- On open: fires two TanStack queries in parallel -- `['groups']` → `GET /api/v1/groups` (user's group list) and `['species', ebird_code, 'groups']` → `GET /api/v1/species/{ebird_code}/groups` (group IDs containing this species). Renders checkboxes checked/unchecked based on the intersection.
- Checkbox toggle: fires `POST /api/v1/groups/{id}/species` (add) or `DELETE /api/v1/groups/{id}/species/{ebird_code}` (remove). On success, invalidate `['species', ebird_code, 'groups']`.
- Create new group: `POST /api/v1/groups` with `{ name }`, then immediately `POST /api/v1/groups/{newId}/species` to add the current species. Invalidate both `['groups']` and `['species', ebird_code, 'groups']`.

Props: `ebird_code: string`

---

#### `PaginationBar.svelte`

Hidden when in search mode (parent passes `searchMode: boolean`). Shows `< 1 2 3 ... N >`.

**Exact styles:**
```css
.pagination {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.25rem;
  padding: 0.5rem 0;
}

.page-btn {
  background: var(--surface);
  border: 1px solid var(--border);
  color: var(--text-secondary);
  border-radius: 6px;
  padding: 0.3125rem 0.625rem;
  font-size: 0.875rem;
  cursor: pointer;
  font-family: inherit;
  min-width: 32px;
  text-align: center;
}

.page-btn:hover:not(:disabled) {
  border-color: var(--accent);
  color: var(--accent);
}

.page-btn.active {
  background: var(--accent);
  border-color: var(--accent);
  color: #fff;
  font-weight: 600;
}

.page-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.ellipsis {
  color: var(--text-muted);
  padding: 0 0.25rem;
  font-size: 0.875rem;
}
```

Ellipsis logic: show first page, last page, current page ±1, ellipsis where gaps exist. Standard pattern.

Props: `total: number`, `offset: number`, `limit: number`, `onPageChange: (offset: number) => void`

---

#### `PhotoGrid.svelte`

2-column grid of images.

**Exact styles:**
```css
.photo-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.375rem;
}

.photo-grid img {
  width: 100%;
  aspect-ratio: 4 / 3;
  object-fit: cover;
  border-radius: 6px;
  background: var(--border);
}
```

Props: `images: Image[]`

---

#### `RecordingsList.svelte`

Stacked rows, each containing a `.recording-meta` label and a `<WavePlayer>` instance (existing component, pass `src={recording.file_path}`).

**Exact styles:**
```css
.recordings-list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.recording-row {
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 0.5rem 0.75rem;
}

.recording-meta {
  font-size: 0.6875rem;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin-bottom: 0.375rem;
}
```

Props: `recordings: Recording[]`

---

### `/explore` page

Layout:
1. Search input (debounced 300ms, resets offset to 0 on change)
2. Species list: `{#each results as s}<SpeciesRow .../>{/each}`
3. `<PaginationBar>` (hidden when `q` non-empty)

TanStack query key: `['species', { q, limit, offset }]`

Search input styles (match existing pattern from `groups/[id]/+page.svelte`):
```css
input[type="text"] {
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
```

---

### `/explore/[ebird_code]` page

Layout (name-first):
1. Common name (`font-size: 1rem; font-weight: 700; color: var(--text)`) + `+ Group` button (same `.group-btn` style, opens `GroupDropdown`)
2. Scientific name (`font-size: 0.8125rem; font-style: italic; color: var(--text-muted)`)
3. Section: "Recordings" label + `<RecordingsList>`
4. Section: "Photos" label + `<PhotoGrid>`

Section label style:
```css
.section-label {
  font-size: 0.6875rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--text-secondary);
  margin-bottom: 0.5rem;
}
```

Section wrapper:
```css
.section {
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 0.75rem 1rem;
  box-shadow: var(--shadow);
}
```

Header (name + group button row):
```css
.species-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 0.75rem;
}
```

404 handling: if detail query returns 404, `goto('/explore')`.

TanStack query key: `['species', ebird_code]`

---

## Error handling

- List page: `isError` → inline message `"Couldn't load species."` below search bar, centered, `color: var(--text-muted)`
- Detail page: `isError` (non-404) → same inline message
- Group dropdown mutations: on error, show `"Failed. Try again."` in small text (`font-size: 0.75rem; color: #ef4444`) inside the dropdown, dismiss on next action
- 404 on detail → `goto('/explore')`

---

## Testing

**Backend** (`internal/api/species_test.go`):
- `GET /api/v1/species` -- no params (returns paginated list, correct `count`, `next` set, `previous` null)
- `GET /api/v1/species?offset=20` -- `previous` set, `next` set or null depending on total
- `GET /api/v1/species?q=robin` -- search mode, `next`/`previous` null regardless of result count
- `GET /api/v1/species/:ebird_code` -- returns species + recordings + images
- `GET /api/v1/species/nonexistent` -- 404
- `GET /api/v1/species/:ebird_code/groups` -- returns group IDs containing this species for current user
- `GET /api/v1/species/:ebird_code/groups` -- returns empty list when species is in no groups

**Frontend** (`src/routes/explore/page.test.ts`, `src/routes/explore/[ebird_code]/page.test.ts`):
- List renders species rows from mocked API response
- Pagination bar renders; hidden when search query is non-empty
- Detail page renders name, recordings count, images count from mocked response
- Detail page redirects to `/explore` on 404

---

## Out of scope (this iteration)

- Region filter (eBird proxy for state/county browsing) -- separate task
- Species edit/admin actions
- Sorting options other than alphabetical by common name
