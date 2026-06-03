import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen } from '@testing-library/svelte'
import { auth } from '$stores/auth'
import Page from './+page.svelte'

beforeEach(() => {
  auth.set({ id: 1, email: 'admin@example.com', name: 'Admin', is_admin: true })
})

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe('Admin species page', () => {
  it('renders a search form', () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ results: [], count: 0, next: null, previous: null }),
    }))
    render(Page)
    expect(screen.getByRole('textbox')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /search/i })).toBeInTheDocument()
  })

  it('renders species results as links', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: () =>
        Promise.resolve({
          results: [
            {
              ebird_code: 'sonspa',
              common_name: 'Song Sparrow',
              scientific_name: 'Melospiza melodia',
              image_url: null,
            },
          ],
          count: 1,
          next: null,
          previous: null,
        }),
    }))
    render(Page)
    await vi.waitFor(() => {
      expect(screen.getByRole('link', { name: /song sparrow/i })).toBeInTheDocument()
    })
  })

  it('shows Next link when next is not null', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: () =>
        Promise.resolve({
          results: [
            {
              ebird_code: 'sonspa',
              common_name: 'Song Sparrow',
              scientific_name: 'Melospiza melodia',
              image_url: null,
            },
          ],
          count: 50,
          next: 'http://localhost/?offset=25',
          previous: null,
        }),
    }))
    render(Page)
    await vi.waitFor(() => {
      expect(screen.getByRole('link', { name: /next/i })).toBeInTheDocument()
    })
  })

  it('does not show Previous link when previous is null', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: () =>
        Promise.resolve({
          results: [
            {
              ebird_code: 'sonspa',
              common_name: 'Song Sparrow',
              scientific_name: 'Melospiza melodia',
              image_url: null,
            },
          ],
          count: 50,
          next: 'http://localhost/?offset=25',
          previous: null,
        }),
    }))
    render(Page)
    await vi.waitFor(() => {
      expect(screen.queryByRole('link', { name: /previous/i })).not.toBeInTheDocument()
    })
  })

  it('handles fetch errors and displays error message', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: false,
      status: 500,
      json: () => Promise.resolve({ error: 'Internal server error' }),
    }))
    render(Page)
    await vi.waitFor(() => {
      expect(screen.getByText(/failed to load: 500/i)).toBeInTheDocument()
    })
  })
})
