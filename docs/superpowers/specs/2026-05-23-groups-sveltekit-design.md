# Groups + SvelteKit Migration Design

## Goal

Migrate the frontend from plain Vite to SvelteKit (SPA mode), wire the dashboard to real API data, and build group management -- creating/editing/deleting groups and adding/removing species -- so users can build their own practice lists and run real quiz sessions.

## Architecture

**SvelteKit in SPA mode.** No SSR -- all routing is client-side. Configured with `adapter-static` and a catch-all fallback so the Go server can serve the built files and let SvelteKit handle routes. Existing components (`QuizCard`, `RevealCard`, `ImageQuizCard`, `StatsBar`, `GroupList`) are already prop-driven and port over unchanged.

Auth stays client-side: root `+layout.svelte` checks `GET /api/v1/me` on load. Unauthenticated users see the Login component (not a route -- just a conditional render in the layout). The existing JWT/cookie auth is unchanged.

The `view` and `session` stores are retired -- replaced by SvelteKit's `goto()` and URL params.

## Route Structure

```
/                        Dashboard (real group data, due counts)
/groups                  Groups list (create, rename, delete)
/groups/[id]             Group detail (species list, add/remove, practice buttons)
/groups/[id]/quiz        Quiz session (?lane=audio|image)
/explore                 Stub ("Explore is coming soon")
```

## Backend API

Eight new endpoints, all behind `RequireAuth`. All group write operations verify `owner_id = user_id` before proceeding.

| Method   | Path                                   | Description                                              |
|----------|----------------------------------------|----------------------------------------------------------|
| GET      | /api/v1/groups                         | List user's groups with audio_due + image_due counts     |
| POST     | /api/v1/groups                         | Create group `{name}`                                    |
| PATCH    | /api/v1/groups/:id                     | Rename group `{name}`                                    |
| DELETE   | /api/v1/groups/:id                     | Delete group (cascades to group_species, leaves cards)   |
| GET      | /api/v1/groups/:id/species             | List species in a group                                  |
| POST     | /api/v1/groups/:id/species             | Add species `{species_id}`, auto-creates cards both lanes|
| DELETE   | /api/v1/groups/:id/species/:species_id | Remove species from group (leaves cards intact)          |
| GET      | /api/v1/species?q=                     | Search species by common/scientific name, top 20         |

**Due count query:** Single SQL query using `COUNT(CASE WHEN lane='audio' AND due <= NOW() THEN 1 END)` per group, joining `cards` filtered by `user_id`.

**Add species to group:** Inserts into `group_species`, then upserts cards for both `audio` and `image` lanes (idempotent -- `ON CONFLICT DO NOTHING`). Cards are not deleted when a species is removed from a group; they become dormant (not surfaced by any group's quiz session).

**Authorization check:** Before any mutating group operation, query `SELECT owner_id FROM groups WHERE id = $1` and 404 if not found, 403 if `owner_id != user_id`.

## SQL Queries (no schema changes)

New queries to add to `backend/internal/store/queries/`:

**groups.sql:**
- `ListUserGroups` `:many` -- groups owned by user with `audio_due`, `image_due` counts
- `CreateGroup` `:one` -- insert, return full row
- `UpdateGroupName` `:one` -- update name, return full row
- `DeleteGroup` `:exec` -- delete by id + owner_id (safe combined check)
- `GetGroupOwner` `:one` -- returns `owner_id` for auth check
- `ListGroupSpecies` `:many` -- species in a group with id, common_name, scientific_name, ebird_code
- `AddSpeciesToGroup` `:exec` -- insert into group_species, ON CONFLICT DO NOTHING
- `RemoveSpeciesFromGroup` `:exec` -- delete from group_species

**species.sql:**
- `SearchSpecies` `:many` -- `ILIKE '%'||$1||'%'` on common_name and scientific_name, ORDER BY common_name, LIMIT 20

## Frontend Pages

**`+layout.svelte` (root):** Checks `/api/v1/me` on mount. If unauthenticated, renders `<Login />`. If authenticated, renders the app shell (header with wordmark + theme toggle + nav links) and `<slot />`. Handles the auth cookie check that `App.svelte` currently owns.

**`/` -- Dashboard:** Fetches `GET /api/v1/groups`. Shows `<GroupList>` with real data. Empty state: "No groups yet" with a link to `/groups`. `<StatsBar>` shows total due across all groups.

**`/groups` -- Groups list:** Lists all user groups. Inline "New Group" form (name input + submit). Each group row: name, species count, edit (rename inline) and delete actions. Click name → `/groups/[id]`.

**`/groups/[id]` -- Group detail:** Shows group name, "Practice Audio" / "Practice Image" buttons (navigate to `/groups/[id]/quiz?lane=audio|image`). Species list with remove button per species. Search bar below: live search via `GET /api/v1/species?q=` (debounced, 300ms), results show common name + scientific name + Add button.

**`/groups/[id]/quiz` -- Quiz:** Current `Quiz.svelte` logic, now reads `groupId` from route params and `lane` from `?lane=` query param. On finish → `goto(\`/groups/${groupId}\`)`.

**`/explore` -- Stub:** Single centered message: "Explore is coming soon."

## Auth

Unchanged. `GET /api/v1/me` is the auth check. JWT in HttpOnly cookie. `is_admin` claim available for future admin panel. No token rotation for now.

## Testing

**Backend:** Same stub-querier pattern as existing quiz/preferences tests. New handler file `backend/internal/api/groups.go` with tests in `groups_test.go`. Cover: list groups, create group, auth check (wrong owner returns 403), add species (verifies card upsert calls), search species.

**Frontend:** Existing component tests (`QuizCard`, `RevealCard`, `GroupList`, `StatsBar`) port unchanged. New tests for Dashboard (mocked fetch), Groups list page (create/delete), Group detail (search + add species). Quiz test updated to read lane from URL param instead of session store.
