import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen } from '@testing-library/svelte'
import { get } from 'svelte/store'
import { auth } from '$stores/auth'
import Layout from './+layout.svelte'

const mockMatchMedia = (prefersDark = true) => {
  vi.stubGlobal(
    'matchMedia',
    vi.fn().mockImplementation((query: string) => ({
      matches: query === '(prefers-color-scheme: dark)' ? prefersDark : !prefersDark,
      media: query,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    })),
  )
}

beforeEach(() => {
  auth.set(null)
  document.documentElement.removeAttribute('data-theme')
  mockMatchMedia()
})

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe('Layout', () => {
  it('shows Login when /api/v1/me returns 401', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false }))
    render(Layout)
    await vi.waitFor(() => {
      expect(screen.getByRole('link', { name: /sign in with google/i })).toBeInTheDocument()
    })
  })

  it('shows app shell and sets auth when /api/v1/me returns 200', async () => {
    const user = { id: 1, email: 'test@example.com', name: 'Test User' }
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve(user) }),
    )
    render(Layout)
    await vi.waitFor(() => {
      expect(screen.getByText(/FlockDeck/)).toBeInTheDocument()
    })
    expect(get(auth)).toEqual(user)
  })

  it('shows theme toggle when authenticated', async () => {
    const user = { id: 1, email: 'test@example.com', name: 'Test User' }
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve(user) }),
    )
    render(Layout)
    await vi.waitFor(() => {
      expect(screen.getByRole('button', { name: /switch to .* mode/i })).toBeInTheDocument()
    })
  })

  it('shows About link when authenticated', async () => {
    const user = { id: 1, email: 'test@example.com', name: 'Test User' }
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve(user) }),
    )
    render(Layout)
    await vi.waitFor(() => {
      expect(screen.getByRole('link', { name: /about/i })).toBeInTheDocument()
    })
  })

  it('shows Admin link when user is admin', async () => {
    const user = { id: 1, email: 'test@example.com', name: 'Test User', is_admin: true }
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve(user) }),
    )
    render(Layout)
    await vi.waitFor(() => {
      expect(screen.getByRole('link', { name: /admin/i })).toBeInTheDocument()
    })
  })

  it('hides Admin link for non-admin users', async () => {
    const user = { id: 1, email: 'test@example.com', name: 'Test User', is_admin: false }
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve(user) }),
    )
    render(Layout)
    await vi.waitFor(() => {
      expect(screen.getByText(/FlockDeck/)).toBeInTheDocument()
    })
    expect(screen.queryByRole('link', { name: /admin/i })).not.toBeInTheDocument()
  })
})
