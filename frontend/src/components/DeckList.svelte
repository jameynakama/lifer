<script lang="ts">
  import type { Deck } from '../types'

  let {
    decks,
    onPractice,
  }: {
    decks: Deck[]
    onPractice: (deck: Deck, lane: 'audio' | 'image') => void
  } = $props()
</script>

<div class="deck-list">
  {#each decks as deck (deck.id)}
    <div class="deck-card card">
      <a href="/decks/{deck.id}" class="deck-name">{deck.name}</a>
      <div class="deck-actions">
        {#if deck.audio_due > 0}
          <button class="btn-lane" onclick={() => onPractice(deck, 'audio')}>
            🔊 Audio · {deck.audio_due}
          </button>
        {/if}
        {#if deck.image_due > 0}
          <button class="btn-lane" onclick={() => onPractice(deck, 'image')}>
            👁 Image · {deck.image_due}
          </button>
        {/if}
        {#if deck.audio_due === 0 && deck.image_due === 0}
          <span class="all-done">All done</span>
        {/if}
      </div>
    </div>
  {/each}
</div>

<style>
  .deck-list {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 0.625rem;
  }

  @media (max-width: 639px) {
    .deck-list {
      grid-template-columns: 1fr;
    }
  }
  .deck-card {
    padding: 0.875rem 1rem;
    display: flex;
    justify-content: space-between;
    align-items: center;
    box-shadow: var(--shadow-stacked);
  }
  .deck-name {
    flex: 1;
    font-size: 0.9375rem;
    font-weight: 600;
    color: var(--text);
    text-decoration: none;
  }
  .deck-actions {
    display: flex;
    gap: 0.5rem;
  }
  .btn-lane {
    background: var(--surface);
    border: 1px solid var(--border);
    color: var(--text-secondary);
    border-radius: 8px;
    padding: 0.375rem 0.625rem;
    font-size: 0.75rem;
    font-weight: 600;
    cursor: pointer;
    font-family: inherit;
    box-shadow: var(--shadow);
  }
  .all-done {
    font-size: 0.75rem;
    color: var(--text-muted);
  }
</style>
