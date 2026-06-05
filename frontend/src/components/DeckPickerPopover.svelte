<script lang="ts">
  import { useQueryClient } from '@tanstack/svelte-query'
  import { apiPost } from '$lib/api'
  import { createDecksQuery, queryKeys } from '$lib/queries'
  import type { Deck } from '../types'
  import DeckCreateForm from './DeckCreateForm.svelte'

  interface Props {
    onPick: (deckId: number) => void
    onClose: () => void
  }
  let { onPick, onClose }: Props = $props()

  const queryClient = useQueryClient()
  const decksQuery = createDecksQuery()

  const decks = $derived(decksQuery.data?.decks ?? [])
  const loading = $derived(decksQuery.isPending)

  async function createAndPick(name: string) {
    const deck = await apiPost<Deck>('/api/v1/decks', { name })
    queryClient.invalidateQueries({ queryKey: queryKeys.decks })
    onPick(deck.id)
  }
</script>

<div
  class="backdrop"
  data-testid="backdrop"
  role="presentation"
  onclick={onClose}
  onkeydown={(e) => e.key === 'Escape' && onClose()}
></div>
<div class="popover" role="dialog" aria-label="Pick a deck">
  <p class="popover-title">Add to deck</p>
  <DeckCreateForm label="+ Create & add" onCreate={createAndPick} onEscape={onClose} />
  {#if loading}
    <p class="status">Loading...</p>
  {:else if decks.length === 0}
    <p class="status">No decks yet. <a href="/decks">Create one first.</a></p>
  {:else}
    <ul class="deck-list list-reset">
      {#each decks as deck (deck.id)}
        <li>
          <button class="deck-item" onclick={() => onPick(deck.id)}>{deck.name}</button>
        </li>
      {/each}
    </ul>
  {/if}
</div>

<style>
  .backdrop {
    position: fixed;
    inset: 0;
    z-index: 20;
    background: transparent;
  }
  .popover {
    position: fixed;
    bottom: 5rem;
    left: 50%;
    transform: translateX(-50%);
    width: min(320px, calc(100vw - 2rem));
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: 12px;
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.2);
    z-index: 21;
    padding: 0.75rem 0;
    max-height: 60vh;
    overflow-y: auto;
  }
  .popover-title {
    font-size: 0.8125rem;
    font-weight: 600;
    color: var(--text-muted);
    padding: 0 1rem 0.5rem;
    margin: 0;
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }
  .deck-item {
    display: block;
    width: 100%;
    text-align: left;
    background: transparent;
    border: none;
    padding: 0.625rem 1rem;
    font-size: 0.9375rem;
    color: var(--text);
    cursor: pointer;
    font-family: inherit;
  }
  .deck-item:hover {
    background: color-mix(in srgb, var(--accent) 8%, transparent);
  }
  .status {
    padding: 0.5rem 1rem;
    font-size: 0.875rem;
    color: var(--text-muted);
  }
</style>
