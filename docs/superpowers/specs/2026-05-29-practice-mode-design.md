# Practice Mode Design

**Date:** 2026-05-29
**Status:** Approved

## Overview

Add a free practice mode that lets users drill all species in a group regardless of FSRS schedule. Answers don't affect spaced repetition state. Useful for drilling before a birdwatching trip or learning new species before they've been scheduled.

## Backend

### New SQL query

Add to `backend/internal/store/queries/cards.sql`:

```sql
-- name: GetGroupPracticeCards :many
SELECT s.ebird_code, s.common_name, s.scientific_name,
       (SELECT file_path FROM species_recordings
        WHERE species_code = s.ebird_code AND quality IN ('A', 'B')
        ORDER BY random() LIMIT 1) AS audio_url,
       (SELECT file_path FROM species_images
        WHERE species_code = s.ebird_code
        ORDER BY random() LIMIT 1) AS image_url
FROM species s
JOIN group_species gs ON gs.species_code = s.ebird_code
WHERE gs.group_id = $1;
```

Returns all species in the group with one randomly-selected audio URL and one randomly-selected image URL per species. Both may be NULL if no media exists for a species.

### New handler

Add `getPracticeCards` to `backend/internal/api/quiz.go`:

- Validates `lane` query param (`audio` or `image`)
- Calls `GetGroupPracticeCards`
- Maps each row to `nextCardResponse` (same shape as quiz endpoint):
  - Audio lane: `media_url = audio_url`, `photo_url = image_url`
  - Image lane: `media_url = image_url`, `photo_url = image_url`
  - `lane` field set to the requested lane
- The correlated subqueries return NULL (not an error) when no media exists; sqlc generates nullable URL fields. The handler filters out any row where the relevant URL is NULL before building the response.
- Returns `200` with `[]nextCardResponse` (may be empty if no species have media)

### New route

Add to auth group in `backend/internal/api/router.go`:

```
GET /groups/{id}/practice
```

Run `just generate` after adding the SQL query to regenerate sqlc types.

## Frontend

### New practice page

`frontend/src/routes/groups/[id]/practice/+page.svelte`

**Data loading:**
- On mount, fetch `GET /api/v1/groups/:id/practice?lane=` once
- Fisher-Yates shuffle the practice card array client-side
- Derive the typeahead species list from the practice cards (`cards.map(c => ({ ebird_code, common_name, scientific_name }))`) -- no separate species fetch needed
- All subsequent card navigation is index-driven -- no per-card API calls

**Quiz loop:**
- Reuses `QuizCard`, `ImageQuizCard`, and `RevealCard` components unchanged
- `onReveal`: same logic as quiz (compare `selected.ebird_code === card.ebird_code`)
- `onNext`: increments index only -- no POST to `/rate`
- `{#key card.ebird_code}` wrapper to reset component state between cards

**StatsBar:** shows `Practiced: X / Y` (current index + 1 and total count)

**Done screen** (when index reaches end of shuffled array):
- "Practice Again" button: reshuffles the array and resets index to 0
- "Back to Group" button: navigates to `/groups/:id`
- No "Come back later" copy (FSRS scheduling not involved)

**Error states:**
- If the practice endpoint returns an empty array (group has no species with media): show "No species with media in this group." and a back button
- Network errors: show retry button

### Groups list page changes

`frontend/src/routes/groups/+page.svelte`

Add `let practiceMode = $state(false)` (resets on navigation, no localStorage).

Add a "Free Practice" toggle button in the page header alongside the "Groups" h1.

When `practiceMode` is true:
- Render a banner below the header: "Free practice mode -- answers won't affect your spaced repetition schedule"
- Each group row replaces its due badges with two small buttons:
  - "▶ Audio" → `/groups/:id/practice?lane=audio`
  - "◉ Image" → `/groups/:id/practice?lane=image`
- Delete button remains

When `practiceMode` is false, page is unchanged from current state.

### Group detail page changes

`frontend/src/routes/groups/[id]/+page.svelte`

**Rename existing buttons** (FSRS quiz -- scheduled cards only):
- "Practice Audio" → "Study Audio"
- "Practice Image" → "Study Image"

**Add new practice buttons** in a second row below:
- "Practice Audio" → `/groups/:id/practice?lane=audio`
- "Practice Image" → `/groups/:id/practice?lane=image`

Study buttons: filled accent style (current `.btn-practice` style).
Practice buttons: outline style (accent border, transparent background, accent text) to visually distinguish the two modes.

## What's not in scope

- Preserving practice session state across navigation (session is lost on back/forward)
- Per-species skip during practice
- Tracking practice history or scores
- Any FSRS rating during practice
