/**
 * Shared TanStack Query keys and query factories. Every consumer of the same
 * endpoint must use the same key, or invalidations won't propagate.
 */
import { createQuery } from '@tanstack/svelte-query'
import { apiGet } from './api'
import type { DecksResponse, PresetDeck, StatsResponse } from '../types'

export const queryKeys = {
  decks: ['decks'] as const,
  presets: ['decks', 'presets'] as const,
  speciesAll: ['species', 'all'] as const,
  species: (code: string) => ['species', code] as const,
  speciesDecks: (code: string) => ['species', code, 'decks'] as const,
  stats: (lane: string) => ['stats', lane] as const,
  /** Prefix of every stats(lane) key -- invalidates all lane tabs at once. */
  statsAll: ['stats'] as const,
}

export const decksQueryOptions = () => ({
  queryKey: queryKeys.decks,
  queryFn: () => apiGet<DecksResponse>('/api/v1/decks'),
})

export const presetsQueryOptions = () => ({
  queryKey: queryKeys.presets,
  queryFn: () => apiGet<PresetDeck[]>('/api/v1/decks/presets'),
})

export function statsQueryOptions(lane: '' | 'audio' | 'image') {
  return {
    queryKey: queryKeys.stats(lane || 'combined'),
    queryFn: () => apiGet<StatsResponse>(`/api/v1/stats${lane ? `?lane=${lane}` : ''}`),
  }
}

/** The user's decks with due counts and next_due_at. Shared key: ['decks']. */
export function createDecksQuery() {
  return createQuery(() => decksQueryOptions())
}

/** Preset (eBird region) decks. Shared key: ['decks', 'presets']. */
export function createPresetsQuery() {
  return createQuery(() => presetsQueryOptions())
}
