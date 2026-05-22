# Lifer Styling Design

**Date:** 2026-05-21
**Status:** Approved

## Overview

Style all Svelte frontend views and components to match the approved mockups. Dark/light mode with system default + manual override. Inter font. Global CSS custom properties for the token layer; Svelte scoped `<style>` blocks for component-specific layout.

---

## CSS Architecture

**Global file:** `frontend/src/app.css` — imported once in `main.ts`. Contains:
- CSS reset (box-sizing, margin, padding)
- CSS custom properties (design tokens) for both themes
- `body` defaults: font, background, color, min-height
- Dark/light mode switching via `[data-theme]` attribute + `prefers-color-scheme` media query

**Component styles:** Scoped `<style>` blocks inside each `.svelte` file. Reference global tokens via `var(--token-name)`. No component defines its own colors — all colors come from tokens.

**No Tailwind.** No CSS framework.

---

## Design Tokens

Two themes: `dark` (default) and `light`. Tokens defined in `app.css`, swapped by setting `data-theme="light"` on `<html>`.

| Token | Dark | Light |
|---|---|---|
| `--bg` | `#0f172a` | `#f8f7f4` |
| `--surface` | `#1e293b` | `#ffffff` |
| `--border` | `#334155` | `#e2e8f0` |
| `--text` | `#f1f5f9` | `#1c1917` |
| `--text-muted` | `#64748b` | `#a8a29e` |
| `--text-secondary` | `#94a3b8` | `#78716c` |
| `--accent` | `#2563eb` | `#4f7942` |
| `--accent-hover` | `#1d4ed8` | `#3d6133` |
| `--shadow` | `none` | `0 1px 3px rgba(0,0,0,.08)` |

Confidence button colors are fixed regardless of theme (semantic meaning):
| Rating | Color |
|---|---|
| Again (1) | `#7f1d1d` |
| Hard (2) | `#78350f` |
| Good (3) | `#14532d` |
| Easy (4) | `#1e3a8a` |

---

## Dark / Light Mode

**Default:** follows `prefers-color-scheme` OS setting.
**Override:** user clicks a sun/moon toggle button. Preference persisted to `localStorage` under key `lifer-theme`.

**Implementation:**
- On app init, read `localStorage`. If set, apply `data-theme` to `<html>`. If not set, apply based on `matchMedia('(prefers-color-scheme: dark)')`.
- Toggle button in the top bar of every view (App-level, not per-view).
- Toggle button shows ☀️ in dark mode, 🌙 in light mode.

Theme logic lives in `frontend/src/lib/theme.ts` — a small module with `initTheme()` and `toggleTheme()`. Called from `App.svelte` on mount.

---

## Typography

**Font:** Inter (Google Fonts), weights 400 / 600 / 700.
**Loading:** `<link>` in `index.html` (preconnect + stylesheet). No JS font loading.

```html
<link rel="preconnect" href="https://fonts.googleapis.com">
<link href="https://fonts.googleapis.com/css2?family=Inter:ital,wght@0,400;0,600;0,700;1,400&display=swap" rel="stylesheet">
```

Body default: `font-family: 'Inter', system-ui, sans-serif; font-size: 16px; line-height: 1.5`.

---

## Layout

All views: `max-width: 480px`, centered (`margin: 0 auto`), `padding: 0 1.5rem 2rem`. This gives a phone-width column on desktop and fills the screen on mobile.

Top bar: `display: flex; justify-content: space-between; align-items: center; padding: 1rem 0 1.25rem`. Contains app name left, theme toggle right.

---

## Component Specs

### App.svelte
- Renders top bar when `$view !== 'login'`: "Lifer" wordmark left, theme toggle (☀️/🌙) right.
- Login gets no top bar -- the atmospheric layout fills the full viewport.
- Calls `initTheme()` in `$effect`.
- Spinner uses `--text-muted` border, `--text-secondary` active slice.

### Login.svelte
- Full-viewport centered layout.
- Background: dark slate (`#0f172a`) regardless of theme (login is always dark — atmospheric).
- Layered: diagonal gradient (`#0c2340` → `#0f172a` → `#1a0a0a`) + ghost bird emoji at ~5% opacity.
- Content: "Lifer" wordmark (30px, 700), tagline ("Learn bird songs by ear"), italic detail line ("47 species · Pacific Northwest" — static placeholder until real data), Google sign-in button.
- Google sign-in button: white background, dark text, Google logo SVG, `border-radius: 8px`.

### StatsBar.svelte
- `display: flex; gap: 0.5rem`.
- Each stat: `background: var(--surface); border-radius: 8px; padding: 0.625rem 0.875rem; flex: 1; text-align: center; box-shadow: var(--shadow)`.
- Value: `var(--text)`, 20px, 700.
- Label: `var(--text-muted)`, 9px, uppercase, `letter-spacing: 0.06em`.

### Dashboard.svelte
- StatsBar.
- "Start Practice" CTA button: full width, `background: var(--accent)`, 15px, 600, `border-radius: 10px`, 13px vertical padding.
- Group note below CTA: `var(--text-secondary)`, 11px, centered. Shows top group name + due count.
- GroupList below that.

### GroupList.svelte
- Each item: `background: var(--surface); border-radius: 10px; padding: 0.75rem 0.875rem; box-shadow: var(--shadow)`.
- Group name: `var(--text)`, 14px, 600.
- Due count: `var(--text-muted)`, 11px.
- Practice button: `background: var(--accent-hover)`, small (`padding: 6px 12px`, 12px font).

### QuizCard.svelte
- Audio player: styled `<audio controls>` inside a `var(--surface)` container, full width, `border-radius: 8px`.
- Text input: `background: var(--surface); border: 1px solid var(--border); border-radius: 8px; padding: 0.688rem 0.875rem; color: var(--text); font-size: 14px`.
- Reveal button: `background: var(--surface); border: 1px solid var(--border); color: var(--text-secondary)`, full width, 14px, 600, `border-radius: 10px`.

### RevealCard.svelte
- Species photo: full width, `border-radius: 10px; max-height: 200px; object-fit: cover`.
- Common name: `var(--text)`, 16px, 700.
- Scientific name: `var(--text-muted)`, 13px, italic.
- "How well did you know it?" label: `var(--text-muted)`, 11px, uppercase.
- Confidence grid: `display: grid; grid-template-columns: repeat(4, 1fr); gap: 0.5rem`.
- Each button: fixed color (see token table above), 12px, 600, `border-radius: 8px`, `padding: 10px 4px`.

### Quiz.svelte
- StatsBar with session stats (remaining, reviewed, streak).
- QuizCard or RevealCard below.

---

## Files Changed

| File | Change |
|---|---|
| `frontend/index.html` | Add Inter font `<link>` tags |
| `frontend/src/app.css` | Create: reset, tokens, body defaults |
| `frontend/src/main.ts` | Import `./app.css` |
| `frontend/src/lib/theme.ts` | Create: `initTheme()`, `toggleTheme()`, `getTheme()` |
| `frontend/src/App.svelte` | Add top bar + theme toggle, call `initTheme()` |
| `frontend/src/views/Login.svelte` | Full atmospheric layout |
| `frontend/src/views/Dashboard.svelte` | Wire token-based styles |
| `frontend/src/views/Quiz.svelte` | Wire token-based styles |
| `frontend/src/components/StatsBar.svelte` | Token-based pill styles |
| `frontend/src/components/GroupList.svelte` | Token-based list styles |
| `frontend/src/components/QuizCard.svelte` | Token-based input + button styles |
| `frontend/src/components/RevealCard.svelte` | Photo, names, confidence button styles |

---

## Out of Scope

- Admin UI styling
- Animations / transitions (can add later)
- Image identification quiz mode
- Mobile-specific breakpoints beyond the 480px max-width column
