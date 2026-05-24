import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/svelte'
import { goto } from '$app/navigation'
import { page } from '$app/state'
import GroupDetailPage from './+page.svelte'

const species = [
  { id: 7, common_name: 'Song Sparrow', scientific_name: 'Melospiza melodia', ebird_code: 'sonspa' },
]

beforeEach(() => {
  page.params = { id: '42' }
  vi.mocked(goto).mockClear()
})

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe('Group detail page', () => {
  it('renders species list for the group', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true, json: () => Promise.resolve(species),
    }))
    render(GroupDetailPage)
    await vi.waitFor(() => {
      expect(screen.getAllByText(/song sparrow/i).length).toBeGreaterThan(0)
    })
  })

  it('navigates to audio quiz on Practice Audio click', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true, json: () => Promise.resolve(species),
    }))
    render(GroupDetailPage)
    await vi.waitFor(() => screen.getByRole('button', { name: /practice audio/i }))
    await fireEvent.click(screen.getByRole('button', { name: /practice audio/i }))
    expect(goto).toHaveBeenCalledWith('/groups/42/quiz?lane=audio')
  })

  it('searches species and shows results', async () => {
    vi.stubGlobal('fetch', vi.fn().mockImplementation((url: string) => {
      if (url.includes('/api/v1/species')) {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve([
            { id: 8, common_name: 'Fox Sparrow', scientific_name: 'Passerella iliaca', ebird_code: 'foxspa' },
          ]),
        })
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve(species) })
    }))
    render(GroupDetailPage)
    await vi.waitFor(() => screen.getByPlaceholderText(/search species/i))
    await fireEvent.input(screen.getByPlaceholderText(/search species/i), {
      target: { value: 'fox' },
    })
    await vi.waitFor(() => {
      expect(screen.getAllByText(/fox sparrow/i).length).toBeGreaterThan(0)
    })
  })

  it('removes species from group on Remove click', async () => {
    let deleteCalled = false
    vi.stubGlobal('fetch', vi.fn().mockImplementation((url: string, opts?: RequestInit) => {
      if (opts?.method === 'DELETE') {
        deleteCalled = true
        return Promise.resolve({ ok: true })
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve(species) })
    }))
    render(GroupDetailPage)
    await vi.waitFor(() => screen.getByRole('button', { name: /remove/i }))
    await fireEvent.click(screen.getByRole('button', { name: /remove/i }))
    await vi.waitFor(() => { expect(deleteCalled).toBe(true) })
  })
})
