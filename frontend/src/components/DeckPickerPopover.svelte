<script lang="ts">
  import { useQueryClient } from '@tanstack/svelte-query'
  import { apiPost } from '$lib/api'
  import { createDecksQuery, queryKeys } from '$lib/queries'
  import type { Deck } from '../types'

  interface Props {
    onPick: (deckId: number) => void
    onClose: () => void
  }
  let { onPick, onClose }: Props = $props()

  const queryClient = useQueryClient()
  const decksQuery = createDecksQuery()

  const decks = $derived(decksQuery.data?.decks ?? [])
  const loading = $derived(decksQuery.isPending)

  let newDeckName = $state('')
  let creating = $state(false)

  async function createAndPick() {
    const name = newDeckName.trim()
    if (!name) return
    creating = true
    try {
      const deck = await apiPost<Deck>('/api/v1/decks', { name })
      queryClient.invalidateQueries({ queryKey: queryKeys.decks })
      onPick(deck.id)
    } catch {
      // creation failed; input stays for retry
    } finally {
      creating = false
    }
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
  <div class="create-section">
    <input
      class="create-input"
      type="text"
      placeholder="New deck name…"
      bind:value={newDeckName}
      onkeydown={(e) => e.key === 'Enter' && createAndPick()}
    />
    <button
      class="create-btn"
      onclick={createAndPick}
      disabled={!newDeckName.trim() || creating}
    >{creating ? 'Creating…' : '+ Create & add'}</button>
  </div>
  {#if loading}
    <p class="status">Loading...</p>
  {:else if decks.length === 0}
    <p class="status">No decks yet. <a href="/decks">Create one first.</a></p>
  {:else}
    <ul class="deck-list">
      {#each decks as deck (deck.id)}
        <li>
          <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_noninteractive_element_interactions -->
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
  .create-section {
    padding: 0.625rem 0.75rem;
    border-bottom: 1px solid var(--border);
    display: flex;
    flex-direction: column;
    gap: 0.375rem;
  }
  .create-input {
    background: var(--bg);
    border: 1px solid var(--border);
    color: var(--text);
    border-radius: 6px;
    padding: 0.375rem 0.5rem;
    font-size: 0.8125rem;
    font-family: inherit;
    width: 100%;
    box-sizing: border-box;
  }
  .create-btn {
    background: var(--accent);
    color: #fff;
    border: none;
    border-radius: 6px;
    padding: 0.3125rem 0.625rem;
    font-size: 0.75rem;
    font-weight: 600;
    cursor: pointer;
    font-family: inherit;
    width: 100%;
  }
  .create-btn:disabled {
    opacity: 0.6;
    cursor: not-allowed;
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
  .deck-list {
    list-style: none;
    padding: 0;
    margin: 0;
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
