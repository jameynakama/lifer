import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { screen } from '@testing-library/svelte'
import { goto } from '$app/navigation'
import { page } from '$app/state'
import { renderWithClient } from '../../test-utils'
import StatsPage from './+page.svelte'
import type { StatsResponse } from '../../types'

const combined: StatsResponse = {
  totals: {
    species: 112,
    cards: 224,
    known: 38,
    reviews: 412,
    lapses: 91,
    attempts: 73,
    correct: 57,
    reviews_last_7d: 23,
  },
  progress: { not_seen: 141, learning: 28, known: 38, relearning: 17 },
  lanes: {
    audio: { cards: 112, known: 14 },
    image: { cards: 112, known: 24 },
    gaps: [
      {
        ebird_code: 'varthr',
        common_name: 'Varied Thrush',
        scientific_name: 'Ixoreus naevius',
        known_lane: 'image',
        weak_lane: 'audio',
      },
    ],
  },
  confusions: [
    {
      actual: {
        ebird_code: 'foxspa',
        common_name: 'Fox Sparrow',
        scientific_name: 'Passerella iliaca',
      },
      guessed: {
        ebird_code: 'sonspa',
        common_name: 'Song Sparrow',
        scientific_name: 'Melospiza melodia',
      },
      count: 4,
    },
  ],
  families: [{ family: 'Waterfowl', attempts: 32, correct: 10 }],
  fading: [
    {
      ebird_code: 'ruckin',
      common_name: 'Ruby-crowned Kinglet',
      scientific_name: 'Corthylio calendula',
      lane: 'audio',
      retrievability: 0.71,
      due_in_days: 3,
    },
  ],
  remember: { now: 38, in_a_week: 31, in_a_month: 24 },
  hard_media: [
    {
      ebird_code: 'sonspa',
      common_name: 'Song Sparrow',
      scientific_name: 'Melospiza melodia',
      lane: 'audio',
      media_id: 'XC58291',
      media_url: 'https://example.com/x.mp3',
      attempts: 4,
      correct: 0,
    },
  ],
}

beforeEach(() => {
  page.url = new URL('http://localhost/stats') as typeof page.url
  vi.mocked(goto).mockClear()
})

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe('Stats page', () => {
  it('renders panels from the combined response', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve(combined) }),
    )
    renderWithClient(StatsPage)
    await vi.waitFor(() => {
      expect(screen.getByText(/fox sparrow/i)).toBeInTheDocument()
      expect(screen.getByText(/waterfowl/i)).toBeInTheDocument()
      expect(screen.getByText(/ruby-crowned kinglet/i)).toBeInTheDocument()
    })
  })

  it('fetches the combined endpoint by default and shows ear vs eye', async () => {
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve(combined) })
    vi.stubGlobal('fetch', fetchMock)
    renderWithClient(StatsPage)
    await vi.waitFor(() => {
      const urls = fetchMock.mock.calls.map((c: unknown[]) => c[0] as string)
      expect(urls.some((u) => u.endsWith('/api/v1/stats'))).toBe(true)
      expect(screen.getByText(/ear vs eye/i)).toBeInTheDocument()
    })
  })

  it('lane tab links to ?lane=audio', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve(combined) }),
    )
    renderWithClient(StatsPage)
    await vi.waitFor(() => screen.getByRole('link', { name: /audio/i }))
    expect(screen.getByRole('link', { name: /audio/i })).toHaveAttribute(
      'href',
      '/stats?lane=audio',
    )
  })

  it('omits ear vs eye on a lane tab and fetches with the lane param', async () => {
    page.url = new URL('http://localhost/stats?lane=audio') as typeof page.url
    const { lanes: _lanes, ...laneResp } = combined
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve(laneResp) })
    vi.stubGlobal('fetch', fetchMock)
    renderWithClient(StatsPage)
    await vi.waitFor(() => screen.getByText(/fox sparrow/i))
    expect(screen.queryByText(/ear vs eye/i)).not.toBeInTheDocument()
    const urls = fetchMock.mock.calls.map((c: unknown[]) => c[0] as string)
    expect(urls.some((u) => u.includes('/api/v1/stats?lane=audio'))).toBe(true)
  })

  it('shows empty states for log-driven panels', async () => {
    const empty: StatsResponse = {
      ...combined,
      confusions: [],
      families: [],
      fading: [],
      hard_media: [],
    }
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve(empty) }),
    )
    renderWithClient(StatsPage)
    await vi.waitFor(() => {
      expect(screen.getByText(/no confusions/i)).toBeInTheDocument()
    })
  })
})
