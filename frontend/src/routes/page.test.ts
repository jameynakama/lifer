import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/svelte'
import { goto } from '$app/navigation'
import Dashboard from './+page.svelte'

const decks = [
  { id: 1, name: 'Pacific Northwest', audio_due: 8, image_due: 5 },
  { id: 2, name: 'My Warblers', audio_due: 3, image_due: 0 },
]

beforeEach(() => {
  vi.mocked(goto).mockClear()
})

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe('Dashboard page', () => {
  it('renders deck names from API', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ decks, next_due_at: null }),
    }))
    render(Dashboard)
    await vi.waitFor(() => {
      expect(screen.getAllByText(/pacific northwest/i).length).toBeGreaterThan(0)
    })
    expect(screen.getAllByText(/my warblers/i).length).toBeGreaterThan(0)
  })

  it('shows empty state when no decks', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ decks: [], next_due_at: null }),
    }))
    render(Dashboard)
    await vi.waitFor(() => {
      expect(screen.getByText(/no decks yet/i)).toBeInTheDocument()
    })
  })

  it('navigates to quiz when Audio button clicked', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ decks, next_due_at: null }),
    }))
    render(Dashboard)
    await vi.waitFor(() => screen.getAllByRole('button', { name: /audio/i }))
    await fireEvent.click(screen.getAllByRole('button', { name: /audio/i })[0])
    expect(goto).toHaveBeenCalledWith('/decks/1/quiz?lane=audio')
  })

  it('navigates to quiz when Image button clicked', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ decks, next_due_at: null }),
    }))
    render(Dashboard)
    await vi.waitFor(() => screen.getAllByRole('button', { name: /image/i }))
    await fireEvent.click(screen.getAllByRole('button', { name: /image/i })[0])
    expect(goto).toHaveBeenCalledWith('/decks/1/quiz?lane=image')
  })
})
