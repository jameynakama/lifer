<script lang="ts">
  import { createQuery, createMutation, useQueryClient } from '@tanstack/svelte-query'
  import { apiDelete, apiGet, apiPost } from '$lib/api'
  import { createDecksQuery, queryKeys } from '$lib/queries'
  import type { Deck } from '../types'
  import DeckCreateForm from './DeckCreateForm.svelte'

  let {
    ebird_code,
    onClose,
  }: {
    ebird_code: string
    onClose: () => void
  } = $props()

  const queryClient = useQueryClient()

  const decksQuery = createDecksQuery()
  const decks = $derived(decksQuery.data?.decks ?? [])

  const membershipQuery = createQuery(() => ({
    queryKey: queryKeys.speciesDecks(ebird_code),
    queryFn: () =>
      apiGet<{ deck_ids: number[] | null }>(`/api/v1/species/${ebird_code}/decks`).then(
        (d) => d.deck_ids ?? []
      ),
  }))

  const addMutation = createMutation(() => ({
    mutationFn: (deckId: number) => apiPost(`/api/v1/decks/${deckId}/species`, { ebird_code }),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: queryKeys.speciesDecks(ebird_code) }),
  }))

  const removeMutation = createMutation(() => ({
    mutationFn: (deckId: number) => apiDelete(`/api/v1/decks/${deckId}/species/${ebird_code}`),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: queryKeys.speciesDecks(ebird_code) }),
  }))

  let mutationError = $state('')

  function toggle(deckId: number, currentlyIn: boolean) {
    mutationError = ''
    if (currentlyIn) {
      removeMutation.mutate(deckId, {
        onError: () => { mutationError = 'Failed. Try again.' },
      })
    } else {
      addMutation.mutate(deckId, {
        onError: () => { mutationError = 'Failed. Try again.' },
      })
    }
  }

  async function createDeck(name: string) {
    mutationError = ''
    try {
      const deck = await apiPost<Deck>('/api/v1/decks', { name })
      await apiPost(`/api/v1/decks/${deck.id}/species`, { ebird_code })
      queryClient.invalidateQueries({ queryKey: queryKeys.decks })
      queryClient.invalidateQueries({ queryKey: queryKeys.speciesDecks(ebird_code) })
    } catch (e) {
      mutationError = 'Failed. Try again.'
      throw e // form keeps the input for retry
    }
  }
</script>

<div
  class="dropdown"
  role="dialog"
  aria-label="Add to deck"
  tabindex="-1"
  onclick={(e) => e.stopPropagation()}
  onkeydown={(e) => e.stopPropagation()}
>
  <DeckCreateForm label="+ Create deck" onCreate={createDeck} onEscape={onClose} />

  <div class="decks-list">
    {#if decksQuery.isPending || membershipQuery.isPending}
      <p class="loading-msg">Loading…</p>
    {:else if decks.length === 0}
      <p class="loading-msg">No decks yet.</p>
    {:else}
      {#each decks as deck (deck.id)}
        {@const isMember = (membershipQuery.data ?? []).includes(deck.id)}
        <label class="deck-item">
          <input
            type="checkbox"
            checked={isMember}
            onchange={() => toggle(deck.id, isMember)}
          />
          {deck.name}
        </label>
      {/each}
    {/if}
  </div>

  {#if mutationError}
    <p class="error-msg">{mutationError}</p>
  {/if}
</div>

<style>
  .dropdown {
    position: absolute;
    top: calc(100% + 4px);
    right: 0;
    width: 220px;
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: 10px;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
    z-index: 100;
    overflow: hidden;
  }

  .decks-list {
    max-height: 200px;
    overflow-y: auto;
    padding: 0.375rem 0;
  }

  .deck-item {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.5rem 0.75rem;
    cursor: pointer;
    font-size: 0.875rem;
    color: var(--text);
  }

  .deck-item:hover {
    background: var(--bg);
  }

  .deck-item input[type='checkbox'] {
    accent-color: var(--accent);
    flex-shrink: 0;
  }

  .loading-msg {
    padding: 0.5rem 0.75rem;
    font-size: 0.8125rem;
    color: var(--text-muted);
    margin: 0;
  }

  .error-msg {
    padding: 0.25rem 0.75rem 0.5rem;
    font-size: 0.75rem;
    color: var(--danger);
    margin: 0;
  }
</style>
