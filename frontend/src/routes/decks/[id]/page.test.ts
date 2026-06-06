import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/svelte'
import { goto } from '$app/navigation'
import { page } from '$app/state'
import DeckDetailPage from './+page.svelte'
import { queryResult } from '../../../test-utils'

vi.mock('@tanstack/svelte-query', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/svelte-query')>()
  return {
    ...actual,
    createQuery: vi.fn(),
    useQueryClient: () => ({ invalidateQueries: vi.fn() }),
  }
})

const speciesWithPrefs = [
  {
    ebird_code: 'sonspa',
    common_name: 'Song Sparrow',
    scientific_name: 'Melospiza melodia',
    audio_enabled: true,
    image_enabled: true,
  },
]

function makeFetch(overrides: Record<string, unknown> = {}) {
  return vi.fn().mockImplementation((url: string, opts?: RequestInit) => {
    if (opts?.method === 'DELETE') {
      return Promise.resolve({ ok: true, status: 204 })
    }
    if (opts?.method === 'POST') {
      // Match the real backend: 201 Created with an empty body
      return Promise.resolve(new Response(null, { status: 201 }))
    }
    if (
      opts?.method === 'PUT' &&
      (overrides.putHandler as (url: string, opts: RequestInit) => unknown)
    ) {
      return (overrides.putHandler as (url: string, opts: RequestInit) => unknown)(url, opts!)
    }
    if (url.match(/\/api\/v1\/decks\/\d+\/species/)) {
      return Promise.resolve({ ok: true, json: () => Promise.resolve(speciesWithPrefs) })
    }
    // deck detail (audio_due / image_due)
    return Promise.resolve({
      ok: true,
      json: () => Promise.resolve({ audio_due: 0, image_due: 0 }),
    })
  })
}

beforeEach(async () => {
  page.params = { id: '42' }
  vi.mocked(goto).mockClear()
  const { createQuery } = await import('@tanstack/svelte-query')
  vi.mocked(createQuery).mockReturnValue(
    queryResult({
      data: [],
      isPending: false,
      isError: false,
    }),
  )
})

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe('Deck detail page', () => {
  it('renders species list for the deck', async () => {
    vi.stubGlobal('fetch', makeFetch())
    render(DeckDetailPage)
    await vi.waitFor(() => {
      expect(screen.getAllByText(/song sparrow/i).length).toBeGreaterThan(0)
    })
  })

  it('navigates to audio quiz on Study Audio click', async () => {
    vi.stubGlobal('fetch', makeFetch())
    render(DeckDetailPage)
    await vi.waitFor(() => screen.getByRole('button', { name: /study audio/i }))
    await fireEvent.click(screen.getByRole('button', { name: /study audio/i }))
    expect(goto).toHaveBeenCalledWith('/decks/42/quiz?lane=audio')
  })

  it('navigates to image quiz on Study Image click', async () => {
    vi.stubGlobal('fetch', makeFetch())
    render(DeckDetailPage)
    await vi.waitFor(() => screen.getByRole('button', { name: /study image/i }))
    await fireEvent.click(screen.getByRole('button', { name: /study image/i }))
    expect(goto).toHaveBeenCalledWith('/decks/42/quiz?lane=image')
  })

  it('navigates to practice page on Practice Audio click', async () => {
    vi.stubGlobal('fetch', makeFetch())
    render(DeckDetailPage)
    await vi.waitFor(() => screen.getByRole('button', { name: /^practice audio$/i }))
    await fireEvent.click(screen.getByRole('button', { name: /^practice audio$/i }))
    expect(goto).toHaveBeenCalledWith('/decks/42/practice?lane=audio')
  })

  it('navigates to practice page on Practice Image click', async () => {
    vi.stubGlobal('fetch', makeFetch())
    render(DeckDetailPage)
    await vi.waitFor(() => screen.getByRole('button', { name: /^practice image$/i }))
    await fireEvent.click(screen.getByRole('button', { name: /^practice image$/i }))
    expect(goto).toHaveBeenCalledWith('/decks/42/practice?lane=image')
  })

  it('searches species and shows results', async () => {
    const foxSparrow = {
      ebird_code: 'foxspa',
      common_name: 'Fox Sparrow',
      scientific_name: 'Passerella iliaca',
      image_url: null,
    }
    const { createQuery } = await import('@tanstack/svelte-query')
    vi.mocked(createQuery).mockReturnValue(
      queryResult({
        data: [foxSparrow],
        isPending: false,
        isError: false,
      }),
    )
    vi.stubGlobal('fetch', makeFetch())
    render(DeckDetailPage)
    await vi.waitFor(() => screen.getByPlaceholderText(/search species/i))
    await fireEvent.input(screen.getByPlaceholderText(/search species/i), {
      target: { value: 'fox' },
    })
    await vi.waitFor(() => {
      expect(screen.getAllByText(/fox sparrow/i).length).toBeGreaterThan(0)
    })
  })

  it('removes species from deck on Remove click', async () => {
    let deleteCalled = false
    vi.stubGlobal(
      'fetch',
      vi.fn().mockImplementation((url: string, opts?: RequestInit) => {
        if (opts?.method === 'DELETE') {
          deleteCalled = true
          return Promise.resolve({ ok: true, status: 204 })
        }
        if (url.match(/\/api\/v1\/decks\/\d+\/species/)) {
          return Promise.resolve({ ok: true, json: () => Promise.resolve(speciesWithPrefs) })
        }
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({ audio_due: 0, image_due: 0 }),
        })
      }),
    )
    render(DeckDetailPage)
    await vi.waitFor(() => screen.getByRole('button', { name: /remove/i }))
    await fireEvent.click(screen.getByRole('button', { name: /remove/i }))
    await vi.waitFor(() => {
      expect(deleteCalled).toBe(true)
    })
    await vi.waitFor(() => {
      expect(screen.queryByText(/song sparrow/i)).toBeNull()
    })
  })

  it('keeps search results and input value after adding a species', async () => {
    const foxSparrow = {
      ebird_code: 'foxspa',
      common_name: 'Fox Sparrow',
      scientific_name: 'Passerella iliaca',
      image_url: null,
    }
    const { createQuery } = await import('@tanstack/svelte-query')
    vi.mocked(createQuery).mockReturnValue(
      queryResult({
        data: [foxSparrow],
        isPending: false,
        isError: false,
      }),
    )
    vi.stubGlobal('fetch', makeFetch())
    render(DeckDetailPage)
    await vi.waitFor(() => screen.getByPlaceholderText(/search species/i))
    await fireEvent.input(screen.getByPlaceholderText(/search species/i), {
      target: { value: 'fox' },
    })
    await vi.waitFor(() => screen.getByRole('button', { name: /^add$/i }))
    await fireEvent.click(screen.getByRole('button', { name: /^add$/i }))
    await vi.waitFor(() => {
      expect(screen.getByDisplayValue('fox')).toBeInTheDocument()
      expect(screen.getAllByText(/fox sparrow/i).length).toBeGreaterThan(0)
    })
  })

  it('adds the species to the deck list in place when Add is clicked', async () => {
    const foxSparrow = {
      ebird_code: 'foxspa',
      common_name: 'Fox Sparrow',
      scientific_name: 'Passerella iliaca',
      image_url: null,
    }
    const { createQuery } = await import('@tanstack/svelte-query')
    vi.mocked(createQuery).mockReturnValue(
      queryResult({
        data: [foxSparrow],
        isPending: false,
        isError: false,
      }),
    )
    vi.stubGlobal('fetch', makeFetch())
    render(DeckDetailPage)
    await vi.waitFor(() => screen.getByPlaceholderText(/search species/i))
    await fireEvent.input(screen.getByPlaceholderText(/search species/i), {
      target: { value: 'fox' },
    })
    await vi.waitFor(() => screen.getByRole('button', { name: /^add$/i }))
    await fireEvent.click(screen.getByRole('button', { name: /^add$/i }))
    await vi.waitFor(() => {
      // species row appears in the deck list (Remove button) without a refresh
      expect(screen.getAllByRole('button', { name: /remove/i }).length).toBeGreaterThan(1)
      // and the search row flips from Add to Added
      expect(screen.getByText(/added/i)).toBeInTheDocument()
      expect(screen.queryByRole('button', { name: /^add$/i })).toBeNull()
    })
  })

  it('shows Added indicator for species already in the deck', async () => {
    const { createQuery } = await import('@tanstack/svelte-query')
    vi.mocked(createQuery).mockReturnValue(
      queryResult({
        data: speciesWithPrefs.map((s) => ({ ...s, image_url: null })),
        isPending: false,
        isError: false,
      }),
    )
    vi.stubGlobal('fetch', makeFetch())
    render(DeckDetailPage)
    await vi.waitFor(() => screen.getByPlaceholderText(/search species/i))
    await fireEvent.input(screen.getByPlaceholderText(/search species/i), {
      target: { value: 'song' },
    })
    await vi.waitFor(() => screen.getByText(/added/i))
    expect(screen.queryByRole('button', { name: /^add$/i })).toBeNull()
  })

  it('clicking Audio toggle fires PUT /preferences with audio_enabled flipped', async () => {
    let putUrl = ''
    let putBody: Record<string, boolean> | null = null

    vi.stubGlobal(
      'fetch',
      makeFetch({
        putHandler: (url: string, opts: RequestInit) => {
          putUrl = url
          putBody = JSON.parse(opts.body as string)
          return Promise.resolve({
            ok: true,
            json: () => Promise.resolve({ audio_enabled: false, image_enabled: true }),
          })
        },
      }),
    )
    render(DeckDetailPage)
    await vi.waitFor(() => screen.getByRole('button', { name: /toggle audio/i }))
    await fireEvent.click(screen.getByRole('button', { name: /toggle audio/i }))
    await vi.waitFor(() => {
      expect(putUrl).toContain('/api/v1/species/sonspa/preferences')
      expect(putBody).toEqual({ audio_enabled: false, image_enabled: true })
    })
  })

  it('clicking Image toggle fires PUT /preferences with image_enabled flipped', async () => {
    let putBody: Record<string, boolean> | null = null

    vi.stubGlobal(
      'fetch',
      makeFetch({
        putHandler: (_url: string, opts: RequestInit) => {
          putBody = JSON.parse(opts.body as string)
          return Promise.resolve({
            ok: true,
            json: () => Promise.resolve({ audio_enabled: true, image_enabled: false }),
          })
        },
      }),
    )
    render(DeckDetailPage)
    await vi.waitFor(() => screen.getByRole('button', { name: /toggle image/i }))
    await fireEvent.click(screen.getByRole('button', { name: /toggle image/i }))
    await vi.waitFor(() => {
      expect(putBody).toEqual({ audio_enabled: true, image_enabled: false })
    })
  })

  it('reverts toggle state on PUT failure', async () => {
    vi.stubGlobal(
      'fetch',
      makeFetch({
        putHandler: () => Promise.resolve({ ok: false, json: () => Promise.resolve({}) }),
      }),
    )
    render(DeckDetailPage)
    await vi.waitFor(() => screen.getByRole('button', { name: /toggle audio/i }))

    // audio starts enabled; click to disable (optimistic), then server fails → reverts
    const toggleBtn = screen.getByRole('button', { name: /toggle audio/i })
    expect(toggleBtn.classList.contains('active')).toBe(true)
    await fireEvent.click(toggleBtn)
    await vi.waitFor(() => {
      // reverted: still active
      expect(
        screen.getByRole('button', { name: /toggle audio/i }).classList.contains('active'),
      ).toBe(true)
    })
  })

  it('disables toggle buttons while PUT is in flight', async () => {
    let resolvePut!: () => void
    vi.stubGlobal(
      'fetch',
      makeFetch({
        putHandler: () =>
          new Promise<{ ok: boolean; json: () => Promise<unknown> }>((resolve) => {
            resolvePut = () =>
              resolve({
                ok: true,
                json: () => Promise.resolve({ audio_enabled: false, image_enabled: true }),
              })
          }),
      }),
    )
    render(DeckDetailPage)
    await vi.waitFor(() => screen.getByRole('button', { name: /toggle audio/i }))

    await fireEvent.click(screen.getByRole('button', { name: /toggle audio/i }))
    // while PUT is pending both toggles are disabled
    await vi.waitFor(() => {
      expect(screen.getByRole('button', { name: /toggle audio/i })).toBeDisabled()
      expect(screen.getByRole('button', { name: /toggle image/i })).toBeDisabled()
    })

    resolvePut()
    // after PUT resolves buttons are re-enabled
    await vi.waitFor(() => {
      expect(screen.getByRole('button', { name: /toggle audio/i })).not.toBeDisabled()
    })
  })

  it('shows description returned from API', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockImplementation((url: string, opts?: RequestInit) => {
        if (opts?.method === 'DELETE') return Promise.resolve({ ok: true, status: 204 })
        if (url.match(/\/api\/v1\/decks\/\d+\/species/)) {
          return Promise.resolve({ ok: true, json: () => Promise.resolve([]) })
        }
        return Promise.resolve({
          ok: true,
          json: () =>
            Promise.resolve({
              name: 'My Deck',
              description: 'Birds that look alike',
              audio_due: 0,
              image_due: 0,
            }),
        })
      }),
    )
    render(DeckDetailPage)
    await vi.waitFor(() => {
      expect(screen.getByText(/birds that look alike/i)).toBeInTheDocument()
    })
  })

  it('shows checkboxes in search results', async () => {
    const foxSparrow = {
      ebird_code: 'foxspa',
      common_name: 'Fox Sparrow',
      scientific_name: 'Passerella iliaca',
      image_url: null,
    }
    const { createQuery } = await import('@tanstack/svelte-query')
    vi.mocked(createQuery).mockReturnValue(
      queryResult({
        data: [foxSparrow],
        isPending: false,
        isError: false,
      }),
    )
    vi.stubGlobal('fetch', makeFetch())
    render(DeckDetailPage)
    await vi.waitFor(() => screen.getByPlaceholderText(/search species/i))
    await fireEvent.input(screen.getByPlaceholderText(/search species/i), {
      target: { value: 'fox' },
    })
    await vi.waitFor(() => {
      expect(screen.getAllByRole('checkbox').length).toBeGreaterThan(0)
    })
  })

  it('shows bulk add bar when search results are checked', async () => {
    const foxSparrow = {
      ebird_code: 'foxspa',
      common_name: 'Fox Sparrow',
      scientific_name: 'Passerella iliaca',
      image_url: null,
    }
    const { createQuery } = await import('@tanstack/svelte-query')
    vi.mocked(createQuery).mockReturnValue(
      queryResult({
        data: [foxSparrow],
        isPending: false,
        isError: false,
      }),
    )
    vi.stubGlobal('fetch', makeFetch())
    render(DeckDetailPage)
    await vi.waitFor(() => screen.getByPlaceholderText(/search species/i))
    await fireEvent.input(screen.getByPlaceholderText(/search species/i), {
      target: { value: 'fox' },
    })
    await vi.waitFor(() => screen.getByRole('checkbox', { name: /fox sparrow/i }))
    await fireEvent.click(screen.getByRole('checkbox', { name: /fox sparrow/i }))
    await vi.waitFor(() => {
      expect(screen.getByRole('toolbar')).toBeInTheDocument()
      expect(screen.getByText(/1 selected/i)).toBeInTheDocument()
    })
  })

  it('checked species stay visible when search query changes', async () => {
    const foxSparrow = {
      ebird_code: 'foxspa',
      common_name: 'Fox Sparrow',
      scientific_name: 'Passerella iliaca',
      image_url: null,
    }
    const junco = {
      ebird_code: 'daejun',
      common_name: 'Dark-eyed Junco',
      scientific_name: 'Junco hyemalis',
      image_url: null,
    }
    const { createQuery } = await import('@tanstack/svelte-query')
    vi.mocked(createQuery).mockReturnValue(
      queryResult({
        data: [foxSparrow, junco],
        isPending: false,
        isError: false,
      }),
    )
    vi.stubGlobal('fetch', makeFetch())
    render(DeckDetailPage)
    const input = screen.getByPlaceholderText(/search species/i)
    await fireEvent.input(input, { target: { value: 'fox' } })
    await vi.waitFor(() => screen.getAllByRole('checkbox'))
    await fireEvent.click(screen.getAllByRole('checkbox')[0])
    // Now search for junco -- fox sparrow no longer in search results but should stay pinned
    await fireEvent.input(input, { target: { value: 'junco' } })
    await vi.waitFor(() => {
      expect(screen.getAllByText(/fox sparrow/i).length).toBeGreaterThan(0)
      expect(screen.getAllByText(/dark-eyed junco/i).length).toBeGreaterThan(0)
    })
  })

  it('bulk add posts species_codes to bulk endpoint and clears selection', async () => {
    const foxSparrow = {
      ebird_code: 'foxspa',
      common_name: 'Fox Sparrow',
      scientific_name: 'Passerella iliaca',
      image_url: null,
    }
    const { createQuery } = await import('@tanstack/svelte-query')
    vi.mocked(createQuery).mockReturnValue(
      queryResult({
        data: [foxSparrow],
        isPending: false,
        isError: false,
      }),
    )
    let bulkCalled = false
    let bulkBody: unknown = null
    vi.stubGlobal(
      'fetch',
      vi.fn().mockImplementation((url: string, opts?: RequestInit) => {
        if (opts?.method === 'POST' && url.includes('/bulk')) {
          bulkCalled = true
          bulkBody = JSON.parse(opts.body as string)
          return Promise.resolve({ ok: true, json: () => Promise.resolve({ added: 1 }) })
        }
        if (url.match(/\/api\/v1\/decks\/\d+\/species$/)) {
          return Promise.resolve({ ok: true, json: () => Promise.resolve(speciesWithPrefs) })
        }
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({ audio_due: 0, image_due: 0 }),
        })
      }),
    )
    render(DeckDetailPage)
    await vi.waitFor(() => screen.getByPlaceholderText(/search species/i))
    await fireEvent.input(screen.getByPlaceholderText(/search species/i), {
      target: { value: 'fox' },
    })
    await vi.waitFor(() => screen.getByRole('checkbox', { name: /fox sparrow/i }))
    await fireEvent.click(screen.getByRole('checkbox', { name: /fox sparrow/i }))
    await vi.waitFor(() => screen.getByRole('button', { name: /add 1 species/i }))
    await fireEvent.click(screen.getByRole('button', { name: /add 1 species/i }))
    await vi.waitFor(() => {
      expect(bulkCalled).toBe(true)
      expect((bulkBody as Record<string, unknown>).species_codes).toContain('foxspa')
    })
    await vi.waitFor(() => {
      expect(screen.queryByRole('toolbar')).toBeNull()
    })
  })

  it('sends updated description in PATCH when description edited', async () => {
    let patchBody: Record<string, unknown> | null = null
    vi.stubGlobal(
      'fetch',
      vi.fn().mockImplementation((url: string, opts?: RequestInit) => {
        if (opts?.method === 'PATCH') {
          patchBody = JSON.parse(opts.body as string)
          return Promise.resolve({ ok: true, status: 204 })
        }
        if (opts?.method === 'DELETE') return Promise.resolve({ ok: true, status: 204 })
        if (url.match(/\/api\/v1\/decks\/\d+\/species/)) {
          return Promise.resolve({ ok: true, json: () => Promise.resolve([]) })
        }
        return Promise.resolve({
          ok: true,
          json: () =>
            Promise.resolve({
              name: 'My Deck',
              description: 'Old desc',
              audio_due: 0,
              image_due: 0,
            }),
        })
      }),
    )
    render(DeckDetailPage)
    await vi.waitFor(() => screen.getByText(/old desc/i))
    await fireEvent.click(screen.getByRole('button', { name: /edit description/i }))
    const input = screen.getByDisplayValue('Old desc')
    await fireEvent.input(input, { target: { value: 'New desc' } })
    await fireEvent.blur(input)
    await vi.waitFor(() => {
      expect(patchBody?.description).toBe('New desc')
    })
  })
})
