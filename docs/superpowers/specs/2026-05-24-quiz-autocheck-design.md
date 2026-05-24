# Quiz Auto-Check with Typeahead Design

## Goal

Replace the inert free-text guess input in the quiz with a typeahead that filters the group's species list client-side, auto-checks the answer on reveal, and pre-suggests an FSRS rating based on correctness -- while keeping manual rating overrides available.

## Architecture

All changes are frontend-only. No new API endpoints or schema changes are needed. The group species list is fetched once on quiz mount and passed down as a prop; all typeahead filtering is client-side with zero additional requests per keystroke.

The quiz page owns the answer-check state (`correct`, `guessed`). The `SpeciesTypeahead` component owns filtering and selection UI. `QuizCard`/`ImageQuizCard` wire them together. `RevealCard` receives the result and surfaces the suggested rating.

## Components

### New: `SpeciesTypeahead.svelte`

**Props:** `species: Species[]`, `onSelect: (species: Species | null) => void`

**Behavior:**
- Filters `species` client-side on `common_name` and `scientific_name` (case-insensitive substring) when input has ≥2 characters
- Dropdown shows up to 10 results; typing more characters narrows further
- Keyboard nav: arrow up/down moves highlight, Enter selects highlighted item, Escape closes dropdown without selecting
- Clicking a result selects it: input fills with `common_name`, dropdown closes, `onSelect(species)` fires
- Clearing input below 2 chars resets selection and fires `onSelect(null)`

### Modified: `QuizCard.svelte` and `ImageQuizCard.svelte`

- Replace free-text `<input>` with `<SpeciesTypeahead {species} onSelect={handleSelect} />`
- New prop: `species: Species[]`
- `onReveal` signature changes to `onReveal(selected: Species | null) => void`
- Reveal button is **disabled** until a species is selected (enabled when `selected !== null`)
- **"I don't know" button** sits alongside Reveal, always enabled; clicking it calls `onReveal(null)` directly, bypassing the species selection requirement

### Modified: `RevealCard.svelte`

- New props: `correct: boolean`, `guessed: Species | null`
- Result banner above the photo:
  - Correct: "✓ [common name]" in green
  - Wrong with a guess: "✗ You guessed: [guessed name]" in red
  - Wrong, no guess: "✗ You didn't know" in red
  - Followed by correct species name if wrong
- Suggested rating highlighted with a CSS outline ring: **Good** (3) if correct, **Again** (1) if incorrect
- All four rating buttons (Again / Hard / Good / Easy) remain clickable -- the highlight is a suggestion only

### Modified: `quiz/+page.svelte`

- Fetch group species list once on mount via `GET /api/v1/groups/${groupId}/species`, store as `groupSpecies: Species[]`
- Pass `species={groupSpecies}` to QuizCard and ImageQuizCard
- `onReveal(selected: Species | null)`:
  - Computes `correct = selected !== null && selected.id === card.species_id`
  - Stores `guessed = selected`
  - Sets `revealed = true`
- `onRate(rating: number)`: unchanged -- POSTs to `/api/v1/groups/${groupId}/rate`, fetches next card, clears `guessed`

## Data Flow

```
Quiz page mounts
  → fetch /api/v1/groups/{id}/species  (once)
  → fetch /api/v1/groups/{id}/next?lane={lane}  (once, then after each rating)

User types in SpeciesTypeahead
  → client-side filter (no requests)
  → selects species or clicks "I don't know"

onReveal(selected) fires in quiz page
  → correct = selected?.id === card.species_id
  → revealed = true, guessed = selected

RevealCard shown with correct/guessed/onRate
  → user clicks rating (or suggested rating)
  → POST /api/v1/groups/{id}/rate
  → fetch next card, clear guessed
```

## FSRS Rating Mapping

| Outcome | Suggested button | User can override to |
|---------|-----------------|----------------------|
| Correct | Good (3) | Hard (2) or Easy (4) |
| Incorrect / didn't know | Again (1) | Hard (2), Good (3), Easy (4) |

## Types

`Species` is currently defined locally inside the quiz route. Since it's now used across `SpeciesTypeahead`, `QuizCard`, `ImageQuizCard`, `RevealCard`, and the quiz page, it moves to `src/types.ts`:

```typescript
export interface Species {
  id: number
  common_name: string
  scientific_name: string
  ebird_code: string
}
```

## Testing

- `SpeciesTypeahead`: filters correctly at ≥2 chars, keyboard nav, "I don't know" fires onSelect(null), clears on backspace
- `QuizCard` / `ImageQuizCard`: Reveal disabled until selection, enabled after selection
- `RevealCard`: correct banner + Good highlight when correct; wrong banner + Again highlight when incorrect; all buttons still clickable
- `quiz/+page.svelte`: group species fetched on mount, correct/guessed state passed to RevealCard, cleared after rating

## Out of Scope

- Failure queue (repeat until correct within session) -- defer
- "Hard" auto-assignment for slow-but-correct answers -- defer
- Explore page search -- uses server-side search, unaffected by this change
