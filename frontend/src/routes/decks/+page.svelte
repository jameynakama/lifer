<script lang="ts">
  import { goto } from '$app/navigation'
  import type { Deck, DecksResponse, PresetDeck } from '../../types'
  import PresetDeckList from '$components/PresetDeckList.svelte'

  let decks: Deck[] = $state([])
  let presetDecks: PresetDeck[] = $state([])
  let loading = $state(true)
  let presetsLoading = $state(true)
  let newName = $state('')
  let newDescription = $state('')
  let creating = $state(false)
  let practiceMode = $state(false)
  let cloning: Set<number> = $state(new Set())

  async function loadDecks() {
    try {
      const res = await fetch('/api/v1/decks')
      if (res.ok) {
        const data: DecksResponse = await res.json()
        decks = data.decks
      }
    } catch {
      // network error, loading still ends
    } finally {
      loading = false
    }
  }

  async function loadPresetDecks() {
    try {
      const res = await fetch('/api/v1/decks/presets')
      if (res.ok) presetDecks = await res.json()
    } catch {
      // network error
    } finally {
      presetsLoading = false
    }
  }

  async function cloneDeck(id: number) {
    cloning = new Set([...cloning, id])
    try {
      const res = await fetch(`/api/v1/decks/${id}/clone`, { method: 'POST' })
      if (res.ok) {
        const created = await res.json()
        goto(`/decks/${created.id}`)
      }
    } finally {
      cloning = new Set([...cloning].filter((c) => c !== id))
    }
  }

  async function createDeck() {
    if (!newName.trim()) return
    creating = true
    try {
      const res = await fetch('/api/v1/decks', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: newName.trim(), description: newDescription.trim() || undefined }),
      })
      if (res.ok) {
        const created = await res.json()
        decks = [...decks, { ...created, audio_due: 0, image_due: 0 }]
        newName = ''
        newDescription = ''
      }
    } finally {
      creating = false
    }
  }

  async function deleteDeck(id: number) {
    if (!confirm('Delete this deck? This cannot be undone.')) return
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
  loadPresetDecks()
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
    <div class="create-fields">
      <input
        type="text"
        bind:value={newName}
        placeholder="Deck name"
        disabled={creating}
      />
      <input
        type="text"
        bind:value={newDescription}
        placeholder="Description (optional)"
        disabled={creating}
      />
    </div>
    <button type="submit" disabled={creating || !newName.trim()}>Create</button>
  </form>

  {#if loading}
    <p class="status">Loading...</p>
  {:else if decks.length === 0}
    <p class="empty">No decks yet. Create one to get started, or clone a Starter Deck below.</p>
  {:else}
    <ul class="deck-list">
      {#each decks as deck (deck.id)}
        <li class="deck-row">
          <div class="deck-info">
            <a href="/decks/{deck.id}" class="deck-name">{deck.name}</a>
            {#if deck.description}
              <span class="deck-description">{deck.description}</span>
            {/if}
          </div>
          <div class="deck-meta">
            {#if practiceMode}
              <button
                class="btn-action"
                onclick={() => goto(`/decks/${deck.id}/practice?lane=audio`)}
              >▶ Audio</button>
              <button
                class="btn-action"
                onclick={() => goto(`/decks/${deck.id}/practice?lane=image`)}
              >◉ Image</button>
            {:else}
              {#if deck.audio_due > 0}
                <button
                  class="btn-action"
                  onclick={() => goto(`/decks/${deck.id}/quiz?lane=audio`)}
                >🔊 Audio · {deck.audio_due}</button>
              {/if}
              {#if deck.image_due > 0}
                <button
                  class="btn-action"
                  onclick={() => goto(`/decks/${deck.id}/quiz?lane=image`)}
                >👁 Image · {deck.image_due}</button>
              {/if}
              {#if deck.audio_due === 0 && deck.image_due === 0}
                <span class="all-done">All done</span>
              {/if}
            {/if}
            <button class="btn-delete" onclick={() => deleteDeck(deck.id)}>Delete</button>
          </div>
        </li>
      {/each}
    </ul>
  {/if}
  {#if !presetsLoading && presetDecks.length > 0}
    <hr class="section-divider" />
    <h2 class="section-heading">Starter Decks</h2>
    <p class="section-subheading">Clone one to get started instantly</p>
    <PresetDeckList {presetDecks} {cloning} onClone={cloneDeck} />
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
    align-items: flex-start;
  }
  .create-fields {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 0.375rem;
  }
  .create-form input {
    width: 100%;
    background: var(--surface);
    border: 1px solid var(--border);
    color: var(--text);
    border-radius: 8px;
    padding: 0.5rem 0.75rem;
    font-size: 0.9375rem;
    font-family: inherit;
    box-sizing: border-box;
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
    gap: 1rem;
  }
  .deck-row {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: 10px;
    padding: 0.875rem 1rem;
    display: flex;
    align-items: center;
    gap: 0.75rem;
    box-shadow:
      -1px 2px 0 0 var(--surface),
      -1px 2px 0 1px var(--border),
      -3px 5px 0 0 var(--surface),
      -3px 5px 0 1px var(--border),
      var(--shadow);
  }
  .deck-info {
    display: flex;
    flex-direction: column;
    gap: 0.125rem;
    flex: 1;
    min-width: 0;
  }
  .deck-name {
    color: var(--text);
    font-weight: 600;
    text-decoration: none;
    font-size: 0.9375rem;
  }
  .deck-description {
    font-size: 0.8125rem;
    color: var(--text-muted);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .deck-meta {
    display: flex;
    align-items: center;
    gap: 0.375rem;
    flex-shrink: 0;
  }
  .btn-action {
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
  .btn-delete:hover {
    border-color: #ef4444;
    color: #ef4444;
  }
  .status, .empty {
    text-align: center;
    color: var(--text-muted);
    padding: 2rem 0;
  }
  .section-divider {
    border: none;
    border-top: 1px solid var(--border);
    margin: 0;
  }
  .section-heading {
    font-size: 1rem;
    font-weight: 700;
    color: var(--text);
    margin: 0;
  }
  .section-subheading {
    font-size: 0.8125rem;
    color: var(--text-muted);
    margin: -0.75rem 0 0;
  }
</style>
