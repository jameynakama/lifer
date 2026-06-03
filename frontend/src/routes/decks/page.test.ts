import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/svelte'
import { goto } from '$app/navigation'
import DecksPage from './+page.svelte'

const decks = [
  { id: 1, name: 'My Warblers', audio_due: 2, image_due: 0 },
]

beforeEach(() => {
  vi.mocked(goto).mockClear()
})

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe('Decks page', () => {
  it('renders deck list from API', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true, json: () => Promise.resolve({ decks, next_due_at: null }),
    }))
    render(DecksPage)
    await vi.waitFor(() => {
      expect(screen.getAllByText(/my warblers/i).length).toBeGreaterThan(0)
    })
  })

  it('creates a deck when form submitted', async () => {
    let postCalled = false
    vi.stubGlobal('fetch', vi.fn().mockImplementation((url: string, opts?: RequestInit) => {
      if (opts?.method === 'POST') {
        postCalled = true
        return Promise.resolve({ ok: true, json: () => Promise.resolve({ id: 2, name: 'New Deck', audio_due: 0, image_due: 0 }) })
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve({ decks, next_due_at: null }) })
    }))
    render(DecksPage)
    await vi.waitFor(() => screen.getByPlaceholderText(/deck name/i))
    await fireEvent.input(screen.getByPlaceholderText(/deck name/i), { target: { value: 'New Deck' } })
    await fireEvent.click(screen.getByRole('button', { name: /create/i }))
    await vi.waitFor(() => { expect(postCalled).toBe(true) })
    await vi.waitFor(() => {
      expect(screen.queryByText(/new deck/i)).not.toBeNull()
    })
  })

  it('navigates to deck detail when name clicked', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true, json: () => Promise.resolve({ decks, next_due_at: null }),
    }))
    render(DecksPage)
    await vi.waitFor(() => screen.getByRole('link', { name: /my warblers/i }))
    // Link navigates via href, not goto -- just verify the link exists with correct href
    const link = screen.getByRole('link', { name: /my warblers/i })
    expect(link).toHaveAttribute('href', '/decks/1')
  })

  it('deletes a deck when delete button clicked', async () => {
    let deleteCalled = false
    vi.stubGlobal('confirm', vi.fn().mockReturnValue(true))
    vi.stubGlobal('fetch', vi.fn().mockImplementation((url: string, opts?: RequestInit) => {
      if (opts?.method === 'DELETE') {
        deleteCalled = true
        return Promise.resolve({ ok: true })
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve({ decks, next_due_at: null }) })
    }))
    render(DecksPage)
    await vi.waitFor(() => screen.getByRole('button', { name: /delete/i }))
    await fireEvent.click(screen.getByRole('button', { name: /delete/i }))
    await vi.waitFor(() => { expect(deleteCalled).toBe(true) })
    await vi.waitFor(() => {
      expect(screen.queryByText(/my warblers/i)).toBeNull()
    })
  })

  it('shows Free Practice toggle button', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true, json: () => Promise.resolve({ decks, next_due_at: null }),
    }))
    render(DecksPage)
    await vi.waitFor(() => {
      expect(screen.getByRole('button', { name: /free practice/i })).toBeInTheDocument()
    })
  })

  it('shows practice banner when Free Practice toggled on', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true, json: () => Promise.resolve({ decks, next_due_at: null }),
    }))
    render(DecksPage)
    await vi.waitFor(() => screen.getByRole('button', { name: /free practice/i }))
    await fireEvent.click(screen.getByRole('button', { name: /free practice/i }))
    await vi.waitFor(() => {
      expect(screen.getByText(/answers won't affect/i)).toBeInTheDocument()
    })
  })

  it('shows practice audio quick button in practice mode', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true, json: () => Promise.resolve({ decks, next_due_at: null }),
    }))
    render(DecksPage)
    await vi.waitFor(() => screen.getByRole('button', { name: /free practice/i }))
    await fireEvent.click(screen.getByRole('button', { name: /free practice/i }))
    await vi.waitFor(() => {
      expect(screen.getByRole('button', { name: /▶ audio/i })).toBeInTheDocument()
    })
  })

  it('quick audio button navigates to practice page', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true, json: () => Promise.resolve({ decks, next_due_at: null }),
    }))
    render(DecksPage)
    await vi.waitFor(() => screen.getByRole('button', { name: /free practice/i }))
    await fireEvent.click(screen.getByRole('button', { name: /free practice/i }))
    await vi.waitFor(() => screen.getByRole('button', { name: /▶ audio/i }))
    await fireEvent.click(screen.getByRole('button', { name: /▶ audio/i }))
    expect(goto).toHaveBeenCalledWith('/decks/1/practice?lane=audio')
  })

  it('hides due badges in practice mode', async () => {
    const deckWithDue = [{ id: 1, name: 'My Warblers', audio_due: 2, image_due: 1 }]
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true, json: () => Promise.resolve({ decks: deckWithDue, next_due_at: null }),
    }))
    render(DecksPage)
    // Due buttons visible normally
    await vi.waitFor(() => screen.getByRole('button', { name: /🔊 audio/i }))
    // Toggle practice mode
    await fireEvent.click(screen.getByRole('button', { name: /free practice/i }))
    await vi.waitFor(() => {
      expect(screen.queryByRole('button', { name: /🔊 audio/i })).toBeNull()
    })
  })
})
