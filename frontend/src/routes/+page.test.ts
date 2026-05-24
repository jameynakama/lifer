import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/svelte'
import { goto } from '$app/navigation'
import Dashboard from './+page.svelte'

const groups = [
  { id: 1, name: 'Pacific Northwest', is_preset: false, audio_due: 8, image_due: 5 },
  { id: 2, name: 'My Warblers', is_preset: false, audio_due: 3, image_due: 0 },
]

beforeEach(() => {
  vi.mocked(goto).mockClear()
})

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe('Dashboard page', () => {
  it('renders group names from API', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(groups),
    }))
    render(Dashboard)
    await vi.waitFor(() => {
      expect(screen.getAllByText(/pacific northwest/i).length).toBeGreaterThan(0)
    })
    expect(screen.getAllByText(/my warblers/i).length).toBeGreaterThan(0)
  })

  it('shows empty state when no groups', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve([]),
    }))
    render(Dashboard)
    await vi.waitFor(() => {
      expect(screen.getByText(/no groups yet/i)).toBeInTheDocument()
    })
  })

  it('navigates to quiz when Audio button clicked', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(groups),
    }))
    render(Dashboard)
    await vi.waitFor(() => screen.getAllByRole('button', { name: /audio/i }))
    await fireEvent.click(screen.getAllByRole('button', { name: /audio/i })[0])
    expect(goto).toHaveBeenCalledWith('/groups/1/quiz?lane=audio')
  })

  it('navigates to quiz when Image button clicked', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(groups),
    }))
    render(Dashboard)
    await vi.waitFor(() => screen.getAllByRole('button', { name: /image/i }))
    await fireEvent.click(screen.getAllByRole('button', { name: /image/i })[0])
    expect(goto).toHaveBeenCalledWith('/groups/1/quiz?lane=image')
  })
})
