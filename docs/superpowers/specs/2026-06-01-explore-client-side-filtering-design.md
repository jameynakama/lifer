# Explore Page: Client-Side Filtering

**Date:** 2026-06-01
**Status:** Approved

## Problem

The explore page search fires a backend request on every keystroke because the debounce
timer only resets `offset`, not the query key. At a small scale this is fine; under any
real load it hammers the backend needlessly. The species catalog is static between ingest
runs and bounded in size (~600 species for PNW, ~2000 at most for all of the US), making
it a natural fit for a single upfront fetch + client-side filtering.

## Approach

Load all species once per session; filter and paginate entirely in the browser.
No debounce logic needed -- `$derived` is synchronous and instant at this data scale.

## Backend

### New SQL query -- `ListAllSpecies`

Add to `backend/internal/store/queries/species.sql`:

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

Run `just generate` to produce the typed Go function.

### New handler -- `GET /api/v1/species/all`

New `listAllSpecies` handler in `backend/internal/api/species.go`. Calls
`h.queries.ListAllSpecies(ctx)`, maps rows to `[]SpeciesItem`, returns JSON.
Response shape: a flat array (no pagination envelope -- callers load it all at once).

```json
[
  { "ebird_code": "amiro", "common_name": "American Robin", "scientific_name": "Turdus migratorius", "image_url": "https://..." },
  ...
]
```

### Router

Register inside the existing `RequireAuth` group in `router.go`, before the
`/species/{ebird_code}` wildcard so chi's static-before-param precedence applies:

```
r.Get("/species/all", h.listAllSpecies)
r.Get("/species/{ebird_code}", h.getSpeciesDetail)
```

## Frontend

### Explore page rewrite (`src/routes/explore/+page.svelte`)

Replace the per-search `createQuery` with a single all-species query:

```js
const allSpeciesQuery = createQuery(() => ({
  queryKey: ['species', 'all'],
  queryFn: () => fetch('/api/v1/species/all').then((r) => r.json()),
  staleTime: Infinity,
}))
```

`staleTime: Infinity` means TanStack never refetches this within the tab session.
The request fires once on first mount; subsequent navigations back to `/explore` are free.

### Filtering and pagination (`$derived`)

```js
const defaultLimit = 20
let q = $state('')
let offset = $state(0)

// Reset pagination on new search
$effect(() => { q; offset = 0 })

const allSpecies = $derived(allSpeciesQuery.data ?? [])

const filtered = $derived(
  q.length < 2
    ? allSpecies
    : allSpecies.filter((s) => {
        const lq = q.toLowerCase()
        return s.common_name.toLowerCase().includes(lq)
            || s.scientific_name.toLowerCase().includes(lq)
      })
)

const displayed = $derived(filtered.slice(offset, offset + defaultLimit))
```

When `q` is empty, `filtered` is the full list and pagination applies normally.
When `q` has 2+ chars, `filtered` is the match set; pagination is still available
but usually short enough not to need it.

`PaginationBar` is always present in the template -- total is `filtered.length`.
The component already has `{#if totalPages > 1}` internally, so it renders nothing
when results fit on one page (e.g. 3 search matches with `limit=20`). No visible
behaviour change from the current explore page.

### Input change

The `oninput` handler simplifies to nothing: `q` is reactive state bound directly
to the input with `bind:value`. No debounce needed; `$derived` is synchronous.

## Testing

- Backend: unit test for `listAllSpecies` -- assert 200, correct shape, all species
  returned; assert 401 when unauthenticated.
- Frontend: existing explore page tests should continue to pass after the fetch mock
  is updated to target `/api/v1/species/all` instead of `/api/v1/species`.

## What this does NOT change

- `GET /api/v1/species` (paginated + search) stays as-is -- still used implicitly
  by any existing tests; could be deprecated later once explore is fully client-side.
- `SpeciesTypeahead` in group detail -- already client-side, no change.
- All other routes, middleware, auth behaviour.
