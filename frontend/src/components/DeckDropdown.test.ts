import { describe, it, expect, vi, afterEach } from 'vitest'
import { screen, fireEvent } from '@testing-library/svelte'
import { renderWithClient } from '../test-utils'
import DeckDropdown from './DeckDropdown.svelte'

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

const decks = [
  { id: 1, name: 'My Warblers', audio_due: 0, image_due: 0 },
  { id: 2, name: 'Sparrows', audio_due: 0, image_due: 0 },
]

/** Species sonspa is a member of deck 1 only. */
function makeFetch() {
  return vi.fn().mockImplementation((url: string, opts?: RequestInit) => {
    const method = opts?.method ?? 'GET'
    if (method === 'GET' && url.includes('/species/')) {
      return Promise.resolve({ ok: true, json: () => Promise.resolve({ deck_ids: [1] }) })
    }
    if (method === 'GET') {
      return Promise.resolve({
        ok: true,
        json: () => Promise.resolve({ decks, next_due_at: null }),
      })
    }
    if (method === 'POST' && url.endsWith('/api/v1/decks')) {
      return Promise.resolve({
        ok: true,
        json: () => Promise.resolve({ id: 99, name: 'New Deck' }),
      })
    }
    if (method === 'POST') {
      return Promise.resolve({ ok: true, json: () => Promise.resolve({}) })
    }
    return Promise.resolve({ ok: true, status: 204 })
  })
}

describe('DeckDropdown', () => {
  it('lists decks with membership reflected in checkboxes', async () => {
    vi.stubGlobal('fetch', makeFetch())
    renderWithClient(DeckDropdown, { ebird_code: 'sonspa', onClose: vi.fn() })
    await vi.waitFor(() => screen.getByText(/my warblers/i))
    const boxes = screen.getAllByRole('checkbox') as HTMLInputElement[]
    expect(boxes).toHaveLength(2)
    expect(boxes[0].checked).toBe(true)
    expect(boxes[1].checked).toBe(false)
  })

  it('checking a non-member deck POSTs the species to it', async () => {
    const fetchMock = makeFetch()
    vi.stubGlobal('fetch', fetchMock)
    renderWithClient(DeckDropdown, { ebird_code: 'sonspa', onClose: vi.fn() })
    await vi.waitFor(() => screen.getByText(/sparrows/i))
    await fireEvent.click(screen.getAllByRole('checkbox')[1])
    await vi.waitFor(() => {
      const posts = fetchMock.mock.calls.filter(
        (c: unknown[]) => (c[1] as RequestInit | undefined)?.method === 'POST',
      )
      expect(posts.some((c: unknown[]) => (c[0] as string).includes('/decks/2/species'))).toBe(true)
    })
  })

  it('unchecking a member deck DELETEs the species from it', async () => {
    const fetchMock = makeFetch()
    vi.stubGlobal('fetch', fetchMock)
    renderWithClient(DeckDropdown, { ebird_code: 'sonspa', onClose: vi.fn() })
    await vi.waitFor(() => screen.getByText(/my warblers/i))
    await fireEvent.click(screen.getAllByRole('checkbox')[0])
    await vi.waitFor(() => {
      const deletes = fetchMock.mock.calls.filter(
        (c: unknown[]) => (c[1] as RequestInit | undefined)?.method === 'DELETE',
      )
      expect(
        deletes.some((c: unknown[]) => (c[0] as string).includes('/decks/1/species/sonspa')),
      ).toBe(true)
    })
  })

  it('creating a deck POSTs the deck, adds the species, and clears the input', async () => {
    const fetchMock = makeFetch()
    vi.stubGlobal('fetch', fetchMock)
    renderWithClient(DeckDropdown, { ebird_code: 'sonspa', onClose: vi.fn() })
    const input = await vi.waitFor(() => screen.getByPlaceholderText(/new deck name/i))
    await fireEvent.input(input, { target: { value: 'New Deck' } })
    await fireEvent.click(screen.getByRole('button', { name: /create deck/i }))
    await vi.waitFor(() => {
      const posts = fetchMock.mock.calls.filter(
        (c: unknown[]) => (c[1] as RequestInit | undefined)?.method === 'POST',
      )
      expect(posts.some((c: unknown[]) => (c[0] as string).endsWith('/api/v1/decks'))).toBe(true)
      expect(posts.some((c: unknown[]) => (c[0] as string).includes('/decks/99/species'))).toBe(
        true,
      )
      expect((input as HTMLInputElement).value).toBe('')
    })
  })

  it('pressing Escape in the input calls onClose', async () => {
    const onClose = vi.fn()
    vi.stubGlobal('fetch', makeFetch())
    renderWithClient(DeckDropdown, { ebird_code: 'sonspa', onClose })
    const input = await vi.waitFor(() => screen.getByPlaceholderText(/new deck name/i))
    await fireEvent.keyDown(input, { key: 'Escape' })
    expect(onClose).toHaveBeenCalledOnce()
  })
})
