# Species Lane Preferences -- Design

**Date:** 2026-06-01

## Summary

Two changes to the group detail page:

1. **Persistent search** -- after adding a species to a group, search results stay visible so the user can continue selecting from the same result set.
2. **Audio/Image lane toggles** -- each species row in the group detail gets two small toggle buttons (Audio, Image). Toggling updates the user's per-species preference globally, which creates or deletes the corresponding FSRS card.

## Scope decision: global vs. per-group preferences

Preferences are **per-user-per-species** (global), not per-group. Rationale:

- The toggle represents knowledge state: "I've mastered this bird's call -- disable audio for it everywhere."
- The "audio-only group" use case is already served by creating two separate groups with the same species.
- Per-group preferences would require a `(user_id, group_id, species_code)` junction table, more complex queries, and a more confusing UI.

## Backend

### New SQL query: `ListGroupSpeciesWithPrefs`

Replaces `ListGroupSpecies`. LEFT JOINs `user_species_preferences` on `(species_code, user_id)` and COALESCEs booleans to `true`, preserving the default-both-enabled behavior for species with no preference row.

```sql
-- name: ListGroupSpeciesWithPrefs :many
SELECT s.ebird_code, s.common_name, s.scientific_name,
       COALESCE(p.audio_enabled, true) AS audio_enabled,
       COALESCE(p.image_enabled, true) AS image_enabled
FROM species s
JOIN group_species gs ON gs.species_code = s.ebird_code
LEFT JOIN user_species_preferences p
       ON p.species_code = s.ebird_code AND p.user_id = $2
WHERE gs.group_id = $1
ORDER BY s.common_name;
```

Parameters: `(group_id int64, user_id int64)`.

The `listGroupSpecies` handler already extracts `userID` (for `groupOwnerCheck`), so passing it to the new query requires no structural change.

Run `just generate` after adding the query to regenerate sqlc types. Remove the old `ListGroupSpecies` query and its generated code.

No new HTTP endpoints needed.

## Frontend: group detail page (`/groups/[id]/+page.svelte`)

### Search UX fix

In `addSpecies()`, remove the two lines that clear `searchQuery` and `searchResults`. The search input stays populated and results stay visible after adding a species.

Add a derived set of already-added ebird codes. In the search results list, replace the "Add" button with a dimmed "Added" indicator when the species is already a member.

### Preference toggles

Update the `Species` interface:

```ts
interface Species {
  ebird_code: string
  common_name: string
  scientific_name: string
  audio_enabled: boolean
  image_enabled: boolean
}
```

Each species row gains two small toggle buttons alongside the existing "Remove" button. Active state = accent color, inactive = muted/outlined.

Toggle handler:
- Optimistically flip the value in local `groupSpecies` state.
- Fire `PUT /api/v1/species/:ebird_code/preferences` with `{ audio_enabled, image_enabled }` (both values always sent).
- On success, update from the response to confirm state.
- On error, revert to the previous value.

## Testing

- **Backend:** new test for `listGroupSpecies` handler covering the case where preference rows exist and where they don't (COALESCE default).
- **Frontend:** update existing group detail page tests to include `audio_enabled`/`image_enabled` in mock species data; add tests for the toggle interaction (fires PUT, updates state, reverts on error) and for the persistent-search behavior (results survive after add).
