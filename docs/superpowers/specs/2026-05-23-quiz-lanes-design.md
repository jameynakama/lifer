# Quiz Lanes Redesign

**Date:** 2026-05-23
**Status:** approved

## Overview

Replace the current `user × recording` card model with a two-lane system: one FSRS card per user per species per lane (`audio` or `image`). Each audio review presents a randomly selected recording from that species' pool so users build general confidence rather than memorising a single clip. Each image review presents a randomly selected photo. Lanes are independently scheduled and can be toggled per species per user.

## Goals

- Launch with both audio and image identification from day one
- FSRS scheduling is global per `user × species × lane` -- progress carries across groups
- Quiz sessions are focused: user picks a lane before starting, not a mixed stream
- Schema and frontend components are SvelteKit-ready (no routing coupling)

## Schema

### Migration strategy

Squash all existing migrations (001--003) into a single `001_initial.up.sql` reflecting the final schema. No production data exists, so there is no migration cost.

### cards table (replaces current `user × recording_id` cards)

```sql
CREATE TABLE cards (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    species_id  BIGINT      NOT NULL REFERENCES species(id) ON DELETE CASCADE,
    lane        TEXT        NOT NULL CHECK (lane IN ('audio', 'image')),
    stability   FLOAT       NOT NULL DEFAULT 0,
    difficulty  FLOAT       NOT NULL DEFAULT 0,
    due         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_review TIMESTAMPTZ,
    reps        INT         NOT NULL DEFAULT 0,
    lapses      INT         NOT NULL DEFAULT 0,
    state       SMALLINT    NOT NULL DEFAULT 0, -- 0=new 1=learning 2=review 3=relearning
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, species_id, lane)
);
CREATE INDEX idx_cards_user_lane_due ON cards(user_id, lane, due);
```

`lane` uses a text CHECK constraint rather than a Postgres enum -- enums require a migration to extend.

### user_species_preferences table

```sql
CREATE TABLE user_species_preferences (
    user_id       BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    species_id    BIGINT      NOT NULL REFERENCES species(id) ON DELETE CASCADE,
    audio_enabled BOOLEAN     NOT NULL DEFAULT TRUE,
    image_enabled BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, species_id)
);
```

Both lanes default to enabled when a preference row is created (i.e. when the user first adds a species to any group). Disabling a lane deletes the corresponding card row immediately -- no soft-delete or active flag.

### Card lifecycle

- A preference row is created (with defaults) when a user adds a species to a group.
- Enabling a lane upserts a card row with `due = NOW()`, `state = 0` (new).
- Disabling a lane deletes the card row.
- Cards are global: the same row is used regardless of which group surfaces the species.

## API Endpoints

All endpoints require authentication. User ID is read from the JWT context -- never from the request body.

### GET /api/v1/groups/:id/next?lane=audio|image

Returns the next due card for the authenticated user in the given group and lane.

- Joins `cards → species → group_species` to scope results to the group.
- Picks the card with the earliest `due` that is `<= NOW()`.
- Selects a random recording (`ORDER BY random() LIMIT 1`) from `species_recordings` for the audio lane, or a random image from `species_images` for the image lane.
- Returns 204 when no cards are due.

Response body (200):
```json
{
  "species_id": 42,
  "common_name": "Spotted Towhee",
  "scientific_name": "Pipilo maculatus",
  "media_url": "https://xeno-canto.org/123456/download",
  "photo_url": "https://cdn.download.ams.birds.cornell.edu/api/v1/asset/12345/large",
  "lane": "audio"
}
```

`media_url` is the primary quiz content (audio URL for audio lane, photo URL for image lane). `photo_url` is always a random species photo -- used by `RevealCard` on the audio lane reveal; for the image lane `media_url` and `photo_url` will be the same URL.

### POST /api/v1/groups/:id/rate

Records a rating for a card and updates its FSRS schedule.

Request body:
```json
{ "species_id": 42, "lane": "audio", "rating": 3 }
```

- Rating is 1--4 (Again / Hard / Good / Easy).
- Uses `github.com/open-spaced-repetition/go-fsrs` for scheduling -- do not roll a custom implementation.
- Returns the updated card row.
- Returns 404 if no card exists for `(user_id, species_id, lane)`.

### PUT /api/v1/species/:id/preferences

Upserts the authenticated user's lane preferences for a species.

Request body:
```json
{ "audio_enabled": true, "image_enabled": false }
```

- Upserts `user_species_preferences`.
- For each lane: enabling creates the card row if absent; disabling deletes it.
- Returns the updated preference row.

## Frontend

### Routing / props contract

`Quiz.svelte` accepts `groupId: number` and `lane: 'audio' | 'image'` as **props**. It does not read from the global `view` store for these values. This keeps the component SvelteKit-ready: today the store-based router passes the props; later SvelteKit passes route params as props. No component changes required at migration time.

### Quiz.svelte

- On mount, fetch `GET /api/v1/groups/:groupId/next?lane={lane}`.
- After each rating, POST to `POST /api/v1/groups/:groupId/rate`, then fetch the next card.
- On 204 response, navigate to dashboard with an "all done" state.
- Loading and error states handled locally.

### QuizCard.svelte (audio lane -- no changes)

Receives a real `media_url` instead of a mock path. No structural changes needed.

### ImageQuizCard.svelte (new)

Same structure as `QuizCard.svelte`: guess input + reveal button. Renders `<img src={card.media_url}>` instead of `<audio>`. Reuses the same CSS tokens.

### RevealCard.svelte (no changes)

Works for both lanes. Audio lane: shows photo + species name + rating buttons (current design). Image lane: user was already looking at a photo, so showing photo + name as confirmation is valid.

### BirdCard type

```ts
type BirdCard = {
  species_id: number
  common_name: string
  scientific_name: string
  media_url: string    // audio URL (audio lane) or photo URL (image lane)
  photo_url: string    // always a species photo; used by RevealCard for audio lane reveal
  lane: 'audio' | 'image'
}
```

`photo_path` and `recording_path` are removed in favour of `media_url` + `photo_url`.

## FSRS

Use `github.com/open-spaced-repetition/go-fsrs`. The `POST /rate` handler:
1. Loads the current card state.
2. Calls the FSRS scheduler with the rating.
3. Writes the updated stability, difficulty, due, reps, lapses, state back to the card row.

## Out of scope for this spec

- Explore / Learn view and species detail pages
- Lists UI (adding species to groups from browse views)
- Admin catalog management
- SvelteKit migration (intentionally next after this)
- Group management UI
