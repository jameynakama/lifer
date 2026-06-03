<script lang="ts">
  import { goto } from '$app/navigation'
  import type { Deck } from '../../types'

  let decks: Deck[] = $state([])
  let loading = $state(true)
  let newName = $state('')
  let creating = $state(false)
  let practiceMode = $state(false)

  async function loadDecks() {
    try {
      const res = await fetch('/api/v1/decks')
      if (res.ok) decks = await res.json()
    } catch {
      // network error, loading still ends
    } finally {
      loading = false
    }
  }

  async function createDeck() {
    if (!newName.trim()) return
    creating = true
    try {
      const res = await fetch('/api/v1/decks', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: newName.trim() }),
      })
      if (res.ok) {
        const created = await res.json()
        decks = [...decks, { ...created, audio_due: 0, image_due: 0 }]
        newName = ''
      }
    } finally {
      creating = false
    }
  }

  async function deleteDeck(id: number) {
    try {
      const res = await fetch(`/api/v1/decks/${id}`, { method: 'DELETE' })
      if (res.ok) {
        decks = decks.filter((d) => d.id !== id)
      }
    } catch {
      // network error, leave state unchanged
    }
  }

  loadDecks()
</script>

<div class="decks-page">
  <div class="page-header">
    <h1>Decks</h1>
    <button
      class="btn-toggle"
      class:active={practiceMode}
      onclick={() => (practiceMode = !practiceMode)}
    >
      Free Practice
    </button>
  </div>

  {#if practiceMode}
    <p class="practice-banner">
      Free practice mode -- answers won't affect your spaced repetition schedule
    </p>
  {/if}

  <form class="create-form" onsubmit={(e) => { e.preventDefault(); createDeck() }}>
    <input
      type="text"
      bind:value={newName}
      placeholder="Deck name"
      disabled={creating}
    />
    <button type="submit" disabled={creating || !newName.trim()}>Create</button>
  </form>

  {#if loading}
    <p class="status">Loading...</p>
  {:else if decks.length === 0}
    <p class="empty">No decks yet. Create your first one above.</p>
  {:else}
    <ul class="deck-list">
      {#each decks as deck (deck.id)}
        <li class="deck-row">
          <a href="/decks/{deck.id}" class="deck-name">{deck.name}</a>
          <div class="deck-meta">
            {#if practiceMode}
              <button
                class="btn-practice-quick"
                onclick={() => goto(`/decks/${deck.id}/practice?lane=audio`)}
              >▶ Audio</button>
              <button
                class="btn-practice-quick"
                onclick={() => goto(`/decks/${deck.id}/practice?lane=image`)}
              >◉ Image</button>
            {:else}
              {#if deck.audio_due > 0}
                <span class="due-badge">🔊 {deck.audio_due}</span>
              {/if}
              {#if deck.image_due > 0}
                <span class="due-badge">👁 {deck.image_due}</span>
              {/if}
            {/if}
            <button class="btn-delete" onclick={() => deleteDeck(deck.id)}>Delete</button>
          </div>
        </li>
      {/each}
    </ul>
  {/if}
</div>

<style>
  .decks-page {
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
  }
  .page-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }
  h1 {
    font-size: 1.25rem;
    font-weight: 700;
    color: var(--text);
    margin: 0;
  }
  .btn-toggle {
    background: transparent;
    border: 1px solid var(--border);
    color: var(--text-muted);
    border-radius: 8px;
    padding: 0.375rem 0.75rem;
    font-size: 0.8125rem;
    font-weight: 600;
    cursor: pointer;
    font-family: inherit;
  }
  .btn-toggle.active {
    background: var(--accent);
    border-color: var(--accent);
    color: #fff;
  }
  .practice-banner {
    background: color-mix(in srgb, var(--accent) 12%, transparent);
    border: 1px solid color-mix(in srgb, var(--accent) 30%, transparent);
    border-radius: 8px;
    padding: 0.625rem 0.875rem;
    font-size: 0.8125rem;
    color: var(--text);
    margin: 0;
  }
  .create-form {
    display: flex;
    gap: 0.5rem;
  }
  .create-form input {
    flex: 1;
    background: var(--surface);
    border: 1px solid var(--border);
    color: var(--text);
    border-radius: 8px;
    padding: 0.5rem 0.75rem;
    font-size: 0.9375rem;
    font-family: inherit;
  }
  .create-form button {
    background: var(--accent);
    color: #fff;
    border: none;
    border-radius: 8px;
    padding: 0.5rem 1rem;
    font-size: 0.875rem;
    font-weight: 600;
    cursor: pointer;
    font-family: inherit;
  }
  .create-form button:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
  .deck-list {
    list-style: none;
    padding: 0;
    margin: 0;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }
  .deck-row {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: 10px;
    padding: 0.875rem 1rem;
    display: flex;
    align-items: center;
    gap: 0.75rem;
    box-shadow: var(--shadow);
  }
  .deck-name {
    color: var(--text);
    font-weight: 600;
    text-decoration: none;
    font-size: 0.9375rem;
    flex: 1;
  }
  .deck-meta {
    display: flex;
    align-items: center;
    gap: 0.375rem;
    flex-shrink: 0;
  }
  .due-badge {
    font-size: 0.75rem;
    font-weight: 600;
    color: var(--text-secondary);
    background: var(--border);
    border-radius: 6px;
    padding: 0.1875rem 0.5rem;
  }
  .btn-practice-quick {
    background: transparent;
    border: 1px solid var(--accent);
    color: var(--accent);
    border-radius: 6px;
    padding: 0.25rem 0.625rem;
    font-size: 0.75rem;
    font-weight: 600;
    cursor: pointer;
    font-family: inherit;
  }
  .btn-delete {
    background: transparent;
    border: 1px solid var(--border);
    color: var(--text-muted);
    border-radius: 6px;
    padding: 0.25rem 0.5rem;
    font-size: 0.75rem;
    cursor: pointer;
    font-family: inherit;
  }
  .status, .empty {
    text-align: center;
    color: var(--text-muted);
    padding: 2rem 0;
  }
</style>
