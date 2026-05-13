# Svelte Frontend Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** Build a Svelte + Vite SPA in `frontend/` with login, dashboard, and quiz views for the Lifer bird song identification app.

**Architecture:** State-based view switching via a `$view` Svelte store. `App.svelte` checks `/api/v1/me` on mount and renders the current view. Dashboard shows stats + group list with mock data; Quiz orchestrates QuizCard ↔ RevealCard with mock card data. API stubs are one-line swaps when real endpoints exist.

**Tech Stack:** Svelte 5, Vite 6, Vitest 4, @testing-library/svelte 5, jsdom

> **Status: COMPLETE** (all 10 tasks done). Implemented with Svelte 5 runes (`$props()`, `$state()`, `$derived()`, `$effect()`, `onclick`) rather than Svelte 4 syntax (`export let`, `$:`, `on:click`, `onMount`). Files use `.ts` not `.js`. Required `resolve: { conditions: ['browser'] }` in vite.config.ts to prevent Svelte 5 SSR path in Vitest.

---

## File Map

| File | Responsibility |
|------|---------------|
| `frontend/vite.config.js` | Vite config: svelte plugin, `/api` proxy to `:8080`, Vitest jsdom env |
| `frontend/src/test-setup.js` | Imports `@testing-library/jest-dom` matchers |
| `frontend/src/main.js` | Mounts `App` to `#app` |
| `frontend/src/stores/auth.js` | `$auth` writable: user object or null |
| `frontend/src/stores/view.js` | `$view` writable: `'login' \| 'dashboard' \| 'quiz'` |
| `frontend/src/stores/session.js` | `$session` writable: `{ groupId: string \| null }` |
| `frontend/src/views/Login.svelte` | "Sign in with Google" link, no JS auth logic |
| `frontend/src/views/Dashboard.svelte` | Stats bar + Start Practice + group list (mock data) |
| `frontend/src/views/Quiz.svelte` | Manages QuizCard ↔ RevealCard, mock card queue |
| `frontend/src/components/StatsBar.svelte` | Renders an array of `{label, value}` stat pills |
| `frontend/src/components/QuizCard.svelte` | Audio player + text input + Reveal button |
| `frontend/src/components/RevealCard.svelte` | Photo + species name + 1-4 confidence buttons |
| `frontend/src/components/GroupList.svelte` | List of groups with due counts + Practice buttons |
| `frontend/src/App.svelte` | Fetches `/api/v1/me` on mount, routes by `$view` |

---

## Task 1: Scaffold Vite + Svelte project with Vitest

**Files:**
- Create: `frontend/` (via npm create vite)
- Modify: `frontend/vite.config.js`
- Create: `frontend/src/test-setup.js`
- Modify: `frontend/src/main.js`
- Modify: `Justfile`

- [x] **Step 1: Scaffold the project**

```bash
cd /path/to/lifer
npm create vite@latest frontend -- --template svelte
cd frontend && npm install
```

- [x] **Step 2: Install test dependencies**

```bash
cd frontend
npm install --save-dev vitest @testing-library/svelte @testing-library/jest-dom jsdom
```

- [x] **Step 3: Replace `frontend/vite.config.js`**

```js
import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'

export default defineConfig({
  plugins: [svelte()],
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
    },
  },
  test: {
    environment: 'jsdom',
    setupFiles: ['src/test-setup.js'],
    globals: true,
  },
})
```

- [x] **Step 4: Create `frontend/src/test-setup.js`**

```js
import '@testing-library/jest-dom'
```

- [x] **Step 5: Add test script to `frontend/package.json`**

In the `scripts` section, add:
```json
"test": "vitest run",
"test:watch": "vitest"
```

- [x] **Step 6: Clear template boilerplate**

Run from the repo root. Delete generated files that will be replaced:
```bash
rm frontend/src/App.svelte frontend/src/app.css frontend/public/vite.svg
rm -rf frontend/src/assets
```

Replace `frontend/src/main.js` with:
```js
import App from './App.svelte'

const app = new App({ target: document.getElementById('app') })

export default app
```

Update `frontend/index.html` title to `Lifer`:
```html
<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>Lifer</title>
  </head>
  <body>
    <div id="app"></div>
    <script type="module" src="/src/main.js"></script>
  </body>
</html>
```

- [x] **Step 7: Add frontend target to Justfile**

Append to `Justfile`:
```
# Start the frontend dev server
frontend:
    cd frontend && npm run dev
```

- [x] **Step 8: Verify Vitest is wired up**

```bash
cd frontend && npx vitest run
```

Expected: "No test files found" (not an error about missing config or deps).

- [x] **Step 9: Commit**

```bash
jj describe -m "scaffold Svelte frontend with Vite, Vitest, and /api proxy"
jj new
```

---

## Task 2: Svelte stores

**Files:**
- Create: `frontend/src/stores/auth.js`
- Create: `frontend/src/stores/view.js`
- Create: `frontend/src/stores/session.js`
- Create: `frontend/src/stores/stores.test.js`

- [x] **Step 1: Write failing tests**

Create `frontend/src/stores/stores.test.js`:
```js
import { describe, it, expect } from 'vitest'
import { get } from 'svelte/store'
import { auth } from './auth.js'
import { view } from './view.js'
import { session } from './session.js'

describe('auth store', () => {
  it('starts as null', () => {
    expect(get(auth)).toBe(null)
  })
})

describe('view store', () => {
  it('starts as login', () => {
    expect(get(view)).toBe('login')
  })
})

describe('session store', () => {
  it('starts with null groupId', () => {
    expect(get(session).groupId).toBe(null)
  })
})
```

- [x] **Step 2: Run to verify failure**

```bash
cd frontend && npx vitest run src/stores/stores.test.js
```

Expected: FAIL — "Cannot find module './auth.js'"

- [x] **Step 3: Create `frontend/src/stores/auth.js`**

```js
import { writable } from 'svelte/store'

/** @type {import('svelte/store').Writable<object | null>} */
export const auth = writable(null)
```

- [x] **Step 4: Create `frontend/src/stores/view.js`**

```js
import { writable } from 'svelte/store'

/** @type {import('svelte/store').Writable<'login' | 'dashboard' | 'quiz'>} */
export const view = writable('login')
```

- [x] **Step 5: Create `frontend/src/stores/session.js`**

```js
import { writable } from 'svelte/store'

/** @type {import('svelte/store').Writable<{ groupId: string | null }>} */
export const session = writable({ groupId: null })
```

- [x] **Step 6: Run to verify pass**

```bash
cd frontend && npx vitest run src/stores/stores.test.js
```

Expected: 3 tests pass.

- [x] **Step 7: Commit**

```bash
jj describe -m "add auth, view, and session Svelte stores"
jj new
```

---

## Task 3: Login.svelte

**Files:**
- Create: `frontend/src/views/Login.svelte`
- Create: `frontend/src/views/Login.test.js`

- [x] **Step 1: Write failing test**

Create `frontend/src/views/Login.test.js`:
```js
import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/svelte'
import Login from './Login.svelte'

describe('Login', () => {
  it('renders a link to the Google auth endpoint', () => {
    render(Login)
    const link = screen.getByRole('link', { name: /sign in with google/i })
    expect(link).toHaveAttribute('href', '/api/v1/auth/google')
  })
})
```

- [x] **Step 2: Run to verify failure**

```bash
cd frontend && npx vitest run src/views/Login.test.js
```

Expected: FAIL — "Cannot find module './Login.svelte'"

- [x] **Step 3: Create `frontend/src/views/Login.svelte`**

```svelte
<div class="login">
  <h1>Lifer</h1>
  <p>Bird song identification practice</p>
  <a href="/api/v1/auth/google">Sign in with Google</a>
</div>

<style>
  .login {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    min-height: 100vh;
    gap: 1rem;
  }
</style>
```

- [x] **Step 4: Run to verify pass**

```bash
cd frontend && npx vitest run src/views/Login.test.js
```

Expected: 1 test passes.

- [x] **Step 5: Commit**

```bash
jj describe -m "add Login view with Google auth link"
jj new
```

---

## Task 4: StatsBar.svelte

**Files:**
- Create: `frontend/src/components/StatsBar.svelte`
- Create: `frontend/src/components/StatsBar.test.js`

- [x] **Step 1: Write failing test**

Create `frontend/src/components/StatsBar.test.js`:
```js
import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/svelte'
import StatsBar from './StatsBar.svelte'

describe('StatsBar', () => {
  it('renders each stat value and label', () => {
    render(StatsBar, {
      props: {
        stats: [
          { label: 'Due today', value: 11 },
          { label: 'Streak', value: 5 },
        ],
      },
    })
    expect(screen.getByText('11')).toBeInTheDocument()
    expect(screen.getByText('Due today')).toBeInTheDocument()
    expect(screen.getByText('5')).toBeInTheDocument()
    expect(screen.getByText('Streak')).toBeInTheDocument()
  })

  it('renders nothing when stats is empty', () => {
    const { container } = render(StatsBar, { props: { stats: [] } })
    expect(container.querySelector('.stat')).toBeNull()
  })
})
```

- [x] **Step 2: Run to verify failure**

```bash
cd frontend && npx vitest run src/components/StatsBar.test.js
```

Expected: FAIL — "Cannot find module './StatsBar.svelte'"

- [x] **Step 3: Create `frontend/src/components/StatsBar.svelte`**

```svelte
<script>
  /**
   * @type {Array<{ label: string, value: string | number }>}
   */
  export let stats = []
</script>

<div class="stats-bar">
  {#each stats as stat}
    <div class="stat">
      <span class="value">{stat.value}</span>
      <span class="label">{stat.label}</span>
    </div>
  {/each}
</div>

<style>
  .stats-bar {
    display: flex;
    gap: 0.75rem;
  }
  .stat {
    display: flex;
    flex-direction: column;
    align-items: center;
    background: #1e293b;
    border-radius: 6px;
    padding: 0.5rem 0.75rem;
    min-width: 70px;
  }
  .value {
    font-size: 1.25rem;
    font-weight: 700;
  }
  .label {
    font-size: 0.625rem;
    color: #64748b;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }
</style>
```

- [x] **Step 4: Run to verify pass**

```bash
cd frontend && npx vitest run src/components/StatsBar.test.js
```

Expected: 2 tests pass.

- [x] **Step 5: Commit**

```bash
jj describe -m "add StatsBar component"
jj new
```

---

## Task 5: QuizCard.svelte

**Files:**
- Create: `frontend/src/components/QuizCard.svelte`
- Create: `frontend/src/components/QuizCard.test.js`

- [x] **Step 1: Write failing tests**

Create `frontend/src/components/QuizCard.test.js`:
```js
import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/svelte'
import QuizCard from './QuizCard.svelte'

const card = { recording_path: '/recordings/song-sparrow.mp3' }

describe('QuizCard', () => {
  it('renders an audio player with the recording path', () => {
    render(QuizCard, { props: { card, onReveal: vi.fn() } })
    const audio = document.querySelector('audio')
    expect(audio).not.toBeNull()
    expect(audio.src).toContain('/recordings/song-sparrow.mp3')
  })

  it('renders a text input for the species guess', () => {
    render(QuizCard, { props: { card, onReveal: vi.fn() } })
    expect(screen.getByPlaceholderText(/type species name/i)).toBeInTheDocument()
  })

  it('calls onReveal when Reveal Answer is clicked', async () => {
    const onReveal = vi.fn()
    render(QuizCard, { props: { card, onReveal } })
    await fireEvent.click(screen.getByRole('button', { name: /reveal answer/i }))
    expect(onReveal).toHaveBeenCalledOnce()
  })
})
```

- [x] **Step 2: Run to verify failure**

```bash
cd frontend && npx vitest run src/components/QuizCard.test.js
```

Expected: FAIL — "Cannot find module './QuizCard.svelte'"

- [x] **Step 3: Create `frontend/src/components/QuizCard.svelte`**

```svelte
<script>
  /** @type {{ recording_path: string }} */
  export let card

  /** @type {() => void} */
  export let onReveal

  let guess = ''
</script>

<div class="quiz-card">
  <audio controls src={card.recording_path}></audio>
  <input
    bind:value={guess}
    placeholder="Type species name..."
  />
  <button on:click={onReveal}>Reveal Answer</button>
</div>

<style>
  .quiz-card {
    display: flex;
    flex-direction: column;
    gap: 1rem;
    width: 100%;
  }
  audio {
    width: 100%;
  }
  input {
    width: 100%;
    padding: 0.75rem;
    font-size: 1rem;
    background: #1e293b;
    border: 1px solid #334155;
    border-radius: 6px;
    color: #f1f5f9;
  }
  button {
    width: 100%;
    padding: 0.75rem;
    background: #1e3a5f;
    color: #f1f5f9;
    border: none;
    border-radius: 6px;
    font-size: 1rem;
    cursor: pointer;
  }
</style>
```

- [x] **Step 4: Run to verify pass**

```bash
cd frontend && npx vitest run src/components/QuizCard.test.js
```

Expected: 3 tests pass.

- [x] **Step 5: Commit**

```bash
jj describe -m "add QuizCard component"
jj new
```

---

## Task 6: RevealCard.svelte

**Files:**
- Create: `frontend/src/components/RevealCard.svelte`
- Create: `frontend/src/components/RevealCard.test.js`

- [x] **Step 1: Write failing tests**

Create `frontend/src/components/RevealCard.test.js`:
```js
import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/svelte'
import RevealCard from './RevealCard.svelte'

const card = {
  common_name: 'Song Sparrow',
  scientific_name: 'Melospiza melodia',
  photo_path: '/photos/song-sparrow.jpg',
}

describe('RevealCard', () => {
  it('renders the species common and scientific name', () => {
    render(RevealCard, { props: { card, onRate: vi.fn() } })
    expect(screen.getByText('Song Sparrow')).toBeInTheDocument()
    expect(screen.getByText('Melospiza melodia')).toBeInTheDocument()
  })

  it('renders a species photo', () => {
    render(RevealCard, { props: { card, onRate: vi.fn() } })
    const img = screen.getByRole('img', { name: /song sparrow/i })
    expect(img).toHaveAttribute('src', '/photos/song-sparrow.jpg')
  })

  it('renders four confidence rating buttons', () => {
    render(RevealCard, { props: { card, onRate: vi.fn() } })
    expect(screen.getByRole('button', { name: /again/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /hard/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /good/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /easy/i })).toBeInTheDocument()
  })

  it('calls onRate with 1 when Again is clicked', async () => {
    const onRate = vi.fn()
    render(RevealCard, { props: { card, onRate } })
    await fireEvent.click(screen.getByRole('button', { name: /again/i }))
    expect(onRate).toHaveBeenCalledWith(1)
  })

  it('calls onRate with 4 when Easy is clicked', async () => {
    const onRate = vi.fn()
    render(RevealCard, { props: { card, onRate } })
    await fireEvent.click(screen.getByRole('button', { name: /easy/i }))
    expect(onRate).toHaveBeenCalledWith(4)
  })
})
```

- [x] **Step 2: Run to verify failure**

```bash
cd frontend && npx vitest run src/components/RevealCard.test.js
```

Expected: FAIL — "Cannot find module './RevealCard.svelte'"

- [x] **Step 3: Create `frontend/src/components/RevealCard.svelte`**

```svelte
<script>
  /**
   * @type {{ common_name: string, scientific_name: string, photo_path: string }}
   */
  export let card

  /** @type {(rating: number) => void} */
  export let onRate

  const ratings = [
    { value: 1, label: 'Again', color: '#7f1d1d' },
    { value: 2, label: 'Hard', color: '#78350f' },
    { value: 3, label: 'Good', color: '#14532d' },
    { value: 4, label: 'Easy', color: '#1e3a5f' },
  ]
</script>

<div class="reveal-card">
  <img src={card.photo_path} alt={card.common_name} />
  <div class="species">
    <h2>{card.common_name}</h2>
    <p>{card.scientific_name}</p>
  </div>
  <p class="prompt">How well did you know it?</p>
  <div class="ratings">
    {#each ratings as r}
      <button
        style="background: {r.color}"
        on:click={() => onRate(r.value)}
      >
        {r.value} {r.label}
      </button>
    {/each}
  </div>
</div>

<style>
  .reveal-card {
    display: flex;
    flex-direction: column;
    gap: 1rem;
    width: 100%;
  }
  img {
    width: 100%;
    border-radius: 8px;
    max-height: 200px;
    object-fit: cover;
  }
  .species h2 {
    font-size: 1.25rem;
    font-weight: 600;
    color: #f1f5f9;
    margin: 0;
  }
  .species p {
    font-size: 0.875rem;
    color: #64748b;
    font-style: italic;
    margin: 0;
  }
  .prompt {
    font-size: 0.75rem;
    color: #64748b;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    margin: 0;
  }
  .ratings {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 0.5rem;
  }
  button {
    padding: 0.625rem 0;
    color: #f1f5f9;
    border: none;
    border-radius: 6px;
    font-size: 0.875rem;
    cursor: pointer;
  }
</style>
```

- [x] **Step 4: Run to verify pass**

```bash
cd frontend && npx vitest run src/components/RevealCard.test.js
```

Expected: 5 tests pass.

- [x] **Step 5: Commit**

```bash
jj describe -m "add RevealCard component with confidence rating buttons"
jj new
```

---

## Task 7: GroupList.svelte

**Files:**
- Create: `frontend/src/components/GroupList.svelte`
- Create: `frontend/src/components/GroupList.test.js`

- [x] **Step 1: Write failing tests**

Create `frontend/src/components/GroupList.test.js`:
```js
import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/svelte'
import GroupList from './GroupList.svelte'

const groups = [
  { id: '1', name: 'Pacific Northwest', is_preset: true, due_count: 8 },
  { id: '2', name: 'My Warblers', is_preset: false, due_count: 3 },
]

describe('GroupList', () => {
  it('renders each group name and due count', () => {
    render(GroupList, { props: { groups, onPractice: vi.fn() } })
    expect(screen.getByText('Pacific Northwest')).toBeInTheDocument()
    expect(screen.getByText('8 due')).toBeInTheDocument()
    expect(screen.getByText('My Warblers')).toBeInTheDocument()
    expect(screen.getByText('3 due')).toBeInTheDocument()
  })

  it('renders a Practice button for each group', () => {
    render(GroupList, { props: { groups, onPractice: vi.fn() } })
    const buttons = screen.getAllByRole('button', { name: /practice/i })
    expect(buttons).toHaveLength(2)
  })

  it('calls onPractice with the group when its button is clicked', async () => {
    const onPractice = vi.fn()
    render(GroupList, { props: { groups, onPractice } })
    const buttons = screen.getAllByRole('button', { name: /practice/i })
    await fireEvent.click(buttons[0])
    expect(onPractice).toHaveBeenCalledWith(groups[0])
  })
})
```

- [x] **Step 2: Run to verify failure**

```bash
cd frontend && npx vitest run src/components/GroupList.test.js
```

Expected: FAIL — "Cannot find module './GroupList.svelte'"

- [x] **Step 3: Create `frontend/src/components/GroupList.svelte`**

```svelte
<script>
  /**
   * @typedef {{ id: string, name: string, is_preset: boolean, due_count: number }} Group
   */

  /** @type {Group[]} */
  export let groups = []

  /** @type {(group: Group) => void} */
  export let onPractice
</script>

<ul class="group-list">
  {#each groups as group}
    <li>
      <div class="info">
        <span class="name">{group.name}</span>
        <span class="due">{group.due_count} due</span>
      </div>
      <button on:click={() => onPractice(group)}>Practice</button>
    </li>
  {/each}
</ul>

<style>
  .group-list {
    list-style: none;
    padding: 0;
    margin: 0;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }
  li {
    display: flex;
    justify-content: space-between;
    align-items: center;
    background: #1e293b;
    border-radius: 8px;
    padding: 0.75rem 1rem;
  }
  .info {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
  }
  .name {
    font-size: 0.875rem;
    color: #f1f5f9;
  }
  .due {
    font-size: 0.75rem;
    color: #64748b;
  }
  button {
    padding: 0.375rem 0.75rem;
    background: #1e3a5f;
    color: #f1f5f9;
    border: none;
    border-radius: 6px;
    font-size: 0.875rem;
    cursor: pointer;
  }
</style>
```

- [x] **Step 4: Run to verify pass**

```bash
cd frontend && npx vitest run src/components/GroupList.test.js
```

Expected: 3 tests pass.

- [x] **Step 5: Commit**

```bash
jj describe -m "add GroupList component"
jj new
```

---

## Task 8: Dashboard.svelte

**Files:**
- Create: `frontend/src/views/Dashboard.svelte`
- Create: `frontend/src/views/Dashboard.test.js`

- [x] **Step 1: Write failing tests**

Create `frontend/src/views/Dashboard.test.js`:
```js
import { describe, it, expect, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/svelte'
import { get } from 'svelte/store'
import { view } from '../stores/view.js'
import { session } from '../stores/session.js'
import Dashboard from './Dashboard.svelte'

describe('Dashboard', () => {
  beforeEach(() => {
    view.set('dashboard')
    session.set({ groupId: null })
  })

  it('renders a Start Practice button', () => {
    render(Dashboard)
    expect(screen.getByRole('button', { name: /start practice/i })).toBeInTheDocument()
  })

  it('renders the group with the most due cards prominently', () => {
    render(Dashboard)
    // Pacific Northwest has 8 due, highest of the mock groups
    expect(screen.getByText(/pacific northwest/i)).toBeInTheDocument()
  })

  it('sets session.groupId and switches to quiz when Start Practice clicked', async () => {
    render(Dashboard)
    await fireEvent.click(screen.getByRole('button', { name: /start practice/i }))
    expect(get(session).groupId).toBe('1')
    expect(get(view)).toBe('quiz')
  })

  it('switches to quiz when a group Practice button is clicked', async () => {
    render(Dashboard)
    const buttons = screen.getAllByRole('button', { name: /practice/i })
    await fireEvent.click(buttons[0])
    expect(get(view)).toBe('quiz')
  })
})
```

- [x] **Step 2: Run to verify failure**

```bash
cd frontend && npx vitest run src/views/Dashboard.test.js
```

Expected: FAIL — "Cannot find module './Dashboard.svelte'"

- [x] **Step 3: Create `frontend/src/views/Dashboard.svelte`**

```svelte
<script>
  import { session } from '../stores/session.js'
  import { view } from '../stores/view.js'
  import StatsBar from '../components/StatsBar.svelte'
  import GroupList from '../components/GroupList.svelte'

  // Stub: replace with GET /api/v1/groups when endpoint exists
  const MOCK_GROUPS = [
    { id: '1', name: 'Pacific Northwest', is_preset: true, due_count: 8 },
    { id: '2', name: 'My Warblers', is_preset: false, due_count: 3 },
  ]

  const groups = MOCK_GROUPS

  $: topGroup = groups.reduce(
    (best, g) => (g.due_count > best.due_count ? g : best),
    groups[0]
  )

  $: stats = [
    { label: 'Due today', value: groups.reduce((sum, g) => sum + g.due_count, 0) },
    { label: 'Day streak', value: 5 },
    { label: 'Species', value: 47 },
  ]

  /** @param {{ id: string }} group */
  function startPractice(group) {
    $session = { groupId: group.id }
    $view = 'quiz'
  }
</script>

<div class="dashboard">
  <StatsBar {stats} />

  {#if topGroup}
    <div class="quick-start">
      <button on:click={() => startPractice(topGroup)}>Start Practice</button>
      <p>{topGroup.name} · {topGroup.due_count} due</p>
    </div>
  {/if}

  <GroupList {groups} onPractice={startPractice} />
</div>

<style>
  .dashboard {
    display: flex;
    flex-direction: column;
    gap: 1.5rem;
    padding: 1.5rem;
    max-width: 480px;
    margin: 0 auto;
  }
  .quick-start button {
    width: 100%;
    padding: 1rem;
    background: #1e3a5f;
    color: #f1f5f9;
    border: none;
    border-radius: 8px;
    font-size: 1rem;
    cursor: pointer;
  }
  .quick-start p {
    text-align: center;
    font-size: 0.75rem;
    color: #64748b;
    margin: 0.5rem 0 0;
  }
</style>
```

- [x] **Step 4: Run to verify pass**

```bash
cd frontend && npx vitest run src/views/Dashboard.test.js
```

Expected: 4 tests pass.

- [x] **Step 5: Commit**

```bash
jj describe -m "add Dashboard view with mock group data"
jj new
```

---

## Task 9: Quiz.svelte

**Files:**
- Create: `frontend/src/views/Quiz.svelte`
- Create: `frontend/src/views/Quiz.test.js`

- [x] **Step 1: Write failing tests**

Create `frontend/src/views/Quiz.test.js`:
```js
import { describe, it, expect, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/svelte'
import { get } from 'svelte/store'
import { view } from '../stores/view.js'
import { session } from '../stores/session.js'
import Quiz from './Quiz.svelte'

describe('Quiz', () => {
  beforeEach(() => {
    session.set({ groupId: '1' })
    view.set('quiz')
  })

  it('shows QuizCard initially', () => {
    render(Quiz)
    expect(screen.getByRole('button', { name: /reveal answer/i })).toBeInTheDocument()
  })

  it('shows RevealCard after clicking Reveal Answer', async () => {
    render(Quiz)
    await fireEvent.click(screen.getByRole('button', { name: /reveal answer/i }))
    expect(screen.getByText('Song Sparrow')).toBeInTheDocument()
  })

  it('returns to QuizCard for next card after rating', async () => {
    render(Quiz)
    await fireEvent.click(screen.getByRole('button', { name: /reveal answer/i }))
    await fireEvent.click(screen.getByRole('button', { name: /good/i }))
    expect(screen.getByRole('button', { name: /reveal answer/i })).toBeInTheDocument()
  })

  it('navigates to dashboard when all cards are rated', async () => {
    render(Quiz)
    // Rate both mock cards
    for (let i = 0; i < 2; i++) {
      await fireEvent.click(screen.getByRole('button', { name: /reveal answer/i }))
      await fireEvent.click(screen.getByRole('button', { name: /good/i }))
    }
    expect(get(view)).toBe('dashboard')
  })
})
```

- [x] **Step 2: Run to verify failure**

```bash
cd frontend && npx vitest run src/views/Quiz.test.js
```

Expected: FAIL — "Cannot find module './Quiz.svelte'"

- [x] **Step 3: Create `frontend/src/views/Quiz.svelte`**

```svelte
<script>
  import { view } from '../stores/view.js'
  import StatsBar from '../components/StatsBar.svelte'
  import QuizCard from '../components/QuizCard.svelte'
  import RevealCard from '../components/RevealCard.svelte'

  // Stub: replace with GET /api/v1/groups/:id/next when endpoint exists
  const MOCK_CARDS = [
    {
      id: '1',
      recording_path: '/recordings/song-sparrow.mp3',
      common_name: 'Song Sparrow',
      scientific_name: 'Melospiza melodia',
      photo_path: '/photos/song-sparrow.jpg',
    },
    {
      id: '2',
      recording_path: '/recordings/spotted-towhee.mp3',
      common_name: 'Spotted Towhee',
      scientific_name: 'Pipilo maculatus',
      photo_path: '/photos/spotted-towhee.jpg',
    },
  ]

  let cards = [...MOCK_CARDS]
  let current = cards[0]
  let revealed = false
  let reviewedToday = 0

  $: stats = [
    { label: 'Due remaining', value: cards.length },
    { label: 'Reviewed today', value: reviewedToday },
    { label: 'Streak', value: 5 },
  ]

  function reveal() {
    revealed = true
  }

  /** @param {number} rating */
  function rate(rating) {
    // Stub: POST /api/v1/groups/:id/rate with { card_id: current.id, rating }
    cards = cards.slice(1)
    reviewedToday += 1
    if (cards.length === 0) {
      $view = 'dashboard'
      return
    }
    current = cards[0]
    revealed = false
  }
</script>

<div class="quiz">
  <StatsBar {stats} />
  {#if revealed}
    <RevealCard card={current} onRate={rate} />
  {:else}
    <QuizCard card={current} onReveal={reveal} />
  {/if}
</div>

<style>
  .quiz {
    display: flex;
    flex-direction: column;
    gap: 1.5rem;
    padding: 1.5rem;
    max-width: 480px;
    margin: 0 auto;
  }
</style>
```

- [x] **Step 4: Run to verify pass**

```bash
cd frontend && npx vitest run src/views/Quiz.test.js
```

Expected: 4 tests pass.

- [x] **Step 5: Commit**

```bash
jj describe -m "add Quiz view with mock card queue"
jj new
```

---

## Task 10: App.svelte (auth check + view routing)

**Files:**
- Create: `frontend/src/App.svelte`
- Create: `frontend/src/App.test.js`

- [x] **Step 1: Write failing tests**

Create `frontend/src/App.test.js`:
```js
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, waitFor } from '@testing-library/svelte'
import { get } from 'svelte/store'
import { view } from './stores/view.js'
import { auth } from './stores/auth.js'
import App from './App.svelte'

describe('App', () => {
  beforeEach(() => {
    view.set('login')
    auth.set(null)
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
  })

  it('sets view to login when /api/v1/me returns 401', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false, status: 401 }))
    render(App)
    await waitFor(() => expect(get(view)).toBe('login'))
  })

  it('sets auth and view to dashboard when /api/v1/me returns 200', async () => {
    const user = { id: 1, name: 'Jamey', email: 'jamey@example.com' }
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: async () => user,
    }))
    render(App)
    await waitFor(() => {
      expect(get(view)).toBe('dashboard')
      expect(get(auth)).toEqual(user)
    })
  })
})
```

- [x] **Step 2: Run to verify failure**

```bash
cd frontend && npx vitest run src/App.test.js
```

Expected: FAIL — "Cannot find module './App.svelte'"

- [x] **Step 3: Create `frontend/src/App.svelte`**

```svelte
<script>
  import { onMount } from 'svelte'
  import { auth } from './stores/auth.js'
  import { view } from './stores/view.js'
  import Login from './views/Login.svelte'
  import Dashboard from './views/Dashboard.svelte'
  import Quiz from './views/Quiz.svelte'

  onMount(async () => {
    const res = await fetch('/api/v1/me')
    if (!res.ok) {
      $view = 'login'
      return
    }
    $auth = await res.json()
    $view = 'dashboard'
  })
</script>

{#if $view === 'login'}
  <Login />
{:else if $view === 'dashboard'}
  <Dashboard />
{:else if $view === 'quiz'}
  <Quiz />
{/if}
```

- [x] **Step 4: Run to verify pass**

```bash
cd frontend && npx vitest run src/App.test.js
```

Expected: 2 tests pass.

- [x] **Step 5: Run the full test suite**

```bash
cd frontend && npx vitest run
```

Expected: All tests pass across all files.

- [x] **Step 6: Verify the dev server starts**

```bash
just frontend
```

Open http://localhost:5173 — should show the Login view (since there's no auth cookie in the browser). Clicking "Sign in with Google" should redirect to the backend OAuth flow.

- [x] **Step 7: Commit**

```bash
jj describe -m "add App root component with auth check and view routing"
jj new
```
