import { describe, it, expect, vi, afterEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/svelte'
import AdminDecksPage from './+page.svelte'

const presets = [
  { id: 1, name: 'Confusing Woodpeckers', description: 'Rattle calls', species_count: 5 },
]

const userDecks = [
  { id: 10, name: 'My Warblers', description: '', owner_id: 1, owner_name: 'Alice', owner_email: 'alice@example.com', species_count: 12 },
]

function makeFetchWithUserDecks() {
  return vi.fn().mockImplementation((url: string, opts?: RequestInit) => {
    if (url === '/api/v1/admin/decks' && !opts?.method) {
      return Promise.resolve({ ok: true, json: () => Promise.resolve(userDecks) })
    }
    if (opts?.method === 'DELETE') return Promise.resolve({ ok: true })
    if (opts?.method === 'POST') {
      return Promise.resolve({ ok: true, json: () => Promise.resolve({ id: 99, name: 'New Preset', description: '' }) })
    }
    return Promise.resolve({ ok: true, json: () => Promise.resolve(presets) })
  })
}

function makeFetch(overrides: Record<string, unknown> = {}) {
  return vi.fn().mockImplementation((url: string, opts?: RequestInit) => {
    if (opts?.method === 'DELETE') return Promise.resolve({ ok: true })
    if (opts?.method === 'POST') {
      return Promise.resolve({ ok: true, json: () => Promise.resolve({ id: 99, name: 'New Preset', description: '' }) })
    }
    if (opts?.method === 'PATCH') {
      return Promise.resolve({ ok: true, json: () => Promise.resolve({ id: 1, name: 'Updated', description: '' }) })
    }
    // user decks endpoint -- return empty list so it doesn't collide with preset data
    if (url === '/api/v1/admin/decks') {
      return Promise.resolve({ ok: true, json: () => Promise.resolve([]) })
    }
    if ((overrides.getPresets as boolean) === false) {
      return Promise.resolve({ ok: true, json: () => Promise.resolve([]) })
    }
    return Promise.resolve({ ok: true, json: () => Promise.resolve(presets) })
  })
}

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe('Admin decks page', () => {
  it('lists preset decks from API', async () => {
    vi.stubGlobal('fetch', makeFetch())
    render(AdminDecksPage)
    await vi.waitFor(() => {
      expect(screen.getByText(/confusing woodpeckers/i)).toBeInTheDocument()
      expect(screen.getByText(/5 species/i)).toBeInTheDocument()
    })
  })

  it('creates a preset deck when form submitted', async () => {
    let postBody: Record<string, unknown> | null = null
    vi.stubGlobal('fetch', vi.fn().mockImplementation((url: string, opts?: RequestInit) => {
      if (opts?.method === 'POST') {
        postBody = JSON.parse(opts.body as string)
        return Promise.resolve({ ok: true, json: () => Promise.resolve({ id: 99, name: 'New Preset', description: '' }) })
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve(presets) })
    }))
    render(AdminDecksPage)
    await vi.waitFor(() => screen.getByPlaceholderText(/deck name/i))
    await fireEvent.input(screen.getByPlaceholderText(/deck name/i), { target: { value: 'New Preset' } })
    await fireEvent.click(screen.getByRole('button', { name: /create/i }))
    await vi.waitFor(() => { expect(postBody?.name).toBe('New Preset') })
  })

  it('shows a link to manage species on each preset deck', async () => {
    vi.stubGlobal('fetch', makeFetch())
    render(AdminDecksPage)
    await vi.waitFor(() => screen.getByRole('link', { name: /manage species/i }))
    const link = screen.getByRole('link', { name: /manage species/i })
    expect(link).toHaveAttribute('href', '/decks/1')
  })

  it('lists user decks with owner info', async () => {
    vi.stubGlobal('fetch', makeFetchWithUserDecks())
    render(AdminDecksPage)
    await vi.waitFor(() => {
      expect(screen.getByText(/my warblers/i)).toBeInTheDocument()
      expect(screen.getByText(/alice@example\.com/i)).toBeInTheDocument()
      expect(screen.getByText(/12 species/i)).toBeInTheDocument()
    })
  })

  it('links each user deck to its detail page', async () => {
    vi.stubGlobal('fetch', makeFetchWithUserDecks())
    render(AdminDecksPage)
    await vi.waitFor(() => screen.getByRole('link', { name: /my warblers/i }))
    const link = screen.getByRole('link', { name: /my warblers/i })
    expect(link).toHaveAttribute('href', '/admin/decks/10')
  })

  it('deletes a preset deck on confirm', async () => {
    let deleteCalled = false
    vi.stubGlobal('confirm', vi.fn().mockReturnValue(true))
    vi.stubGlobal('fetch', vi.fn().mockImplementation((url: string, opts?: RequestInit) => {
      if (opts?.method === 'DELETE') {
        deleteCalled = true
        return Promise.resolve({ ok: true })
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve(presets) })
    }))
    render(AdminDecksPage)
    await vi.waitFor(() => screen.getByRole('button', { name: /delete/i }))
    await fireEvent.click(screen.getByRole('button', { name: /delete/i }))
    await vi.waitFor(() => { expect(deleteCalled).toBe(true) })
  })
})
