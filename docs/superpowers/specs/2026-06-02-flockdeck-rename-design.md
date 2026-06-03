# FlockDeck Rename Design

**Date:** 2026-06-02
**Status:** Approved

## Summary

Rename the project from "lifer" to "FlockDeck" and rename the "groups" concept to "decks" throughout the entire stack. Single atomic commit. Migration squashed into `001_initial`.

## Scope

### Database (001_initial.up.sql + down)

- `groups` table → `decks`
- `group_species` join table → `deck_species`; FK column `group_id` → `deck_id`
- Drop `is_preset` column from `decks` (was dead scaffolding, never populated)
- Down migration updated to match

### Go backend

- `go.mod` module path: `github.com/jameynakama/lifer` → `github.com/jameynakama/flockdeck`
- All import paths across every `.go` file updated
- `internal/store/queries/groups.sql` → `decks.sql`; all SQL references to `groups`/`group_species` updated
- `sqlc.yaml` updated if it references query files by name
- Regenerated sqlc output: `groups.sql.go` → `decks.sql.go`
- Struct renames: `Group` → `Deck`, `GroupSpecies` → `DeckSpecies`, `GroupWithDue` → `DeckWithDue`
- All generated method names updated accordingly
- `models.go`: `Deck` struct drops `IsPreset` field
- `internal/api/groups.go` → `decks.go`; all handler functions renamed (`listGroups` → `listDecks`, etc.)
- Router: all `/api/v1/groups/` paths → `/api/v1/decks/`

### Frontend

- `src/routes/groups/` → `src/routes/decks/` (entire directory tree with all pages and tests)
- `GroupList.svelte` → `DeckList.svelte`
- `GroupDropdown.svelte` → `DeckDropdown.svelte`
- All component imports updated at usage sites
- `src/types.ts`: `Group` interface → `Deck`; drop `is_preset` field
- All variable/function names referencing `group`/`groups` → `deck`/`decks` throughout
- `+layout.svelte`: "Lifer" wordmark → "FlockDeck"
- `Login.svelte`: "Lifer" heading → "FlockDeck"
- `src/lib/theme.ts`: storage key `lifer-theme` → `flockdeck-theme`
- All tests updated (strings, mock data shapes, route paths)

## What does NOT change

- eBird codes, species data, recording/image data -- untouched
- FSRS logic, quiz/practice flow -- untouched
- `user_species_preferences` table -- untouched
- `cards` table -- untouched
- The `owner_id` nullable pattern (null = system-owned) remains valid even without `is_preset`

## Post-rename steps (manual, outside code)

1. GitHub repo rename: `gh repo rename flockdeck`
2. Update local remote: `jj git remote set-url origin https://github.com/jameynakama/flockdeck`
3. Update any deploy config that references the old repo name
