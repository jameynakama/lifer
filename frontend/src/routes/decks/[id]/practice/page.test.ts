import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/svelte'
import { goto } from '$app/navigation'
import { page } from '$app/state'
import PracticePage from './+page.svelte'

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

const cards = [
  {
    ebird_code: 'sonspa',
    common_name: 'Song Sparrow',
    scientific_name: 'Melospiza melodia',
    media_url: '/recordings/song-sparrow.mp3',
    photo_url: '/photos/song-sparrow.jpg',
    lane: 'audio',
  },
  {
    ebird_code: 'foxspa',
    common_name: 'Fox Sparrow',
    scientific_name: 'Passerella iliaca',
    media_url: '/recordings/fox-sparrow.mp3',
    photo_url: '/photos/fox-sparrow.jpg',
    lane: 'audio',
  },
]

function makeFetch(data: object[] = cards) {
  return vi.fn().mockResolvedValue({
    ok: true,
    status: 200,
    json: () => Promise.resolve(data),
  })
}

beforeEach(() => {
  page.params = { id: '42' }
  page.url = new URL('http://localhost/decks/42/practice?lane=audio') as typeof page.url
  vi.mocked(goto).mockClear()
})

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe('Practice page', () => {
  it('shows loading initially', () => {
    vi.stubGlobal('fetch', vi.fn().mockReturnValue(new Promise(() => {})))
    render(PracticePage)
    expect(screen.getByText(/loading/i)).toBeInTheDocument()
  })

  it('fetches practice cards from /practice endpoint on mount', async () => {
    const fetchMock = makeFetch()
    vi.stubGlobal('fetch', fetchMock)
    render(PracticePage)
    await vi.waitFor(() => {
      const calls = fetchMock.mock.calls.map((c: unknown[]) => c[0] as string)
      expect(calls.some((url) => url.includes('/practice'))).toBe(true)
    })
  })

  it('does NOT call /species endpoint separately', async () => {
    const fetchMock = makeFetch()
    vi.stubGlobal('fetch', fetchMock)
    render(PracticePage)
    await vi.waitFor(() => screen.getByRole('button', { name: /play/i }))
    const calls = fetchMock.mock.calls.map((c: unknown[]) => c[0] as string)
    expect(calls.every((url) => !url.includes('/species'))).toBe(true)
  })

  it('shows a card after loading', async () => {
    vi.stubGlobal('fetch', makeFetch())
    render(PracticePage)
    await vi.waitFor(() => {
      expect(screen.getByRole('button', { name: /play/i })).toBeInTheDocument()
    })
  })

  it('shows Practiced: 1 / 2 stat initially', async () => {
    vi.stubGlobal('fetch', makeFetch())
    render(PracticePage)
    await vi.waitFor(() => {
      expect(screen.getByText(/practiced/i)).toBeInTheDocument()
      expect(screen.getByText(/1 \/ 2/i)).toBeInTheDocument()
    })
  })

  it('does NOT POST to /rate when Next is clicked', async () => {
    const fetchMock = makeFetch()
    vi.stubGlobal('fetch', fetchMock)
    render(PracticePage)
    await vi.waitFor(() => screen.getByRole('button', { name: /i don't know/i }))
    await fireEvent.click(screen.getByRole('button', { name: /i don't know/i }))
    await vi.waitFor(() => screen.getByRole('button', { name: /next/i }))
    await fireEvent.click(screen.getByRole('button', { name: /next/i }))
    await vi.waitFor(() => {
      const posts = fetchMock.mock.calls.filter(
        (c: unknown[]) => (c[1] as RequestInit | undefined)?.method === 'POST'
      )
      expect(posts.length).toBe(0)
    })
  })

  it('shows done screen with Practice Again and Back to Deck after last card', async () => {
    // Single-card deck so done triggers after one Next
    const singleCard = [cards[0]]
    vi.stubGlobal('fetch', makeFetch(singleCard))
    render(PracticePage)
    await vi.waitFor(() => screen.getByRole('button', { name: /i don't know/i }))
    await fireEvent.click(screen.getByRole('button', { name: /i don't know/i }))
    await vi.waitFor(() => screen.getByRole('button', { name: /next/i }))
    await fireEvent.click(screen.getByRole('button', { name: /next/i }))
    await vi.waitFor(() => {
      expect(screen.getByRole('button', { name: /practice again/i })).toBeInTheDocument()
      expect(screen.getByRole('button', { name: /back to deck/i })).toBeInTheDocument()
    })
  })

  it('Back to Deck navigates to deck detail', async () => {
    const singleCard = [cards[0]]
    vi.stubGlobal('fetch', makeFetch(singleCard))
    render(PracticePage)
    await vi.waitFor(() => screen.getByRole('button', { name: /i don't know/i }))
    await fireEvent.click(screen.getByRole('button', { name: /i don't know/i }))
    await vi.waitFor(() => screen.getByRole('button', { name: /next/i }))
    await fireEvent.click(screen.getByRole('button', { name: /next/i }))
    await vi.waitFor(() => screen.getByRole('button', { name: /back to deck/i }))
    await fireEvent.click(screen.getByRole('button', { name: /back to deck/i }))
    expect(goto).toHaveBeenCalledWith('/decks/42')
  })

  it('shows no-media message when API returns empty array', async () => {
    vi.stubGlobal('fetch', makeFetch([]))
    render(PracticePage)
    await vi.waitFor(() => {
      expect(screen.getByText(/no species with media/i)).toBeInTheDocument()
    })
  })

  it('shows error message on network failure', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('Network error')))
    render(PracticePage)
    await vi.waitFor(() => {
      expect(screen.getByText(/failed to load/i)).toBeInTheDocument()
    })
  })
})
