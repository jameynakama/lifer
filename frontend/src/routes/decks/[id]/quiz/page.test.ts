import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/svelte'
import { goto } from '$app/navigation'
import { page } from '$app/state'
import QuizPage from './+page.svelte'

// WaveSurfer uses Web Audio API not available in jsdom
vi.mock('wavesurfer.js', () => ({
  default: {
    create: vi.fn(() => ({
      on: vi.fn((event: string, cb: () => void) => { if (event === 'ready') cb() }),
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
  page.url = new URL('http://localhost/decks/42/quiz?lane=audio')
  vi.mocked(goto).mockClear()
})

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe('Quiz page', () => {
  it('shows loading initially', () => {
    vi.stubGlobal('fetch', vi.fn().mockReturnValue(new Promise(() => {})))
    render(QuizPage)
    expect(screen.getByText(/loading/i)).toBeInTheDocument()
  })

  it('fetches deck species on mount', async () => {
    const fetchMock = makeFetch()
    vi.stubGlobal('fetch', fetchMock)
    render(QuizPage)
    await vi.waitFor(() => {
      const calls = fetchMock.mock.calls.map((c: unknown[]) => c[0] as string)
      expect(calls.some((url) => url.includes('/species'))).toBe(true)
    })
  })

  it('shows QuizCard with play button when a card is returned', async () => {
    vi.stubGlobal('fetch', makeFetch())
    render(QuizPage)
    await vi.waitFor(() => {
      expect(screen.getByRole('button', { name: /play/i })).toBeInTheDocument()
    })
  })

  it('shows all caught up when 204 is returned for next card', async () => {
    vi.stubGlobal('fetch', makeFetch({ status: 204 }))
    render(QuizPage)
    await vi.waitFor(() => {
      expect(screen.getByText(/all caught up/i)).toBeInTheDocument()
    })
  })

  it('navigates to deck detail when Back to deck is clicked', async () => {
    vi.stubGlobal('fetch', makeFetch({ status: 204 }))
    render(QuizPage)
    await vi.waitFor(() => screen.getByRole('button', { name: /back to deck/i }))
    await fireEvent.click(screen.getByRole('button', { name: /back to deck/i }))
    expect(goto).toHaveBeenCalledWith('/decks/42')
  })

  it('passes correct=true to RevealCard when selected species matches card', async () => {
    vi.stubGlobal('fetch', makeFetch())
    render(QuizPage)
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
    render(QuizPage)
    await vi.waitFor(() => screen.getByRole('button', { name: /i don't know/i }))
    await fireEvent.click(screen.getByRole('button', { name: /i don't know/i }))
    await vi.waitFor(() => {
      expect(screen.getByText(/✗/)).toBeInTheDocument()
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
    render(QuizPage)
    await vi.waitFor(() => screen.getByRole('button', { name: /i don't know/i }))
    await fireEvent.click(screen.getByRole('button', { name: /i don't know/i }))
    await vi.waitFor(() => screen.getByRole('button', { name: /next/i }))
    await fireEvent.click(screen.getByRole('button', { name: /next/i }))
    await vi.waitFor(() => {
      const posts = fetchMock.mock.calls.filter(
        (c: unknown[]) => (c[1] as RequestInit)?.method === 'POST'
      )
      expect(posts.length).toBe(1)
    })
  })
})
