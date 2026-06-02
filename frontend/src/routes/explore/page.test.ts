import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/svelte'
import ExplorePage from './+page.svelte'

vi.mock('@tanstack/svelte-query', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/svelte-query')>()
  return {
    ...actual,
    createQuery: vi.fn(),
  }
})

const mockSpecies = [
  { ebird_code: 'amro', common_name: 'American Robin', scientific_name: 'Turdus migratorius', image_url: null },
  { ebird_code: 'bcch', common_name: 'Black-capped Chickadee', scientific_name: 'Poecile atricapillus', image_url: null },
]

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

  it('shows pagination when species count exceeds limit', async () => {
    const { createQuery } = await import('@tanstack/svelte-query')
    const manySpecies = Array.from({ length: 25 }, (_, i) => ({
      ebird_code: `sp${i}`,
      common_name: `Species ${i}`,
      scientific_name: `Genus species${i}`,
      image_url: null,
    }))
    vi.mocked(createQuery).mockReturnValue({
      data: manySpecies,
      isPending: false,
      isError: false,
    } as any)
    render(ExplorePage)
    await vi.waitFor(() => {
      expect(screen.getByRole('navigation', { name: /pagination/i })).toBeTruthy()
    })
  })

  it('filters species by common name when searching', async () => {
    render(ExplorePage)
    const input = screen.getByPlaceholderText(/search species/i)
    await fireEvent.input(input, { target: { value: 'robin' } })
    await vi.waitFor(() => {
      expect(screen.getByText(/american robin/i)).toBeTruthy()
      expect(screen.queryByText(/black-capped chickadee/i)).toBeNull()
    })
  })

  it('filters species by scientific name when searching', async () => {
    render(ExplorePage)
    const input = screen.getByPlaceholderText(/search species/i)
    await fireEvent.input(input, { target: { value: 'poecile' } })
    await vi.waitFor(() => {
      expect(screen.getByText(/black-capped chickadee/i)).toBeTruthy()
      expect(screen.queryByText(/american robin/i)).toBeNull()
    })
  })

  it('shows all species when search query is shorter than 2 chars', async () => {
    render(ExplorePage)
    const input = screen.getByPlaceholderText(/search species/i)
    await fireEvent.input(input, { target: { value: 'r' } })
    await vi.waitFor(() => {
      expect(screen.getByText(/american robin/i)).toBeTruthy()
      expect(screen.getByText(/black-capped chickadee/i)).toBeTruthy()
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
