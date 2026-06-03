# Desktop Layout Redesign

**Date:** 2026-06-03  
**Status:** Approved

## Problem

The global `.app-container { max-width: 480px }` bottlenecks every page to phone width. On desktop the app renders as a narrow column in the center of the screen with large empty margins. The dashboard in particular feels sparse -- one stat tile and a flat list of decks.

## Goals

- Make the site feel like a website on desktop, not a phone app in a browser
- Richer dashboard stats with a live countdown to next due card
- Mobile remains fully functional; nav header cleaned up

## Out of scope

- Structural changes to Decks list, Deck detail, Explore, or Admin pages (they just gain width)
- "Studied today" stat (requires separate backend work, deferred)
- Region filter, CI/CD, any other roadmap items

---

## Design

### 1. Container strategy

Remove `max-width: 480px` from `.app-container` in `frontend/src/routes/+layout.svelte`. The layout wrapper becomes full-width. Each page adds its own centering wrapper:

| Pages | Max-width |
|---|---|
| Dashboard, Decks, Deck detail, Explore, Explore detail, Admin | 900px |
| Quiz, Practice | 560px |

The wrapper is a simple `<div class="page-content">` with `max-width` + `margin: 0 auto` in each page's `<style>`. No shared component needed -- the pattern is simple enough to repeat.

### 2. Nav header

**Desktop (≥640px):** single row, layout unchanged -- wordmark · nav links · theme toggle · Log out.

**Mobile (<640px):** two rows via CSS only (no JS):
- Row 1: wordmark (left) + theme toggle + Log out (right)
- Row 2: Decks · Explore · Admin (left-aligned, same muted style)

The current header DOM order is: wordmark → `<nav>` → `.header-actions`. Row 1 needs wordmark + actions together, row 2 needs nav. To achieve this cleanly, restructure the header HTML so `.header-actions` immediately follows the wordmark (moving `<nav>` last). On desktop, flex + `order` or natural source order keeps the visual left-center-right layout. On mobile, `flex-wrap: wrap` with explicit widths makes row 1 (wordmark + actions, `width: 100%` via justify-content space-between) and row 2 (`<nav>`, full width, left-aligned).

### 3. Dashboard stats bar

Four tiles in a single row (collapses to a 2×2 grid below 640px):

| Tile | Value | Source |
|---|---|---|
| Due today | `sum(audio_due + image_due)` across decks | Client-side, existing data |
| Audio due | `sum(audio_due)` across decks | Client-side, existing data |
| Image due | `sum(image_due)` across decks | Client-side, existing data |
| Next due in | Countdown or "Now" | `next_due_at` from backend (new) |

**"Next due in" behavior:**
- `total_due > 0`: shows `"Now"` in `var(--accent)` color
- `total_due === 0` and `next_due_at` is set: shows live countdown (`2h 14m`, `45m`, etc.) updated every 30 seconds via `setInterval`; clears on component destroy
- `next_due_at` is null (no cards at all): tile shows `--`

Countdown format: `Xh Ym` if ≥ 1 hour, `Ym` if < 1 hour, `<1m` if under a minute.

### 4. Backend: `next_due_at` field

Add `next_due_at` (nullable RFC3339 timestamp string) to the `GET /api/v1/decks` response. This is the minimum `due` timestamp across all cards for the current user where `due > NOW()`.

SQL (added to the existing decks query or as a separate scalar subquery):

```sql
SELECT MIN(due)
FROM cards
WHERE user_id = @user_id
  AND due > NOW()
```

Return as a top-level field alongside the decks array:

```json
{
  "decks": [...],
  "next_due_at": "2026-06-03T18:42:00Z"
}
```

This requires updating the API handler, sqlc query, and the frontend type for the decks response.

### 5. Dashboard deck grid

Decks rendered as a **2-column CSS grid** (1 column below 640px). Each card:
- Deck name (bold)
- Study buttons for due lanes only (`audio_due > 0` → Audio button, `image_due > 0` → Image button)
- "All done" label when both are 0

The existing `DeckList` component in `frontend/src/components/DeckList.svelte` is updated in-place (it's already used only on the dashboard). The `/decks` management page has its own separate list and is unaffected.

### 6. Mobile layout

| Breakpoint | Stats bar | Deck grid | Container padding |
|---|---|---|---|
| ≥640px | 4-column row | 2-column grid | 1.5rem |
| <640px | 2×2 grid | 1-column | 1rem |

---

## Files changed

**Backend:**
- `backend/internal/store/queries/cards.sql` -- new `GetNextDueAt` query
- `backend/internal/store/` -- regenerate via `just generate`
- `backend/internal/api/decks.go` -- add `next_due_at` to response struct and handler

**Frontend:**
- `frontend/src/routes/+layout.svelte` -- remove global max-width, add two-row mobile nav
- `frontend/src/routes/+page.svelte` -- add page-content wrapper, handle new response shape, pass `next_due_at` to StatsBar
- `frontend/src/routes/decks/+page.svelte` -- update to handle new `{ decks, next_due_at }` response shape (currently assigns `await res.json()` directly to `decks`)
- `frontend/src/components/StatsBar.svelte` -- 4-tile layout, countdown logic
- `frontend/src/components/DeckList.svelte` -- 2-column grid layout
- `frontend/src/types.ts` -- update decks response type

**No new routes or components needed.**
