import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen } from '@testing-library/svelte'
import { goto } from '$app/navigation'
import { page } from '$app/state'
import SpeciesDetailPage from './+page.svelte'
import { queryResult } from '../../../test-utils'

vi.mock('@tanstack/svelte-query', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/svelte-query')>()
  return {
    ...actual,
    createQuery: vi.fn(),
  }
})

const mockDetail = {
  ebird_code: 'amro',
  common_name: 'American Robin',
  scientific_name: 'Turdus migratorius',
  recordings: [
    {
      xeno_canto_id: 'xc123',
      file_path: 'https://r2.example.com/xc123.mp3',
      quality: 'A',
      type: 'song',
    },
  ],
  images: [
    { macaulay_id: 'ml456', file_path: 'https://r2.example.com/ml456.jpg', credit: 'J. Doe' },
  ],
}

beforeEach(async () => {
  page.params = { ebird_code: 'amro' }
  vi.mocked(goto).mockClear()
  const { createQuery } = await import('@tanstack/svelte-query')
  vi.mocked(createQuery).mockReturnValue(
    queryResult({
      data: mockDetail,
      isPending: false,
      isError: false,
    }),
  )
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe('Species detail page', () => {
  it('renders species name', async () => {
    render(SpeciesDetailPage)
    await vi.waitFor(() => {
      expect(screen.getByText(/american robin/i)).toBeTruthy()
      expect(screen.getByText(/turdus migratorius/i)).toBeTruthy()
    })
  })

  it('renders recordings section label', async () => {
    render(SpeciesDetailPage)
    await vi.waitFor(() => {
      expect(screen.getByText(/recordings/i)).toBeTruthy()
    })
  })

  it('renders photos section label', async () => {
    render(SpeciesDetailPage)
    await vi.waitFor(() => {
      expect(screen.getByText(/photos/i)).toBeTruthy()
    })
  })

  it('shows loading state', async () => {
    const { createQuery } = await import('@tanstack/svelte-query')
    vi.mocked(createQuery).mockReturnValue(
      queryResult({
        data: undefined,
        isPending: true,
        isError: false,
      }),
    )
    render(SpeciesDetailPage)
    expect(screen.getByText(/loading/i)).toBeTruthy()
  })

  it('redirects to /explore on error', async () => {
    const { createQuery } = await import('@tanstack/svelte-query')
    vi.mocked(createQuery).mockReturnValue(
      queryResult({
        data: undefined,
        isPending: false,
        isError: true,
      }),
    )
    render(SpeciesDetailPage)
    await vi.waitFor(() => {
      expect(goto).toHaveBeenCalledWith('/explore')
    })
  })
})
