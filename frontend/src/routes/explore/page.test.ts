import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/svelte'
import ExplorePage from './+page.svelte'
import { queryResult } from '../../test-utils'

vi.mock('@tanstack/svelte-query', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/svelte-query')>()
  return {
    ...actual,
    createQuery: vi.fn(),
  }
})

const mockSpecies = [
  {
    ebird_code: 'amro',
    common_name: 'American Robin',
    scientific_name: 'Turdus migratorius',
    image_url: null,
  },
  {
    ebird_code: 'bcch',
    common_name: 'Black-capped Chickadee',
    scientific_name: 'Poecile atricapillus',
    image_url: null,
  },
]

beforeEach(async () => {
  const { createQuery } = await import('@tanstack/svelte-query')
  vi.mocked(createQuery).mockReturnValue(
    queryResult({
      data: mockSpecies,
      isPending: false,
      isError: false,
    }),
  )
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
    vi.mocked(createQuery).mockReturnValue(
      queryResult({
        data: manySpecies,
        isPending: false,
        isError: false,
      }),
    )
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

  it('resets to first page when search query changes', async () => {
    const { createQuery } = await import('@tanstack/svelte-query')
    const manySpecies = Array.from({ length: 25 }, (_, i) => ({
      ebird_code: `sp${i}`,
      common_name: i === 24 ? 'Unique Robin' : `Species ${i}`,
      scientific_name: `Genus species${i}`,
      image_url: null,
    }))
    vi.mocked(createQuery).mockReturnValue(
      queryResult({
        data: manySpecies,
        isPending: false,
        isError: false,
      }),
    )
    render(ExplorePage)

    // Navigate to page 2
    const nextBtn = screen.getByRole('button', { name: /next page/i })
    await fireEvent.click(nextBtn)
    await vi.waitFor(() => {
      expect(screen.getByText('Unique Robin')).toBeTruthy()
    })

    // Type a search -- offset should reset, showing from filtered start
    const input = screen.getByPlaceholderText(/search species/i)
    await fireEvent.input(input, { target: { value: 'species' } })
    await vi.waitFor(() => {
      expect(screen.getByText('Species 0')).toBeTruthy()
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
    render(ExplorePage)
    expect(screen.getByText(/loading/i)).toBeTruthy()
  })

  it('shows error state', async () => {
    const { createQuery } = await import('@tanstack/svelte-query')
    vi.mocked(createQuery).mockReturnValue(
      queryResult({
        data: undefined,
        isPending: false,
        isError: true,
      }),
    )
    render(ExplorePage)
    expect(screen.getByText(/couldn't load species/i)).toBeTruthy()
  })

  it('shows a checkbox for each species row', async () => {
    render(ExplorePage)
    await vi.waitFor(() => {
      const checkboxes = screen.getAllByRole('checkbox')
      expect(checkboxes).toHaveLength(2)
    })
  })

  it('shows bulk add bar when a species is checked', async () => {
    render(ExplorePage)
    await vi.waitFor(() => screen.getAllByRole('checkbox'))
    const [firstCheckbox] = screen.getAllByRole('checkbox')
    await fireEvent.click(firstCheckbox)
    await vi.waitFor(() => {
      expect(screen.getByRole('toolbar')).toBeInTheDocument()
      expect(screen.getByText(/1 selected/i)).toBeInTheDocument()
    })
  })

  it('pinned species remain visible when search changes', async () => {
    render(ExplorePage)
    await vi.waitFor(() => screen.getAllByRole('checkbox'))
    // Check American Robin
    const checkboxes = screen.getAllByRole('checkbox')
    await fireEvent.click(checkboxes[0]) // American Robin
    // Now search for chickadee -- Robin no longer in filtered but stays pinned
    const input = screen.getByPlaceholderText(/search species/i)
    await fireEvent.input(input, { target: { value: 'chickadee' } })
    await vi.waitFor(() => {
      expect(screen.getAllByText(/american robin/i).length).toBeGreaterThan(0)
      expect(screen.getAllByText(/black-capped chickadee/i).length).toBeGreaterThan(0)
    })
  })

  it('shows select-all button when search is active', async () => {
    render(ExplorePage)
    const input = screen.getByPlaceholderText(/search species/i)
    await fireEvent.input(input, { target: { value: 'ro' } })
    await vi.waitFor(() => {
      expect(screen.getByRole('button', { name: /select all/i })).toBeInTheDocument()
    })
  })

  it('select-all selects all displayed non-pinned species', async () => {
    render(ExplorePage)
    const input = screen.getByPlaceholderText(/search species/i)
    await fireEvent.input(input, { target: { value: 'ro' } })
    await vi.waitFor(() => screen.getByRole('button', { name: /select all/i }))
    await fireEvent.click(screen.getByRole('button', { name: /select all/i }))
    await vi.waitFor(() => {
      expect(screen.getByRole('toolbar')).toBeInTheDocument()
    })
  })

  it('clears selection when clear button clicked', async () => {
    render(ExplorePage)
    await vi.waitFor(() => screen.getAllByRole('checkbox'))
    const [firstCheckbox] = screen.getAllByRole('checkbox')
    await fireEvent.click(firstCheckbox)
    await vi.waitFor(() => screen.getByRole('toolbar'))
    await fireEvent.click(screen.getByRole('button', { name: /clear/i }))
    await vi.waitFor(() => {
      expect(screen.queryByRole('toolbar')).toBeNull()
    })
  })
})
