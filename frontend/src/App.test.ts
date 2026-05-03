import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen } from '@testing-library/svelte'
import { get } from 'svelte/store'
import { view } from './stores/view'
import { auth } from './stores/auth'
import App from './App.svelte'

beforeEach(() => {
  view.set('login')
  auth.set(null)
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe('App', () => {
  it('shows Login when /api/v1/me returns 401', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false }))
    render(App)
    await vi.waitFor(() => {
      expect(get(view)).toBe('login')
    })
    expect(screen.getByRole('link', { name: /sign in with google/i })).toBeInTheDocument()
  })

  it('shows Dashboard and sets auth when /api/v1/me returns 200', async () => {
    const user = { id: 1, email: 'test@example.com', name: 'Test User' }
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve(user) }))
    render(App)
    await vi.waitFor(() => {
      expect(get(view)).toBe('dashboard')
    })
    expect(get(auth)).toEqual(user)
    expect(screen.getByRole('button', { name: /start practice/i })).toBeInTheDocument()
  })

  it('shows Quiz when view is quiz', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false }))
    view.set('quiz')
    render(App)
    expect(document.querySelector('audio')).not.toBeNull()
  })
})
