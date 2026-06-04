import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/svelte'
import AdminDecksPage from './+page.svelte'

const presets = [
  { id: 1, name: 'Confusing Woodpeckers', description: 'Rattle calls', species_count: 5 },
]

function makeFetch(overrides: Record<string, unknown> = {}) {
  return vi.fn().mockImplementation((url: string, opts?: RequestInit) => {
    if (opts?.method === 'DELETE') return Promise.resolve({ ok: true })
    if (opts?.method === 'POST') {
      return Promise.resolve({ ok: true, json: () => Promise.resolve({ id: 99, name: 'New Preset', description: '' }) })
    }
    if (opts?.method === 'PATCH') {
      return Promise.resolve({ ok: true, json: () => Promise.resolve({ id: 1, name: 'Updated', description: '' }) })
    }
    if ((overrides.getPresets as boolean) === false) {
      return Promise.resolve({ ok: true, json: () => Promise.resolve([]) })
    }
    return Promise.resolve({ ok: true, json: () => Promise.resolve(presets) })
  })
}

beforeEach(() => {})
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
