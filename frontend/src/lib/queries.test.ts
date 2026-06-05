import { afterEach, describe, expect, it, vi } from 'vitest'
import { decksQueryOptions, presetsQueryOptions, queryKeys } from './queries'

afterEach(() => {
  vi.unstubAllGlobals()
})

function jsonResponse(data: unknown) {
  return new Response(JSON.stringify(data), {
    status: 200,
    headers: { 'Content-Type': 'application/json' },
  })
}

describe('queryKeys', () => {
  it('should derive species keys from the code', () => {
    expect(queryKeys.species('spotto')).toEqual(['species', 'spotto'])
    expect(queryKeys.speciesDecks('spotto')).toEqual(['species', 'spotto', 'decks'])
  })

  it('should keep presets distinct from the decks key', () => {
    expect(queryKeys.presets).not.toEqual(queryKeys.decks)
  })
})

describe('decksQueryOptions', () => {
  it('should fetch the user deck list under the shared key', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(jsonResponse({ decks: [{ id: 1, name: 'Yard' }], next_due_at: null }))
    )

    const opts = decksQueryOptions()
    const data = await opts.queryFn()

    expect(opts.queryKey).toEqual(queryKeys.decks)
    expect(fetch).toHaveBeenCalledWith('/api/v1/decks', expect.anything())
    expect(data.decks[0].name).toBe('Yard')
    expect(data.next_due_at).toBeNull()
  })
})

describe('presetsQueryOptions', () => {
  it('should fetch preset decks under the shared key', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(jsonResponse([{ id: 2, name: 'Oregon', species_count: 120 }]))
    )

    const opts = presetsQueryOptions()
    const data = await opts.queryFn()

    expect(opts.queryKey).toEqual(queryKeys.presets)
    expect(fetch).toHaveBeenCalledWith('/api/v1/decks/presets', expect.anything())
    expect(data[0].species_count).toBe(120)
  })
})
