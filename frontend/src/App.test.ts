import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen } from '@testing-library/svelte'
import { get } from 'svelte/store'
import { view } from './stores/view'
import { auth } from './stores/auth'
import App from './App.svelte'

const mockMatchMedia = (prefersDark = true) => {
  vi.stubGlobal('matchMedia', vi.fn().mockImplementation((query: string) => ({
    matches: query === '(prefers-color-scheme: dark)' ? prefersDark : !prefersDark,
    media: query,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
  })))
}

beforeEach(() => {
  view.set('login')
  auth.set(null)
  document.documentElement.removeAttribute('data-theme')
  mockMatchMedia()
})

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe('App', () => {
  it('shows Login when /api/v1/me returns 401', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false }))
    render(App)
    await vi.waitFor(() => {
      expect(screen.getByRole('link', { name: /sign in with google/i })).toBeInTheDocument()
    })
  })

  it('shows Dashboard and sets auth when /api/v1/me returns 200', async () => {
    const user = { id: 1, email: 'test@example.com', name: 'Test User' }
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve(user) }))
    render(App)
    await vi.waitFor(() => {
      expect(screen.getAllByRole('button', { name: /audio/i }).length).toBeGreaterThan(0)
    })
    expect(get(auth)).toEqual(user)
  })

  it('shows theme toggle when authenticated', async () => {
    const user = { id: 1, email: 'test@example.com', name: 'Test User' }
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve(user) }))
    render(App)
    await vi.waitFor(() => {
      expect(screen.getByRole('button', { name: /switch to .* mode/i })).toBeInTheDocument()
    })
  })
})
