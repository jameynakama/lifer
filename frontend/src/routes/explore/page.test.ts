import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen } from '@testing-library/svelte'
import ExplorePage from './+page.svelte'

vi.mock('@tanstack/svelte-query', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/svelte-query')>()
  return {
    ...actual,
    createQuery: vi.fn(),
  }
})

const mockSpecies = {
  count: 2,
  next: null,
  previous: null,
  results: [
    { ebird_code: 'amro', common_name: 'American Robin', scientific_name: 'Turdus migratorius' },
    { ebird_code: 'bcch', common_name: 'Black-capped Chickadee', scientific_name: 'Poecile atricapillus' },
  ],
}

beforeEach(async () => {
  const { createQuery } = await import('@tanstack/svelte-query')
  vi.mocked(createQuery).mockReturnValue({
    data: mockSpecies,
    isPending: false,
    isError: false,
  } as any)
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe('Explore page', () => {
  it('renders species rows from API response', async () => {
    render(ExplorePage)
    await vi.waitFor(() => {
      expect(screen.getByText(/american robin/i)).toBeTruthy()
      expect(screen.getByText(/black-capped chickadee/i)).toBeTruthy()
    })
  })

  it('shows pagination when count exceeds limit', async () => {
    const { createQuery } = await import('@tanstack/svelte-query')
    vi.mocked(createQuery).mockReturnValue({
      data: { ...mockSpecies, count: 100 },
      isPending: false,
      isError: false,
    } as any)
    render(ExplorePage)
    await vi.waitFor(() => {
      expect(screen.getByRole('navigation', { name: /pagination/i })).toBeTruthy()
    })
  })

  it('shows loading state', async () => {
    const { createQuery } = await import('@tanstack/svelte-query')
    vi.mocked(createQuery).mockReturnValue({
      data: undefined,
      isPending: true,
      isError: false,
    } as any)
    render(ExplorePage)
    expect(screen.getByText(/loading/i)).toBeTruthy()
  })

  it('shows error state', async () => {
    const { createQuery } = await import('@tanstack/svelte-query')
    vi.mocked(createQuery).mockReturnValue({
      data: undefined,
      isPending: false,
      isError: true,
    } as any)
    render(ExplorePage)
    expect(screen.getByText(/couldn't load species/i)).toBeTruthy()
  })
})
