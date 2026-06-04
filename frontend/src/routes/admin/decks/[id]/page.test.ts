import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen } from '@testing-library/svelte'
import DeckDetailPage from './+page.svelte'

const mockDetail = {
  deck: { id: 5, name: 'My Warblers', owner_name: 'Alice', owner_email: 'alice@example.com' },
  species: [
    { ebird_code: 'yewwar', common_name: 'Yellow Warbler', scientific_name: 'Setophaga petechia' },
    { ebird_code: 'amredi', common_name: 'American Redstart', scientific_name: 'Setophaga ruticilla' },
  ],
}

const mockPresetDetail = {
  deck: { id: 2, name: 'Confusing Woodpeckers', owner_name: '', owner_email: '' },
  species: [
    { ebird_code: 'pilwoo', common_name: 'Pileated Woodpecker', scientific_name: 'Dryocopus pileatus' },
  ],
}

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe('Admin deck detail page', () => {
  it('shows deck name and species', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve(mockDetail) }))
    render(DeckDetailPage)
    await vi.waitFor(() => {
      expect(screen.getByText(/my warblers/i)).toBeInTheDocument()
      expect(screen.getByText(/yellow warbler/i)).toBeInTheDocument()
      expect(screen.getByText(/american redstart/i)).toBeInTheDocument()
    })
  })

  it('shows owner info for user decks', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve(mockDetail) }))
    render(DeckDetailPage)
    await vi.waitFor(() => {
      expect(screen.getByText(/alice@example\.com/i)).toBeInTheDocument()
    })
  })

  it('shows Preset badge and no owner email for preset decks', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve(mockPresetDetail) }))
    render(DeckDetailPage)
    await vi.waitFor(() => {
      expect(screen.getByText(/preset/i)).toBeInTheDocument()
      expect(screen.queryByText(/@/)).not.toBeInTheDocument()
    })
  })

  it('shows a back link to /admin/decks', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve(mockDetail) }))
    render(DeckDetailPage)
    await vi.waitFor(() => screen.getByRole('link', { name: /back/i }))
    expect(screen.getByRole('link', { name: /back/i })).toHaveAttribute('href', '/admin/decks')
  })

  it('shows loading state initially', () => {
    vi.stubGlobal('fetch', vi.fn().mockReturnValue(new Promise(() => {})))
    render(DeckDetailPage)
    expect(screen.getByText(/loading/i)).toBeInTheDocument()
  })
})
