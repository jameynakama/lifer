import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { screen, fireEvent } from '@testing-library/svelte'
import { renderWithClient } from '../../../../test-utils'
import { goto } from '$app/navigation'
import { page } from '$app/state'
import QuizPage from './+page.svelte'

// WaveSurfer uses Web Audio API not available in jsdom
vi.mock('wavesurfer.js', () => ({
  default: {
    create: vi.fn(() => ({
      on: vi.fn((event: string, cb: () => void) => {
        if (event === 'ready') cb()
      }),
      playPause: vi.fn(),
      pause: vi.fn(),
      destroy: vi.fn(),
    })),
  },
}))

const card = {
  ebird_code: 'sonspa',
  common_name: 'Song Sparrow',
  scientific_name: 'Melospiza melodia',
  media_url: '/recordings/song-sparrow.mp3',
  photo_url: '/photos/song-sparrow.jpg',
  lane: 'audio',
  recording_type: 'song',
  due_remaining: 5,
  media_id: 'XC1',
}

const species = [
  { ebird_code: 'sonspa', common_name: 'Song Sparrow', scientific_name: 'Melospiza melodia' },
  { ebird_code: 'foxspa', common_name: 'Fox Sparrow', scientific_name: 'Passerella iliaca' },
]

function makeFetch(opts: { card?: object | null; status?: number } = {}) {
  return vi.fn().mockImplementation((url: string) => {
    if (url.includes('/species')) {
      return Promise.resolve({ ok: true, json: () => Promise.resolve(species) })
    }
    if (opts.status === 204 || opts.card === null) {
      return Promise.resolve({ ok: true, status: 204 })
    }
    return Promise.resolve({
      ok: true,
      status: 200,
      json: () => Promise.resolve(opts.card ?? card),
    })
  })
}

beforeEach(() => {
  page.params = { id: '42' }
  page.url = new URL('http://localhost/decks/42/quiz?lane=audio') as typeof page.url
  vi.mocked(goto).mockClear()
})

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe('Quiz page', () => {
  it('shows loading initially', () => {
    vi.stubGlobal('fetch', vi.fn().mockReturnValue(new Promise(() => {})))
    renderWithClient(QuizPage)
    expect(screen.getByText(/loading/i)).toBeInTheDocument()
  })

  it('fetches deck species on mount', async () => {
    const fetchMock = makeFetch()
    vi.stubGlobal('fetch', fetchMock)
    renderWithClient(QuizPage)
    await vi.waitFor(() => {
      const calls = fetchMock.mock.calls.map((c: unknown[]) => c[0] as string)
      expect(calls.some((url) => url.includes('/species'))).toBe(true)
    })
  })

  it('pins the session with a stable due_before on every /next call', async () => {
    const fetchMock = makeFetch()
    vi.stubGlobal('fetch', fetchMock)
    renderWithClient(QuizPage)
    await vi.waitFor(() => {
      const calls = fetchMock.mock.calls.map((c: unknown[]) => c[0] as string)
      expect(calls.some((url) => url.includes('/next'))).toBe(true)
    })

    const nextCalls = fetchMock.mock.calls
      .map((c: unknown[]) => c[0] as string)
      .filter((url) => url.includes('/next'))
    for (const url of nextCalls) {
      // RFC3339 timestamp pinned at session start so cards FSRS re-dues
      // mid-session never repeat.
      expect(url).toMatch(/due_before=[^&]*\d{4}-\d{2}-\d{2}T/)
    }
    expect(new Set(nextCalls.map((u) => u.split('due_before=')[1])).size).toBe(1)
  })

  it('fetches /next exactly once per mount, even when the clock ticks', async () => {
    // Strictly-increasing timestamps: if the mount $effect re-runs (the old
    // sessionStart self-dependency bug), every re-run mints a distinct
    // due_before and fires a duplicate fetch -- this test fails loudly
    // instead of only when CI straddles a millisecond boundary.
    let tick = 0
    vi.spyOn(Date.prototype, 'toISOString').mockImplementation(
      () => `2026-01-01T00:00:00.${String(tick++).padStart(3, '0')}Z`,
    )
    const fetchMock = makeFetch()
    vi.stubGlobal('fetch', fetchMock)
    renderWithClient(QuizPage)
    await vi.waitFor(() => screen.getByRole('button', { name: /i don't know/i }))
    // settle any trailing effect re-runs before counting
    await new Promise((r) => setTimeout(r, 25))
    const nextCalls = fetchMock.mock.calls
      .map((c: unknown[]) => c[0] as string)
      .filter((url) => url.includes('/next'))
    expect(nextCalls).toHaveLength(1)
  })

  it('shows QuizCard with play button when a card is returned', async () => {
    vi.stubGlobal('fetch', makeFetch())
    renderWithClient(QuizPage)
    await vi.waitFor(() => {
      expect(screen.getByRole('button', { name: /play/i })).toBeInTheDocument()
    })
  })

  it('shows all caught up when 204 is returned for next card', async () => {
    vi.stubGlobal('fetch', makeFetch({ status: 204 }))
    renderWithClient(QuizPage)
    await vi.waitFor(() => {
      expect(screen.getByText(/all caught up/i)).toBeInTheDocument()
    })
  })

  it('navigates to deck detail when Back to deck is clicked', async () => {
    vi.stubGlobal('fetch', makeFetch({ status: 204 }))
    renderWithClient(QuizPage)
    await vi.waitFor(() => screen.getByRole('button', { name: /back to deck/i }))
    await fireEvent.click(screen.getByRole('button', { name: /back to deck/i }))
    expect(goto).toHaveBeenCalledWith('/decks/42')
  })

  it('passes correct=true to RevealCard when selected species matches card', async () => {
    vi.stubGlobal('fetch', makeFetch())
    renderWithClient(QuizPage)
    // Wait for quiz card to load
    await vi.waitFor(() => screen.getByRole('combobox'))
    // Type and select the correct species (id=99 = Song Sparrow = card.species_id)
    await fireEvent.input(screen.getByRole('combobox'), { target: { value: 'song' } })
    await vi.waitFor(() => screen.getByRole('option', { name: /song sparrow/i }))
    await fireEvent.mouseDown(screen.getByRole('option', { name: /song sparrow/i }))
    await fireEvent.click(screen.getByRole('button', { name: /reveal answer/i }))
    // RevealCard should show correct banner (✓)
    await vi.waitFor(() => {
      expect(screen.getByText(/✓/)).toBeInTheDocument()
    })
  })

  it("passes correct=false to RevealCard when I don't know is clicked", async () => {
    vi.stubGlobal('fetch', makeFetch())
    renderWithClient(QuizPage)
    await vi.waitFor(() => screen.getByRole('button', { name: /i don't know/i }))
    await fireEvent.click(screen.getByRole('button', { name: /i don't know/i }))
    await vi.waitFor(() => {
      expect(screen.getByText(/✗/)).toBeInTheDocument()
    })
  })

  it('shows Retry on load failure and refetches when clicked', async () => {
    let fail = true
    const fetchMock = vi.fn().mockImplementation((url: string) => {
      if (url.includes('/species')) {
        return Promise.resolve({ ok: true, json: () => Promise.resolve(species) })
      }
      if (fail) return Promise.reject(new Error('Network error'))
      return Promise.resolve({ ok: true, status: 200, json: () => Promise.resolve(card) })
    })
    vi.stubGlobal('fetch', fetchMock)
    renderWithClient(QuizPage)
    await vi.waitFor(() => screen.getByText(/failed to load/i))
    fail = false
    await fireEvent.click(screen.getByRole('button', { name: /retry/i }))
    await vi.waitFor(() => {
      expect(screen.getByRole('button', { name: /play/i })).toBeInTheDocument()
    })
  })

  it('clicking Next POSTs rating and advances to next card', async () => {
    const fetchMock = vi.fn().mockImplementation((url: string, opts?: RequestInit) => {
      if (url.includes('/species')) {
        return Promise.resolve({ ok: true, json: () => Promise.resolve(species) })
      }
      if (opts?.method === 'POST') {
        return Promise.resolve({ ok: true, json: () => Promise.resolve({}) })
      }
      return Promise.resolve({ ok: true, status: 200, json: () => Promise.resolve(card) })
    })
    vi.stubGlobal('fetch', fetchMock)
    renderWithClient(QuizPage)
    await vi.waitFor(() => screen.getByRole('button', { name: /i don't know/i }))
    await fireEvent.click(screen.getByRole('button', { name: /i don't know/i }))
    await vi.waitFor(() => screen.getByRole('button', { name: /next/i }))
    await fireEvent.click(screen.getByRole('button', { name: /next/i }))
    await vi.waitFor(() => {
      const posts = fetchMock.mock.calls.filter(
        (c: unknown[]) => (c[1] as RequestInit)?.method === 'POST',
      )
      expect(posts.length).toBe(1)
    })
  })

  it('POSTs the guessed species and media id with the rating', async () => {
    const fetchMock = vi.fn().mockImplementation((url: string, opts?: RequestInit) => {
      if (url.includes('/species')) {
        return Promise.resolve({ ok: true, json: () => Promise.resolve(species) })
      }
      if (opts?.method === 'POST') {
        return Promise.resolve({ ok: true, json: () => Promise.resolve({}) })
      }
      return Promise.resolve({ ok: true, status: 200, json: () => Promise.resolve(card) })
    })
    vi.stubGlobal('fetch', fetchMock)
    renderWithClient(QuizPage)
    await vi.waitFor(() => screen.getByRole('combobox'))
    await fireEvent.input(screen.getByRole('combobox'), { target: { value: 'fox' } })
    await vi.waitFor(() => screen.getByRole('option', { name: /fox sparrow/i }))
    await fireEvent.mouseDown(screen.getByRole('option', { name: /fox sparrow/i }))
    await fireEvent.click(screen.getByRole('button', { name: /reveal answer/i }))
    await vi.waitFor(() => screen.getByRole('button', { name: /next/i }))
    await fireEvent.click(screen.getByRole('button', { name: /next/i }))
    await vi.waitFor(() => {
      const post = fetchMock.mock.calls.find(
        (c: unknown[]) => (c[1] as RequestInit | undefined)?.method === 'POST',
      )
      const body = JSON.parse((post![1] as RequestInit).body as string)
      expect(body.guessed_species_code).toBe('foxspa')
      expect(body.media_id).toBe('XC1')
      expect(body.rating).toBe(1)
    })
  })

  it('invalidates deck and stats queries after rating', async () => {
    // Due counts and review stats change server-side on every /rate; the
    // cached ['decks'] and ['stats', *] queries must be marked stale or
    // home//decks//stats show pre-quiz numbers until staleTime expires.
    const fetchMock = vi.fn().mockImplementation((url: string, opts?: RequestInit) => {
      if (url.includes('/species')) {
        return Promise.resolve({ ok: true, json: () => Promise.resolve(species) })
      }
      if (opts?.method === 'POST') {
        return Promise.resolve({ ok: true, json: () => Promise.resolve({}) })
      }
      return Promise.resolve({ ok: true, status: 200, json: () => Promise.resolve(card) })
    })
    vi.stubGlobal('fetch', fetchMock)
    const { client } = renderWithClient(QuizPage)
    const invalidate = vi.spyOn(client, 'invalidateQueries')
    await vi.waitFor(() => screen.getByRole('button', { name: /i don't know/i }))
    await fireEvent.click(screen.getByRole('button', { name: /i don't know/i }))
    await vi.waitFor(() => screen.getByRole('button', { name: /next/i }))
    await fireEvent.click(screen.getByRole('button', { name: /next/i }))
    await vi.waitFor(() => {
      expect(invalidate).toHaveBeenCalledWith({ queryKey: ['decks'] })
      expect(invalidate).toHaveBeenCalledWith({ queryKey: ['stats'] })
    })
  })

  it("POSTs a null guess for I don't know", async () => {
    const fetchMock = vi.fn().mockImplementation((url: string, opts?: RequestInit) => {
      if (url.includes('/species')) {
        return Promise.resolve({ ok: true, json: () => Promise.resolve(species) })
      }
      if (opts?.method === 'POST') {
        return Promise.resolve({ ok: true, json: () => Promise.resolve({}) })
      }
      return Promise.resolve({ ok: true, status: 200, json: () => Promise.resolve(card) })
    })
    vi.stubGlobal('fetch', fetchMock)
    renderWithClient(QuizPage)
    await vi.waitFor(() => screen.getByRole('button', { name: /i don't know/i }))
    await fireEvent.click(screen.getByRole('button', { name: /i don't know/i }))
    await vi.waitFor(() => screen.getByRole('button', { name: /next/i }))
    await fireEvent.click(screen.getByRole('button', { name: /next/i }))
    await vi.waitFor(() => {
      const post = fetchMock.mock.calls.find(
        (c: unknown[]) => (c[1] as RequestInit | undefined)?.method === 'POST',
      )
      const body = JSON.parse((post![1] as RequestInit).body as string)
      expect(body.guessed_species_code).toBeNull()
    })
  })
})
