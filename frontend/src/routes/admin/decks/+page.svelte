<script lang="ts">
  import { apiDelete, apiGet, apiPost } from '$lib/api'
  import type { PresetDeck, UserDeck } from '../../../types'

  let presets: PresetDeck[] = $state([])
  let userDecks: UserDeck[] = $state([])
  let loading = $state(true)
  let userDecksLoading = $state(true)
  let newName = $state('')
  let newDescription = $state('')
  let creating = $state(false)

  async function loadPresets() {
    try {
      presets = await apiGet('/api/v1/decks/presets')
    } catch {
      // network error
    } finally {
      loading = false
    }
  }

  async function loadUserDecks() {
    try {
      userDecks = await apiGet('/api/v1/admin/decks')
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
      const created = await apiPost<PresetDeck>('/api/v1/admin/decks', {
        name: newName.trim(),
        description: newDescription.trim() || undefined,
      })
      presets = [...presets, { ...created, species_count: 0 }]
      newName = ''
      newDescription = ''
    } catch {
      // creation failed; form stays filled for retry
    } finally {
      creating = false
    }
  }

  async function deletePreset(id: number) {
    if (!confirm('Delete this preset deck? This cannot be undone.')) return
    try {
      await apiDelete(`/api/v1/admin/decks/${id}`)
      presets = presets.filter((p) => p.id !== id)
    } catch {
      // leave list unchanged
    }
  }

  loadPresets()
  loadUserDecks()
</script>

<div class="admin-decks">
  <section>
    <h1>Preset Decks</h1>

    <form
      class="create-form"
      onsubmit={(e) => {
        e.preventDefault()
        createPreset()
      }}
    >
      <div class="create-fields">
        <input type="text" bind:value={newName} placeholder="Deck name" disabled={creating} />
        <input
          type="text"
          bind:value={newDescription}
          placeholder="Description (optional)"
          disabled={creating}
        />
      </div>
      <button type="submit" class="btn-primary" disabled={creating || !newName.trim()}
        >Create</button
      >
    </form>

    {#if loading}
      <p class="status">Loading...</p>
    {:else if presets.length === 0}
      <p class="status">No preset decks yet.</p>
    {:else}
      <ul class="preset-list list-reset">
        {#each presets as preset (preset.id)}
          <li class="preset-row card">
            <div class="preset-info">
              <a href="/admin/decks/{preset.id}" class="preset-name">{preset.name}</a>
              {#if preset.description}
                <span class="preset-description">{preset.description}</span>
              {/if}
              <span class="species-count">{preset.species_count} species</span>
            </div>
            <div class="preset-actions">
              <a href="/decks/{preset.id}" class="btn-manage">Manage Species</a>
              <button class="btn-delete btn-danger-ghost" onclick={() => deletePreset(preset.id)}
                >Delete</button
              >
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
      <p class="status">No user decks yet.</p>
    {:else}
      <ul class="user-deck-list list-reset">
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
    padding: 0.5rem 1rem;
    font-size: 0.875rem;
  }
  .preset-list {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }
  .preset-row {
    padding: 0.875rem 1rem;
    display: flex;
    align-items: center;
    gap: 0.75rem;
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
  .preset-name:hover {
    color: var(--accent);
  }
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
    padding: 0.25rem 0.5rem;
    font-size: 0.75rem;
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
  .user-deck-list a:hover {
    color: var(--accent);
  }
  .muted {
    font-size: 0.8125rem;
  }
</style>
