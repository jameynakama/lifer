# Quiz Auto-Check with Typeahead Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the inert free-text guess input in the quiz with a typeahead that filters the group's species list client-side, auto-checks the answer on reveal, and pre-suggests an FSRS rating (Good if correct, Again if incorrect).

**Architecture:** All changes are frontend-only. A new `SpeciesTypeahead` component handles filtering/selection. `QuizCard` and `ImageQuizCard` embed it and add an "I don't know" escape. `RevealCard` gains a result banner and a highlighted suggested rating. The quiz page fetches group species once on mount and owns the `correct`/`guessed` state.

**Tech Stack:** Svelte 5 (runes), TypeScript, vitest + @testing-library/svelte, jj for VCS

---

## File Map

**Modified:**
- `frontend/src/types.ts` -- add `Species` interface
- `frontend/src/components/QuizCard.svelte` -- embed typeahead, update onReveal signature, add "I don't know"
- `frontend/src/components/QuizCard.test.ts` -- update for new props/behavior
- `frontend/src/components/ImageQuizCard.svelte` -- same changes as QuizCard
- `frontend/src/components/RevealCard.svelte` -- add correct/guessed props, result banner, suggested rating highlight
- `frontend/src/components/RevealCard.test.ts` -- update for new props
- `frontend/src/routes/groups/[id]/quiz/+page.svelte` -- fetch group species, add guessed/correct state, update onReveal
- `frontend/src/routes/groups/[id]/quiz/page.test.ts` -- update mocks, add new tests

**Created:**
- `frontend/src/components/SpeciesTypeahead.svelte` -- typeahead input + dropdown
- `frontend/src/components/SpeciesTypeahead.test.ts` -- typeahead tests
- `frontend/src/components/ImageQuizCard.test.ts` -- tests (none exist yet)

---

### Task 1: Add `Species` type to `types.ts`

**Files:**
- Modify: `frontend/src/types.ts`

- [ ] **Step 1: Add `Species` interface**

Open `frontend/src/types.ts`. It currently contains `Stat`, `BirdCard`, and `Group`. Add `Species` at the end:

```typescript
export interface Stat {
    label: string;
    value: string | number;
}

export interface BirdCard {
    species_id: number;
    common_name: string;
    scientific_name: string;
    media_url: string;
    photo_url: string;
    lane: 'audio' | 'image';
}

export interface Group {
    id: number;
    name: string;
    is_preset: boolean;
    audio_due: number;
    image_due: number;
}

export interface Species {
    id: number;
    common_name: string;
    scientific_name: string;
    ebird_code: string;
}
```

- [ ] **Step 2: Verify TypeScript is happy**

```bash
cd frontend && npm run check 2>&1 | tail -5
```

Expected: no errors (or same pre-existing errors as before).

- [ ] **Step 3: Commit**

```bash
jj describe -m "feat: add Species type to types.ts"
jj new
```

---

### Task 2: `SpeciesTypeahead` component

**Files:**
- Create: `frontend/src/components/SpeciesTypeahead.svelte`
- Create: `frontend/src/components/SpeciesTypeahead.test.ts`

- [ ] **Step 1: Write failing tests**

Create `frontend/src/components/SpeciesTypeahead.test.ts`:

```typescript
import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/svelte'
import SpeciesTypeahead from './SpeciesTypeahead.svelte'
import type { Species } from '../types'

const species: Species[] = [
  { id: 1, common_name: 'Song Sparrow', scientific_name: 'Melospiza melodia', ebird_code: 'sonspa' },
  { id: 2, common_name: 'Fox Sparrow', scientific_name: 'Passerella iliaca', ebird_code: 'foxspa' },
  { id: 3, common_name: 'Dark-eyed Junco', scientific_name: 'Junco hyemalis', ebird_code: 'daejun' },
]

describe('SpeciesTypeahead', () => {
  it('renders a text input', () => {
    render(SpeciesTypeahead, { props: { species, onSelect: vi.fn() } })
    expect(screen.getByRole('textbox')).toBeInTheDocument()
  })

  it('does not show dropdown for input shorter than 2 chars', async () => {
    render(SpeciesTypeahead, { props: { species, onSelect: vi.fn() } })
    await fireEvent.input(screen.getByRole('textbox'), { target: { value: 's' } })
    expect(screen.queryByRole('listbox')).toBeNull()
  })

  it('shows filtered results for input of 2+ chars', async () => {
    render(SpeciesTypeahead, { props: { species, onSelect: vi.fn() } })
    await fireEvent.input(screen.getByRole('textbox'), { target: { value: 'sp' } })
    expect(screen.getByRole('listbox')).toBeInTheDocument()
    expect(screen.getByText('Song Sparrow')).toBeInTheDocument()
    expect(screen.getByText('Fox Sparrow')).toBeInTheDocument()
    expect(screen.queryByText('Dark-eyed Junco')).toBeNull()
  })

  it('matches on scientific name', async () => {
    render(SpeciesTypeahead, { props: { species, onSelect: vi.fn() } })
    await fireEvent.input(screen.getByRole('textbox'), { target: { value: 'mel' } })
    expect(screen.getByRole('listbox')).toBeInTheDocument()
    expect(screen.getByText('Song Sparrow')).toBeInTheDocument()
  })

  it('limits results to 10', async () => {
    const manySpecies: Species[] = Array.from({ length: 15 }, (_, i) => ({
      id: i + 1,
      common_name: `Sparrow ${i + 1}`,
      scientific_name: `Species ${i + 1}`,
      ebird_code: `sp${i + 1}`,
    }))
    render(SpeciesTypeahead, { props: { species: manySpecies, onSelect: vi.fn() } })
    await fireEvent.input(screen.getByRole('textbox'), { target: { value: 'sp' } })
    const items = screen.getAllByRole('option')
    expect(items.length).toBe(10)
  })

  it('calls onSelect with the species when a result is clicked', async () => {
    const onSelect = vi.fn()
    render(SpeciesTypeahead, { props: { species, onSelect } })
    await fireEvent.input(screen.getByRole('textbox'), { target: { value: 'song' } })
    const item = screen.getByRole('option', { name: /song sparrow/i })
    await fireEvent.mouseDown(item)
    expect(onSelect).toHaveBeenCalledWith(species[0])
  })

  it('fills the input with the species common name after selection', async () => {
    render(SpeciesTypeahead, { props: { species, onSelect: vi.fn() } })
    await fireEvent.input(screen.getByRole('textbox'), { target: { value: 'song' } })
    const item = screen.getByRole('option', { name: /song sparrow/i })
    await fireEvent.mouseDown(item)
    expect(screen.getByRole('textbox')).toHaveValue('Song Sparrow')
  })

  it('closes the dropdown after selection', async () => {
    render(SpeciesTypeahead, { props: { species, onSelect: vi.fn() } })
    await fireEvent.input(screen.getByRole('textbox'), { target: { value: 'song' } })
    const item = screen.getByRole('option', { name: /song sparrow/i })
    await fireEvent.mouseDown(item)
    expect(screen.queryByRole('listbox')).toBeNull()
  })

  it('calls onSelect(null) when input is cleared below 2 chars', async () => {
    const onSelect = vi.fn()
    render(SpeciesTypeahead, { props: { species, onSelect } })
    await fireEvent.input(screen.getByRole('textbox'), { target: { value: 'so' } })
    await fireEvent.input(screen.getByRole('textbox'), { target: { value: 's' } })
    expect(onSelect).toHaveBeenLastCalledWith(null)
  })

  it('selects the first highlighted item on Enter', async () => {
    const onSelect = vi.fn()
    render(SpeciesTypeahead, { props: { species, onSelect } })
    const input = screen.getByRole('textbox')
    await fireEvent.input(input, { target: { value: 'sp' } })
    await fireEvent.keyDown(input, { key: 'Enter' })
    expect(onSelect).toHaveBeenCalledWith(expect.objectContaining({ common_name: 'Song Sparrow' }))
  })

  it('closes the dropdown on Escape without selecting', async () => {
    const onSelect = vi.fn()
    render(SpeciesTypeahead, { props: { species, onSelect } })
    const input = screen.getByRole('textbox')
    await fireEvent.input(input, { target: { value: 'sp' } })
    expect(screen.getByRole('listbox')).toBeInTheDocument()
    await fireEvent.keyDown(input, { key: 'Escape' })
    expect(screen.queryByRole('listbox')).toBeNull()
    expect(onSelect).not.toHaveBeenCalled()
  })
})
```

- [ ] **Step 2: Run to confirm it fails**

```bash
cd frontend && npm test -- src/components/SpeciesTypeahead.test.ts 2>&1 | head -20
```

Expected: FAIL -- file not found.

- [ ] **Step 3: Create `SpeciesTypeahead.svelte`**

Create `frontend/src/components/SpeciesTypeahead.svelte`:

```svelte
<script lang="ts">
  import type { Species } from '../types'

  let { species, onSelect }: {
    species: Species[]
    onSelect: (s: Species | null) => void
  } = $props()

  let query = $state('')
  let highlighted = $state(0)
  let open = $state(false)

  const filtered = $derived(
    query.length < 2
      ? []
      : species
          .filter((s) => {
            const q = query.toLowerCase()
            return (
              s.common_name.toLowerCase().includes(q) ||
              s.scientific_name.toLowerCase().includes(q)
            )
          })
          .slice(0, 10)
  )

  function handleInput() {
    highlighted = 0
    open = query.length >= 2
    if (query.length < 2) {
      onSelect(null)
    }
  }

  function selectSpecies(s: Species) {
    query = s.common_name
    open = false
    highlighted = 0
    onSelect(s)
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      highlighted = Math.min(highlighted + 1, filtered.length - 1)
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      highlighted = Math.max(highlighted - 1, 0)
    } else if (e.key === 'Enter') {
      e.preventDefault()
      if (open && filtered[highlighted]) selectSpecies(filtered[highlighted])
    } else if (e.key === 'Escape') {
      open = false
    }
  }
</script>

<div class="typeahead">
  <input
    type="text"
    bind:value={query}
    oninput={handleInput}
    onkeydown={handleKeydown}
    placeholder="Type species name..."
    class="typeahead-input"
    autocomplete="off"
  />
  {#if open && filtered.length > 0}
    <ul class="dropdown" role="listbox">
      {#each filtered as s, i (s.id)}
        <li
          class="dropdown-item"
          class:highlighted={i === highlighted}
          role="option"
          aria-selected={i === highlighted}
          aria-label="{s.common_name} {s.scientific_name}"
          onmousedown={() => selectSpecies(s)}
        >
          <span class="common">{s.common_name}</span>
          <span class="scientific">{s.scientific_name}</span>
        </li>
      {/each}
    </ul>
  {/if}
</div>

<style>
  .typeahead {
    position: relative;
  }
  .typeahead-input {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: 8px;
    padding: 0.6875rem 0.875rem;
    width: 100%;
    font-size: 0.875rem;
    color: var(--text);
    font-family: inherit;
    outline: none;
    box-sizing: border-box;
  }
  .typeahead-input:focus {
    border-color: var(--accent);
  }
  .dropdown {
    position: absolute;
    top: calc(100% + 4px);
    left: 0;
    right: 0;
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: 8px;
    list-style: none;
    margin: 0;
    padding: 0.25rem 0;
    z-index: 10;
    box-shadow: var(--shadow);
    max-height: 280px;
    overflow-y: auto;
  }
  .dropdown-item {
    display: flex;
    flex-direction: column;
    gap: 0.125rem;
    padding: 0.5rem 0.875rem;
    cursor: pointer;
  }
  .dropdown-item.highlighted,
  .dropdown-item:hover {
    background: var(--accent);
  }
  .dropdown-item.highlighted .common,
  .dropdown-item.highlighted .scientific,
  .dropdown-item:hover .common,
  .dropdown-item:hover .scientific {
    color: #fff;
  }
  .common {
    font-size: 0.875rem;
    font-weight: 600;
    color: var(--text);
  }
  .scientific {
    font-size: 0.75rem;
    color: var(--text-muted);
    font-style: italic;
  }
</style>
```

- [ ] **Step 4: Run tests**

```bash
cd frontend && npm test -- src/components/SpeciesTypeahead.test.ts
```

Expected: all 10 tests pass.

- [ ] **Step 5: Commit**

```bash
jj describe -m "feat: SpeciesTypeahead component with client-side filtering"
jj new
```

---

### Task 3: Update `QuizCard`

**Files:**
- Modify: `frontend/src/components/QuizCard.svelte`
- Modify: `frontend/src/components/QuizCard.test.ts`

- [ ] **Step 1: Update `QuizCard.test.ts` with new tests**

Replace the entire contents of `frontend/src/components/QuizCard.test.ts`:

```typescript
import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/svelte'
import QuizCard from './QuizCard.svelte'
import type { BirdCard, Species } from '../types'

const card: BirdCard = {
  species_id: 1,
  common_name: 'Song Sparrow',
  scientific_name: 'Melospiza melodia',
  media_url: '/recordings/song-sparrow.mp3',
  photo_url: '/photos/song-sparrow.jpg',
  lane: 'audio' as const,
}

const species: Species[] = [
  { id: 1, common_name: 'Song Sparrow', scientific_name: 'Melospiza melodia', ebird_code: 'sonspa' },
  { id: 2, common_name: 'Fox Sparrow', scientific_name: 'Passerella iliaca', ebird_code: 'foxspa' },
]

describe('QuizCard', () => {
  it('renders an audio player with the media url', () => {
    render(QuizCard, { props: { card, species, onReveal: vi.fn() } })
    const audio = document.querySelector('audio')
    expect(audio).not.toBeNull()
    expect(audio!.src).toContain('/recordings/song-sparrow.mp3')
  })

  it('Reveal button is disabled initially', () => {
    render(QuizCard, { props: { card, species, onReveal: vi.fn() } })
    expect(screen.getByRole('button', { name: /reveal answer/i })).toBeDisabled()
  })

  it('Reveal button is enabled after a species is selected from the typeahead', async () => {
    render(QuizCard, { props: { card, species, onReveal: vi.fn() } })
    const input = screen.getByRole('textbox')
    await fireEvent.input(input, { target: { value: 'song' } })
    const option = screen.getByRole('option', { name: /song sparrow/i })
    await fireEvent.mouseDown(option)
    expect(screen.getByRole('button', { name: /reveal answer/i })).not.toBeDisabled()
  })

  it('calls onReveal with the selected species when Reveal is clicked', async () => {
    const onReveal = vi.fn()
    render(QuizCard, { props: { card, species, onReveal } })
    const input = screen.getByRole('textbox')
    await fireEvent.input(input, { target: { value: 'song' } })
    await fireEvent.mouseDown(screen.getByRole('option', { name: /song sparrow/i }))
    await fireEvent.click(screen.getByRole('button', { name: /reveal answer/i }))
    expect(onReveal).toHaveBeenCalledWith(species[0])
  })

  it("calls onReveal(null) when I don't know is clicked", async () => {
    const onReveal = vi.fn()
    render(QuizCard, { props: { card, species, onReveal } })
    await fireEvent.click(screen.getByRole('button', { name: /i don't know/i }))
    expect(onReveal).toHaveBeenCalledWith(null)
  })
})
```

- [ ] **Step 2: Run to confirm it fails**

```bash
cd frontend && npm test -- src/components/QuizCard.test.ts 2>&1 | head -30
```

Expected: FAIL -- QuizCard still has old props/behavior.

- [ ] **Step 3: Update `QuizCard.svelte`**

Replace the entire contents of `frontend/src/components/QuizCard.svelte`:

```svelte
<script lang="ts">
  import type { BirdCard, Species } from '../types'
  import SpeciesTypeahead from './SpeciesTypeahead.svelte'

  let { card, species, onReveal }: {
    card: BirdCard
    species: Species[]
    onReveal: (selected: Species | null) => void
  } = $props()

  let selected: Species | null = $state(null)
</script>

<div class="quiz-card">
  <div class="audio-wrapper">
    <audio controls src={card.media_url}>Your browser does not support audio.</audio>
  </div>
  <SpeciesTypeahead {species} onSelect={(s) => { selected = s }} />
  <div class="actions">
    <button
      class="btn-reveal"
      onclick={() => onReveal(selected)}
      disabled={selected === null}
    >
      Reveal answer
    </button>
    <button class="btn-skip" onclick={() => onReveal(null)}>
      I don't know
    </button>
  </div>
</div>

<style>
  .quiz-card {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }
  .audio-wrapper {
    background: var(--surface);
    border-radius: 8px;
    padding: 0.25rem;
    box-shadow: var(--shadow);
  }
  audio {
    width: 100%;
    display: block;
  }
  .actions {
    display: grid;
    grid-template-columns: 1fr auto;
    gap: 0.5rem;
  }
  .btn-reveal {
    background: var(--surface);
    border: 1px solid var(--border);
    color: var(--text-secondary);
    border-radius: 10px;
    padding: 0.75rem;
    font-size: 0.875rem;
    font-weight: 600;
    cursor: pointer;
    font-family: inherit;
    box-shadow: var(--shadow);
  }
  .btn-reveal:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }
  .btn-skip {
    background: transparent;
    border: 1px solid var(--border);
    color: var(--text-muted);
    border-radius: 10px;
    padding: 0.75rem 1rem;
    font-size: 0.8125rem;
    cursor: pointer;
    font-family: inherit;
    white-space: nowrap;
  }
</style>
```

- [ ] **Step 4: Run tests**

```bash
cd frontend && npm test -- src/components/QuizCard.test.ts
```

Expected: all 4 tests pass.

- [ ] **Step 5: Commit**

```bash
jj describe -m "feat: QuizCard uses SpeciesTypeahead with auto-check"
jj new
```

---

### Task 4: Update `ImageQuizCard`

**Files:**
- Modify: `frontend/src/components/ImageQuizCard.svelte`
- Create: `frontend/src/components/ImageQuizCard.test.ts`

- [ ] **Step 1: Write failing tests**

Create `frontend/src/components/ImageQuizCard.test.ts`:

```typescript
import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/svelte'
import ImageQuizCard from './ImageQuizCard.svelte'
import type { BirdCard, Species } from '../types'

const card: BirdCard = {
  species_id: 1,
  common_name: 'Song Sparrow',
  scientific_name: 'Melospiza melodia',
  media_url: '/photos/song-sparrow.jpg',
  photo_url: '/photos/song-sparrow.jpg',
  lane: 'image' as const,
}

const species: Species[] = [
  { id: 1, common_name: 'Song Sparrow', scientific_name: 'Melospiza melodia', ebird_code: 'sonspa' },
  { id: 2, common_name: 'Fox Sparrow', scientific_name: 'Passerella iliaca', ebird_code: 'foxspa' },
]

describe('ImageQuizCard', () => {
  it('renders an image with the media url', () => {
    render(ImageQuizCard, { props: { card, species, onReveal: vi.fn() } })
    const img = document.querySelector('img.quiz-photo')
    expect(img).not.toBeNull()
    expect(img!.getAttribute('src')).toBe('/photos/song-sparrow.jpg')
  })

  it('Reveal button is disabled initially', () => {
    render(ImageQuizCard, { props: { card, species, onReveal: vi.fn() } })
    expect(screen.getByRole('button', { name: /reveal answer/i })).toBeDisabled()
  })

  it('Reveal button is enabled after a species is selected from the typeahead', async () => {
    render(ImageQuizCard, { props: { card, species, onReveal: vi.fn() } })
    const input = screen.getByRole('textbox')
    await fireEvent.input(input, { target: { value: 'song' } })
    const option = screen.getByRole('option', { name: /song sparrow/i })
    await fireEvent.mouseDown(option)
    expect(screen.getByRole('button', { name: /reveal answer/i })).not.toBeDisabled()
  })

  it('calls onReveal with the selected species when Reveal is clicked', async () => {
    const onReveal = vi.fn()
    render(ImageQuizCard, { props: { card, species, onReveal } })
    const input = screen.getByRole('textbox')
    await fireEvent.input(input, { target: { value: 'song' } })
    await fireEvent.mouseDown(screen.getByRole('option', { name: /song sparrow/i }))
    await fireEvent.click(screen.getByRole('button', { name: /reveal answer/i }))
    expect(onReveal).toHaveBeenCalledWith(species[0])
  })

  it("calls onReveal(null) when I don't know is clicked", async () => {
    const onReveal = vi.fn()
    render(ImageQuizCard, { props: { card, species, onReveal } })
    await fireEvent.click(screen.getByRole('button', { name: /i don't know/i }))
    expect(onReveal).toHaveBeenCalledWith(null)
  })
})
```

- [ ] **Step 2: Run to confirm it fails**

```bash
cd frontend && npm test -- src/components/ImageQuizCard.test.ts 2>&1 | head -20
```

Expected: FAIL -- ImageQuizCard still has old props.

- [ ] **Step 3: Update `ImageQuizCard.svelte`**

Replace the entire contents of `frontend/src/components/ImageQuizCard.svelte`:

```svelte
<script lang="ts">
  import type { BirdCard, Species } from '../types'
  import SpeciesTypeahead from './SpeciesTypeahead.svelte'

  let { card, species, onReveal }: {
    card: BirdCard
    species: Species[]
    onReveal: (selected: Species | null) => void
  } = $props()

  let selected: Species | null = $state(null)
</script>

<div class="quiz-card">
  <div class="image-wrapper">
    <img src={card.media_url} alt="Identify this bird" class="quiz-photo" />
  </div>
  <SpeciesTypeahead {species} onSelect={(s) => { selected = s }} />
  <div class="actions">
    <button
      class="btn-reveal"
      onclick={() => onReveal(selected)}
      disabled={selected === null}
    >
      Reveal answer
    </button>
    <button class="btn-skip" onclick={() => onReveal(null)}>
      I don't know
    </button>
  </div>
</div>

<style>
  .quiz-card {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }
  .image-wrapper {
    background: var(--surface);
    border-radius: 8px;
    overflow: hidden;
    box-shadow: var(--shadow);
  }
  .quiz-photo {
    width: 100%;
    display: block;
    max-height: 280px;
    object-fit: cover;
  }
  .actions {
    display: grid;
    grid-template-columns: 1fr auto;
    gap: 0.5rem;
  }
  .btn-reveal {
    background: var(--surface);
    border: 1px solid var(--border);
    color: var(--text-secondary);
    border-radius: 10px;
    padding: 0.75rem;
    font-size: 0.875rem;
    font-weight: 600;
    cursor: pointer;
    font-family: inherit;
    box-shadow: var(--shadow);
  }
  .btn-reveal:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }
  .btn-skip {
    background: transparent;
    border: 1px solid var(--border);
    color: var(--text-muted);
    border-radius: 10px;
    padding: 0.75rem 1rem;
    font-size: 0.8125rem;
    cursor: pointer;
    font-family: inherit;
    white-space: nowrap;
  }
</style>
```

- [ ] **Step 4: Run tests**

```bash
cd frontend && npm test -- src/components/ImageQuizCard.test.ts
```

Expected: all 5 tests pass.

- [ ] **Step 5: Commit**

```bash
jj describe -m "feat: ImageQuizCard uses SpeciesTypeahead with auto-check"
jj new
```

---

### Task 5: Update `RevealCard`

**Files:**
- Modify: `frontend/src/components/RevealCard.svelte`
- Modify: `frontend/src/components/RevealCard.test.ts`

- [ ] **Step 1: Update `RevealCard.test.ts`**

Replace the entire contents of `frontend/src/components/RevealCard.test.ts`:

```typescript
import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/svelte'
import RevealCard from './RevealCard.svelte'
import type { BirdCard, Species } from '../types'

const card: BirdCard = {
  species_id: 1,
  common_name: 'Song Sparrow',
  scientific_name: 'Melospiza melodia',
  media_url: '/recordings/song-sparrow.mp3',
  photo_url: '/photos/song-sparrow.jpg',
  lane: 'audio' as const,
}

const songSparrow: Species = {
  id: 1, common_name: 'Song Sparrow', scientific_name: 'Melospiza melodia', ebird_code: 'sonspa',
}
const foxSparrow: Species = {
  id: 2, common_name: 'Fox Sparrow', scientific_name: 'Passerella iliaca', ebird_code: 'foxspa',
}

describe('RevealCard', () => {
  it('renders the species common and scientific name', () => {
    render(RevealCard, { props: { card, correct: true, guessed: songSparrow, onRate: vi.fn() } })
    expect(screen.getByText('Song Sparrow')).toBeInTheDocument()
    expect(screen.getByText('Melospiza melodia')).toBeInTheDocument()
  })

  it('renders a species photo', () => {
    render(RevealCard, { props: { card, correct: true, guessed: songSparrow, onRate: vi.fn() } })
    const img = screen.getByRole('img', { name: /song sparrow/i })
    expect(img).toHaveAttribute('src', '/photos/song-sparrow.jpg')
  })

  it('renders four confidence rating buttons', () => {
    render(RevealCard, { props: { card, correct: true, guessed: songSparrow, onRate: vi.fn() } })
    expect(screen.getByRole('button', { name: /again/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /hard/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /good/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /easy/i })).toBeInTheDocument()
  })

  it('calls onRate with 1 when Again is clicked', async () => {
    const onRate = vi.fn()
    render(RevealCard, { props: { card, correct: false, guessed: null, onRate } })
    await fireEvent.click(screen.getByRole('button', { name: /again/i }))
    expect(onRate).toHaveBeenCalledWith(1)
  })

  it('calls onRate with 4 when Easy is clicked', async () => {
    const onRate = vi.fn()
    render(RevealCard, { props: { card, correct: true, guessed: songSparrow, onRate } })
    await fireEvent.click(screen.getByRole('button', { name: /easy/i }))
    expect(onRate).toHaveBeenCalledWith(4)
  })

  it('shows correct banner when answer is right', () => {
    render(RevealCard, { props: { card, correct: true, guessed: songSparrow, onRate: vi.fn() } })
    expect(screen.getByText(/✓/)).toBeInTheDocument()
    expect(screen.getByText(/song sparrow/i, { selector: '.result-banner *' })).toBeInTheDocument()
  })

  it('shows wrong banner with guessed name when answer is wrong', () => {
    render(RevealCard, { props: { card, correct: false, guessed: foxSparrow, onRate: vi.fn() } })
    expect(screen.getByText(/✗/)).toBeInTheDocument()
    expect(screen.getByText(/you guessed: fox sparrow/i)).toBeInTheDocument()
  })

  it("shows 'You didn't know' when I don't know was selected", () => {
    render(RevealCard, { props: { card, correct: false, guessed: null, onRate: vi.fn() } })
    expect(screen.getByText(/you didn't know/i)).toBeInTheDocument()
  })

  it('Good button has suggested class when correct', () => {
    render(RevealCard, { props: { card, correct: true, guessed: songSparrow, onRate: vi.fn() } })
    expect(screen.getByRole('button', { name: /good/i })).toHaveClass('suggested')
    expect(screen.getByRole('button', { name: /again/i })).not.toHaveClass('suggested')
  })

  it('Again button has suggested class when incorrect', () => {
    render(RevealCard, { props: { card, correct: false, guessed: null, onRate: vi.fn() } })
    expect(screen.getByRole('button', { name: /again/i })).toHaveClass('suggested')
    expect(screen.getByRole('button', { name: /good/i })).not.toHaveClass('suggested')
  })
})
```

- [ ] **Step 2: Run to confirm it fails**

```bash
cd frontend && npm test -- src/components/RevealCard.test.ts 2>&1 | head -30
```

Expected: FAIL -- RevealCard missing `correct` and `guessed` props.

- [ ] **Step 3: Update `RevealCard.svelte`**

Replace the entire contents of `frontend/src/components/RevealCard.svelte`:

```svelte
<script lang="ts">
  import type { BirdCard, Species } from '../types'

  let { card, correct, guessed, onRate }: {
    card: BirdCard
    correct: boolean
    guessed: Species | null
    onRate: (rating: number) => void
  } = $props()

  const suggestedRating = $derived(correct ? 3 : 1)

  const ratings = [
    { label: 'Again', value: 1 },
    { label: 'Hard', value: 2 },
    { label: 'Good', value: 3 },
    { label: 'Easy', value: 4 },
  ]
</script>

<div class="reveal-card">
  <div class="result-banner" class:correct class:incorrect={!correct}>
    {#if correct}
      <span>✓ {card.common_name}</span>
    {:else if guessed}
      <span>✗ You guessed: {guessed.common_name}</span>
    {:else}
      <span>✗ You didn't know</span>
    {/if}
  </div>

  <img src={card.photo_url} alt={card.common_name} class="photo" />
  <div class="species">
    <p class="common-name">{card.common_name}</p>
    <p class="scientific-name">{card.scientific_name}</p>
  </div>
  <p class="how-well">How well did you know it?</p>
  <div class="ratings">
    {#each ratings as rating}
      <button
        class="rating-{rating.label.toLowerCase()}"
        class:suggested={rating.value === suggestedRating}
        onclick={() => onRate(rating.value)}
      >
        {rating.label}
      </button>
    {/each}
  </div>
</div>

<style>
  .reveal-card {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }
  .result-banner {
    border-radius: 8px;
    padding: 0.625rem 0.875rem;
    font-size: 0.875rem;
    font-weight: 600;
  }
  .result-banner.correct {
    background: rgba(20, 83, 45, 0.12);
    color: #14532d;
    border: 1px solid rgba(20, 83, 45, 0.3);
  }
  .result-banner.incorrect {
    background: rgba(127, 29, 29, 0.12);
    color: #7f1d1d;
    border: 1px solid rgba(127, 29, 29, 0.3);
  }
  :global([data-theme="dark"]) .result-banner.correct {
    background: rgba(20, 83, 45, 0.25);
    color: #4ade80;
    border-color: rgba(20, 83, 45, 0.6);
  }
  :global([data-theme="dark"]) .result-banner.incorrect {
    background: rgba(127, 29, 29, 0.25);
    color: #f87171;
    border-color: rgba(127, 29, 29, 0.6);
  }
  .photo {
    width: 100%;
    border-radius: 10px;
    max-height: 200px;
    object-fit: cover;
  }
  .common-name {
    color: var(--text);
    font-size: 1rem;
    font-weight: 700;
  }
  .scientific-name {
    color: var(--text-muted);
    font-size: 0.8125rem;
    font-style: italic;
  }
  .how-well {
    color: var(--text-muted);
    font-size: 0.6875rem;
    text-transform: uppercase;
    letter-spacing: 0.06em;
  }
  .ratings {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 0.5rem;
  }
  .ratings button {
    border: none;
    border-radius: 8px;
    padding: 0.625rem 0.25rem;
    font-size: 0.75rem;
    font-weight: 600;
    cursor: pointer;
    font-family: inherit;
    color: #fff;
  }
  .rating-again { background: #7f1d1d; }
  .rating-hard  { background: #78350f; }
  .rating-good  { background: #14532d; }
  .rating-easy  { background: #1e3a8a; }
  .ratings button.suggested {
    outline: 3px solid rgba(255, 255, 255, 0.85);
    outline-offset: 2px;
  }
</style>
```

- [ ] **Step 4: Run tests**

```bash
cd frontend && npm test -- src/components/RevealCard.test.ts
```

Expected: all 10 tests pass.

- [ ] **Step 5: Commit**

```bash
jj describe -m "feat: RevealCard shows correct/wrong result and suggested rating"
jj new
```

---

### Task 6: Update quiz page

**Files:**
- Modify: `frontend/src/routes/groups/[id]/quiz/+page.svelte`
- Modify: `frontend/src/routes/groups/[id]/quiz/page.test.ts`

- [ ] **Step 1: Update `page.test.ts`**

Replace the entire contents of `frontend/src/routes/groups/[id]/quiz/page.test.ts`:

```typescript
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/svelte'
import { goto } from '$app/navigation'
import { page } from '$app/state'
import QuizPage from './+page.svelte'

const card = {
  species_id: 99,
  common_name: 'Song Sparrow',
  scientific_name: 'Melospiza melodia',
  media_url: '/recordings/song-sparrow.mp3',
  photo_url: '/photos/song-sparrow.jpg',
  lane: 'audio',
}

const species = [
  { id: 99, common_name: 'Song Sparrow', scientific_name: 'Melospiza melodia', ebird_code: 'sonspa' },
  { id: 88, common_name: 'Fox Sparrow', scientific_name: 'Passerella iliaca', ebird_code: 'foxspa' },
]

function makeFetch(opts: { card?: object | null; status?: number } = {}) {
  return vi.fn().mockImplementation((url: string) => {
    if (url.includes('/species')) {
      return Promise.resolve({ ok: true, json: () => Promise.resolve(species) })
    }
    if (opts.status === 204 || opts.card === null) {
      return Promise.resolve({ ok: true, status: 204 })
    }
    return Promise.resolve({
      ok: true,
      status: 200,
      json: () => Promise.resolve(opts.card ?? card),
    })
  })
}

beforeEach(() => {
  page.params = { id: '42' }
  page.url = new URL('http://localhost/groups/42/quiz?lane=audio')
  vi.mocked(goto).mockClear()
})

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe('Quiz page', () => {
  it('shows loading initially', () => {
    vi.stubGlobal('fetch', vi.fn().mockReturnValue(new Promise(() => {})))
    render(QuizPage)
    expect(screen.getByText(/loading/i)).toBeInTheDocument()
  })

  it('fetches group species on mount', async () => {
    const fetchMock = makeFetch()
    vi.stubGlobal('fetch', fetchMock)
    render(QuizPage)
    await vi.waitFor(() => {
      const calls = fetchMock.mock.calls.map((c: unknown[]) => c[0] as string)
      expect(calls.some((url) => url.includes('/species'))).toBe(true)
    })
  })

  it('shows QuizCard (audio element) when a card is returned', async () => {
    vi.stubGlobal('fetch', makeFetch())
    render(QuizPage)
    await vi.waitFor(() => {
      expect(document.querySelector('audio')).not.toBeNull()
    })
  })

  it('shows All done when 204 is returned for next card', async () => {
    vi.stubGlobal('fetch', makeFetch({ status: 204 }))
    render(QuizPage)
    await vi.waitFor(() => {
      expect(screen.getByText(/all done/i)).toBeInTheDocument()
    })
  })

  it('navigates to group detail when Back to group is clicked', async () => {
    vi.stubGlobal('fetch', makeFetch({ status: 204 }))
    render(QuizPage)
    await vi.waitFor(() => screen.getByRole('button', { name: /back to group/i }))
    await fireEvent.click(screen.getByRole('button', { name: /back to group/i }))
    expect(goto).toHaveBeenCalledWith('/groups/42')
  })

  it('passes correct=true to RevealCard when selected species matches card', async () => {
    vi.stubGlobal('fetch', makeFetch())
    render(QuizPage)
    // Wait for quiz card to load
    await vi.waitFor(() => screen.getByRole('textbox'))
    // Type and select the correct species (id=99 = Song Sparrow = card.species_id)
    await fireEvent.input(screen.getByRole('textbox'), { target: { value: 'song' } })
    await vi.waitFor(() => screen.getByRole('option', { name: /song sparrow/i }))
    await fireEvent.mouseDown(screen.getByRole('option', { name: /song sparrow/i }))
    await fireEvent.click(screen.getByRole('button', { name: /reveal answer/i }))
    // RevealCard should show correct banner (✓)
    await vi.waitFor(() => {
      expect(screen.getByText(/✓/)).toBeInTheDocument()
    })
  })

  it("passes correct=false to RevealCard when I don't know is clicked", async () => {
    vi.stubGlobal('fetch', makeFetch())
    render(QuizPage)
    await vi.waitFor(() => screen.getByRole('button', { name: /i don't know/i }))
    await fireEvent.click(screen.getByRole('button', { name: /i don't know/i }))
    await vi.waitFor(() => {
      expect(screen.getByText(/✗/)).toBeInTheDocument()
    })
  })
})
```

- [ ] **Step 2: Run to confirm it fails**

```bash
cd frontend && npm test -- 'src/routes/groups/\[id\]/quiz/page.test.ts' 2>&1 | head -30
```

Expected: FAIL -- quiz page still has old onReveal signature, no species fetch.

- [ ] **Step 3: Update `+page.svelte`**

Replace the entire contents of `frontend/src/routes/groups/[id]/quiz/+page.svelte`:

```svelte
<script lang="ts">
  import { goto } from '$app/navigation'
  import { page } from '$app/state'
  import type { BirdCard, Species } from '../../../../types'
  import QuizCard from '$components/QuizCard.svelte'
  import ImageQuizCard from '$components/ImageQuizCard.svelte'
  import RevealCard from '$components/RevealCard.svelte'
  import StatsBar from '$components/StatsBar.svelte'

  let groupId = $derived(page.params.id)
  let lane: 'audio' | 'image' = $derived(
    page.url.searchParams.get('lane') === 'image' ? 'image' : 'audio'
  )

  let card: BirdCard | null = $state(null)
  let groupSpecies: Species[] = $state([])
  let revealed = $state(false)
  let done = $state(false)
  let reviewed = $state(0)
  let loading = $state(true)
  let error = $state('')
  let guessed: Species | null = $state(null)
  let correct = $state(false)

  async function loadGroupSpecies() {
    try {
      const res = await fetch(`/api/v1/groups/${groupId}/species`)
      if (res.ok) groupSpecies = await res.json()
    } catch {
      // non-fatal -- typeahead will be empty but quiz still works
    }
  }

  async function fetchNext() {
    loading = true
    error = ''
    try {
      const res = await fetch(`/api/v1/groups/${groupId}/next?lane=${lane}`)
      if (res.status === 204) {
        done = true
        card = null
        return
      }
      if (!res.ok) throw new Error(`Server error ${res.status}`)
      card = await res.json()
    } catch {
      error = 'Failed to load next card.'
    } finally {
      loading = false
    }
  }

  function onReveal(selected: Species | null) {
    guessed = selected
    correct = selected !== null && card !== null && selected.id === card.species_id
    revealed = true
  }

  async function onRate(rating: number) {
    if (!card) return
    try {
      await fetch(`/api/v1/groups/${groupId}/rate`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ species_id: card.species_id, lane: card.lane, rating }),
      })
    } catch {
      // non-fatal
    }
    reviewed += 1
    revealed = false
    guessed = null
    await fetchNext()
  }

  const stats = $derived([
    { label: 'Reviewed', value: reviewed },
    { label: 'Lane', value: lane === 'audio' ? '🔊 Audio' : '👁 Image' },
  ])

  $effect(() => {
    if (groupId) {
      reviewed = 0
      done = false
      revealed = false
      card = null
      guessed = null
      correct = false
      groupSpecies = []
      loadGroupSpecies()
      fetchNext()
    }
  })
</script>

<div class="quiz">
  <StatsBar {stats} />

  {#if loading}
    <p class="status">Loading...</p>
  {:else if error}
    <p class="status error">{error}</p>
  {:else if done}
    <div class="done">
      <p>All done for now!</p>
      <button onclick={() => goto(`/groups/${groupId}`)}>Back to group</button>
    </div>
  {:else if card}
    {#if revealed}
      <RevealCard {card} {correct} {guessed} {onRate} />
    {:else if lane === 'audio'}
      <QuizCard {card} species={groupSpecies} {onReveal} />
    {:else}
      <ImageQuizCard {card} species={groupSpecies} {onReveal} />
    {/if}
  {:else}
    <p class="status error">Something went wrong. <button onclick={fetchNext}>Retry</button></p>
  {/if}
</div>

<style>
  .quiz {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }
  .status {
    text-align: center;
    color: var(--text-muted);
    padding: 2rem 0;
  }
  .error {
    color: #b91c1c;
  }
  .done {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 1rem;
    padding: 2rem 0;
  }
  .done p {
    color: var(--text);
    font-size: 1rem;
    font-weight: 600;
  }
  .done button {
    background: var(--accent);
    color: #fff;
    border: none;
    border-radius: 10px;
    padding: 0.75rem 1.5rem;
    font-size: 0.9375rem;
    font-weight: 600;
    cursor: pointer;
    font-family: inherit;
  }
</style>
```

- [ ] **Step 4: Run quiz page tests**

```bash
cd frontend && npm test -- 'src/routes/groups/\[id\]/quiz/page.test.ts'
```

Expected: all 7 tests pass.

- [ ] **Step 5: Run all frontend tests**

```bash
cd frontend && npm test
```

Expected: all tests pass. Pay attention to any failures in `QuizCard.test.ts`, `RevealCard.test.ts`, or layout tests -- they may need the updated props threading through.

- [ ] **Step 6: Commit**

```bash
jj describe -m "feat: quiz auto-check with typeahead, correct/incorrect reveal"
jj new
```

---

## Self-Review Checklist

**Spec coverage:**
- [x] SpeciesTypeahead: filters on ≥2 chars, substring match on common + scientific name, max 10 results -- Task 2
- [x] Keyboard nav: arrows, Enter, Escape -- Task 2
- [x] Click to select: fills input, fires onSelect(species) -- Task 2
- [x] Clear below 2 chars: fires onSelect(null) -- Task 2
- [x] QuizCard: embeds SpeciesTypeahead, Reveal disabled until selection -- Task 3
- [x] "I don't know" button calls onReveal(null) -- Task 3
- [x] ImageQuizCard: same as QuizCard -- Task 4
- [x] RevealCard: correct banner (✓), wrong banner (✗ guessed / ✗ didn't know) -- Task 5
- [x] Suggested rating: Good highlighted when correct, Again when incorrect -- Task 5
- [x] All 4 rating buttons clickable regardless of suggestion -- Task 5
- [x] Quiz page: fetches group species once on mount -- Task 6
- [x] Quiz page: correct = selected.id === card.species_id -- Task 6
- [x] Quiz page: guessed cleared after rating -- Task 6
- [x] Species type moved to types.ts, imported everywhere -- Task 1 + all tasks

**Type consistency:**
- `onReveal(selected: Species | null)` -- defined in Tasks 3/4, used in Task 6 ✓
- `correct: boolean`, `guessed: Species | null` -- defined in Task 5, passed from Task 6 ✓
- `species: Species[]` prop -- defined in Tasks 2/3/4, sourced from Task 6 ✓
