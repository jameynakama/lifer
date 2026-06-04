<script lang="ts">
  import type { PresetDeck } from '../../../types'

  interface UserDeck {
    id: number
    name: string
    description: string
    owner_name: string
    owner_email: string
    species_count: number
  }

  let presets: PresetDeck[] = $state([])
  let userDecks: UserDeck[] = $state([])
  let loading = $state(true)
  let userDecksLoading = $state(true)
  let newName = $state('')
  let newDescription = $state('')
  let creating = $state(false)

  async function loadPresets() {
    try {
      const res = await fetch('/api/v1/decks/presets')
      if (res.ok) presets = await res.json()
    } catch {
      // network error
    } finally {
      loading = false
    }
  }

  async function loadUserDecks() {
    try {
      const res = await fetch('/api/v1/admin/decks')
      if (res.ok) userDecks = await res.json()
    } catch {
      // network error
    } finally {
      userDecksLoading = false
    }
  }

  async function createPreset() {
    if (!newName.trim()) return
    creating = true
    try {
      const res = await fetch('/api/v1/admin/decks', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: newName.trim(), description: newDescription.trim() || undefined }),
      })
      if (res.ok) {
        const created = await res.json()
        presets = [...presets, { ...created, species_count: 0 }]
        newName = ''
        newDescription = ''
      }
    } finally {
      creating = false
    }
  }

  async function deletePreset(id: number) {
    if (!confirm('Delete this preset deck? This cannot be undone.')) return
    const res = await fetch(`/api/v1/admin/decks/${id}`, { method: 'DELETE' })
    if (res.ok) presets = presets.filter((p) => p.id !== id)
  }

  loadPresets()
  loadUserDecks()
</script>

<div class="admin-decks">
  <section>
    <h1>Preset Decks</h1>

    <form class="create-form" onsubmit={(e) => { e.preventDefault(); createPreset() }}>
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
    {:else if presets.length === 0}
      <p class="empty">No preset decks yet.</p>
    {:else}
      <ul class="preset-list">
        {#each presets as preset (preset.id)}
          <li class="preset-row">
            <div class="preset-info">
              <a href="/admin/decks/{preset.id}" class="preset-name">{preset.name}</a>
              {#if preset.description}
                <span class="preset-description">{preset.description}</span>
              {/if}
              <span class="species-count">{preset.species_count} species</span>
            </div>
            <div class="preset-actions">
              <a href="/decks/{preset.id}" class="btn-manage">Manage Species</a>
              <button class="btn-delete" onclick={() => deletePreset(preset.id)}>Delete</button>
            </div>
          </li>
        {/each}
      </ul>
    {/if}
  </section>

  <section>
    <h1>User Decks</h1>

    {#if userDecksLoading}
      <p class="status">Loading...</p>
    {:else if userDecks.length === 0}
      <p class="empty">No user decks yet.</p>
    {:else}
      <ul class="user-deck-list">
        {#each userDecks as deck (deck.id)}
          <li>
            <div class="deck-main">
              <a href="/admin/decks/{deck.id}">{deck.name}</a>
              <span class="muted">{deck.species_count} species</span>
            </div>
            <span class="muted">{deck.owner_name} &lt;{deck.owner_email}&gt;</span>
          </li>
        {/each}
      </ul>
    {/if}
  </section>
</div>

<style>
  .admin-decks {
    display: flex;
    flex-direction: column;
    gap: 2rem;
  }
  h1 {
    font-size: 1.25rem;
    font-weight: 700;
    color: var(--text);
    margin: 0 0 1rem 0;
  }
  .create-form {
    display: flex;
    gap: 0.5rem;
    align-items: flex-start;
    margin-bottom: 1rem;
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
  .preset-list {
    list-style: none;
    padding: 0;
    margin: 0;
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }
  .preset-row {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: 10px;
    padding: 0.875rem 1rem;
    display: flex;
    align-items: center;
    gap: 0.75rem;
    box-shadow: var(--shadow);
  }
  .preset-info {
    display: flex;
    flex-direction: column;
    gap: 0.125rem;
    flex: 1;
    min-width: 0;
  }
  .preset-name {
    font-weight: 600;
    font-size: 0.9375rem;
    color: var(--text);
    text-decoration: none;
  }
  .preset-name:hover { color: var(--accent); }
  .preset-description {
    font-size: 0.8125rem;
    color: var(--text-muted);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .species-count {
    font-size: 0.75rem;
    color: var(--text-muted);
  }
  .preset-actions {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    flex-shrink: 0;
  }
  .btn-manage {
    background: var(--surface);
    border: 1px solid var(--accent);
    color: var(--accent);
    border-radius: 8px;
    padding: 0.375rem 0.625rem;
    font-size: 0.75rem;
    font-weight: 600;
    text-decoration: none;
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
  .user-deck-list {
    list-style: none;
    padding: 0;
    margin: 0;
  }
  .user-deck-list li {
    display: flex;
    flex-direction: column;
    gap: 0.1rem;
    padding: 0.4rem 0;
    border-bottom: 1px solid var(--border);
  }
  .deck-main {
    display: flex;
    align-items: baseline;
    gap: 0.5rem;
  }
  .user-deck-list a {
    color: var(--text);
    text-decoration: none;
  }
  .user-deck-list a:hover { color: var(--accent); }
  .muted {
    color: var(--text-muted);
    font-size: 0.8125rem;
  }
  .status, .empty {
    color: var(--text-muted);
    padding: 1rem 0;
  }
</style>
