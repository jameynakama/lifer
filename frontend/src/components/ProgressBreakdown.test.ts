import { describe, it, expect, vi, afterEach } from 'vitest'
import { screen, fireEvent } from '@testing-library/svelte'
import { renderWithClient } from '../test-utils'
import ProgressBreakdown from './ProgressBreakdown.svelte'

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

const progress = { egg: 0, nestling: 10, fledgling: 51, juvenile: 2, immature: 0, adult: 0 }

describe('ProgressBreakdown', () => {
  it('renders all six tiers including empty ones', () => {
    renderWithClient(ProgressBreakdown, { progress })
    for (const name of ['Egg', 'Nestling', 'Fledgling', 'Juvenile', 'Immature', 'Adult']) {
      expect(screen.getByText(new RegExp(name))).toBeTruthy()
    }
  })

  it('shows the count per tier', () => {
    renderWithClient(ProgressBreakdown, { progress })
    // The button contains both label and count spans
    const fledglingBtn = screen.getByText(/Fledgling/).closest('button')!
    expect(fledglingBtn).toHaveTextContent('51')
  })

  it('shows zero counts for empty tiers with empty class', () => {
    renderWithClient(ProgressBreakdown, { progress })
    // Egg has count 0 — button should have the .empty class
    const eggBtn = screen.getByText(/Egg/).closest('button')
    expect(eggBtn).toHaveClass('empty')
  })

  it('shows error message when the tier fetch fails', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(new Response('Internal Server Error', { status: 500 })),
    )
    renderWithClient(ProgressBreakdown, { progress })
    const fledglingBtn = screen.getByText(/Fledgling/).closest('button')!
    await fireEvent.click(fledglingBtn)
    await vi.waitFor(() => {
      expect(screen.getByText(/couldn't load birds/i)).toBeInTheDocument()
    })
  })

  it('clicking a tier with count > 0 calls apiGet for that stage', async () => {
    const birds = [
      {
        ebird_code: 'sonspa',
        common_name: 'Song Sparrow',
        scientific_name: 'Melospiza melodia',
        lane: 'audio',
        stability: 3,
      },
    ]
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(new Response(JSON.stringify(birds), { status: 200 })),
    )
    renderWithClient(ProgressBreakdown, { progress })
    const fledglingBtn = screen.getByText(/Fledgling/).closest('button')!
    await fireEvent.click(fledglingBtn)
    await vi.waitFor(() => {
      expect(screen.getByText(/Song Sparrow/i)).toBeInTheDocument()
    })
    const fetchMock = vi.mocked(fetch as typeof fetch)
    const urls = fetchMock.mock.calls.map((c) => c[0] as string)
    expect(urls.some((u) => u.includes('/api/v1/stats/tier/fledgling'))).toBe(true)
    // Each bird links to its explore detail page.
    const link = screen.getByRole('link', { name: /Song Sparrow/i })
    expect(link).toHaveAttribute('href', '/explore/sonspa')
  })
})
