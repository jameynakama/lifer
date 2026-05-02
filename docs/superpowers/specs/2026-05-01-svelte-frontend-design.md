# Svelte Frontend Design

**Date:** 2026-05-01
**Status:** Approved

## Overview

A Svelte + Vite single-page app in `frontend/` for the Lifer bird song/call identification quiz. No SvelteKit yet -- view switching is state-based via a Svelte store. The quiz loop is the core interaction: hear a recording, type a guess, reveal the answer, rate confidence 1-4 (FSRS).

## File Structure

```
frontend/
  src/
    App.svelte             # root: reads $view, renders current view
    stores/
      auth.js              # $auth: user object from /api/v1/me, null if logged out
      view.js              # $view: 'login' | 'dashboard' | 'quiz'
      session.js           # $session: active group ID (current card managed locally in Quiz.svelte)
    views/
      Login.svelte         # "Sign in with Google" link to /api/v1/auth/google
      Dashboard.svelte     # stats bar + quick-start button + groups list
      Quiz.svelte          # orchestrates QuizCard <-> RevealCard swap
    components/
      StatsBar.svelte      # flexible stats bar; props vary by context (see Views)
      QuizCard.svelte      # audio player, text input, reveal button
      RevealCard.svelte    # species photo + name + confidence buttons 1-4
      GroupList.svelte     # groups with due counts and Practice buttons
  index.html
  vite.config.js           # proxies /api -> :8080 to avoid CORS in dev
  package.json
```

## Routing

No router library. A `$view` Svelte store (`'login' | 'dashboard' | 'quiz'`) drives which view `App.svelte` renders. This is sufficient for the current app -- the user flow is linear (login → dashboard → quiz → dashboard). SvelteKit is the natural upgrade path when outer pages (about, etc.) are needed.

## Views

### Login
A centered "Sign in with Google" link pointing to `/api/v1/auth/google`. Full page navigation, not a fetch -- the OAuth redirect flow handles the rest. On return, the backend sets the HttpOnly JWT cookie and redirects to `http://localhost:5173/`, where `App.svelte` re-checks auth.

### Dashboard
Layout: stats bar (due today, day streak, species count) at top, large "Start Practice" button for the most-due group below it, remaining groups listed underneath with individual Practice buttons and due counts. An "+ Add group" affordance at the bottom.

### Quiz
Two states, swapped in place:

**QuizCard state:** Stats bar (remaining due in session, reviewed today, streak) + audio player + text input + Reveal button.

**RevealCard state:** Stats bar + species photo + correct name (common + scientific) + "How well did you know it?" + confidence buttons 1-4 (Again / Hard / Good / Easy), color-coded red/amber/green/blue.

On rating, POST to backend, advance to next card. When the session queue is empty, return to dashboard.

## Data Flow

1. `App.svelte` mounts → `GET /api/v1/me`
   - 401: `$view = 'login'`
   - 200: set `$auth`, `$view = 'dashboard'`
2. `Dashboard.svelte` mounts → `GET /api/v1/groups` (due counts included)
3. User clicks Start / Practice → set `$session` (group ID), `$view = 'quiz'`
4. `Quiz.svelte` mounts → `GET /api/v1/groups/:id/next` for first card
5. User reveals + rates → `POST /api/v1/groups/:id/rate` → fetch next card
6. No more cards → `$view = 'dashboard'`

Auth is cookie-based (HttpOnly JWT). JS never reads the token -- it rides along on every fetch automatically.

## API Contract

These backend endpoints are required. During frontend development, stub them with hardcoded mock data.

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/api/v1/me` | Auth check + user info (exists) |
| `GET` | `/api/v1/groups` | User's groups with due counts |
| `GET` | `/api/v1/groups/:id/next` | Next due card (recording path, common name, scientific name, photo path) |
| `POST` | `/api/v1/groups/:id/rate` | Submit confidence rating 1-4, advance FSRS |

## Out of Scope

- Admin UI (catalog management)
- Group creation/editing UI
- Image identification quiz mode
- FSRS implementation (backend concern)
- SvelteKit migration
