<script lang="ts">
  import { goto } from '$app/navigation'
  import { page } from '$app/state'

  interface Species {
    ebird_code: string
    common_name: string
    scientific_name: string
  }

  let groupId = $derived(page.params.id)
  let groupSpecies: Species[] = $state([])
  let searchQuery = $state('')
  let searchResults: Species[] = $state([])
  let loading = $state(true)
  let searchTimer: ReturnType<typeof setTimeout> | null = null

  async function loadSpecies() {
    try {
      const res = await fetch(`/api/v1/groups/${groupId}/species`)
      if (res.ok) groupSpecies = await res.json()
    } catch {
      // network error, leave groupSpecies empty
    } finally {
      loading = false
    }
  }

  function onSearchInput() {
    if (searchTimer) clearTimeout(searchTimer)
    if (!searchQuery.trim()) { searchResults = []; return }
    searchTimer = setTimeout(async () => {
      const res = await fetch(`/api/v1/species?q=${encodeURIComponent(searchQuery)}`)
      if (res.ok) searchResults = await res.json()
    }, 300)
  }

  async function addSpecies(ebirdCode: string) {
    const res = await fetch(`/api/v1/groups/${groupId}/species`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ ebird_code: ebirdCode }),
    })
    if (res.ok) {
      const added = searchResults.find((s) => s.ebird_code === ebirdCode)
      if (added) groupSpecies = [...groupSpecies, added]
      searchQuery = ''
      searchResults = []
    }
  }

  async function removeSpecies(ebirdCode: string) {
    const res = await fetch(`/api/v1/groups/${groupId}/species/${ebirdCode}`, {
      method: 'DELETE',
    })
    if (res.ok) {
      groupSpecies = groupSpecies.filter((s) => s.ebird_code !== ebirdCode)
    }
  }

  $effect(() => {
    if (groupId) loadSpecies()
  })

  $effect(() => {
    return () => {
      if (searchTimer) clearTimeout(searchTimer)
    }
  })
</script>

<div class="group-detail">
  <div class="actions">
    <button class="btn-practice" onclick={() => goto(`/groups/${groupId}/quiz?lane=audio`)}>
      Practice Audio
    </button>
    <button class="btn-practice" onclick={() => goto(`/groups/${groupId}/quiz?lane=image`)}>
      Practice Image
    </button>
  </div>

  {#if loading}
    <p class="status">Loading...</p>
  {:else if groupSpecies.length === 0}
    <p class="empty">No species yet. Search below to add some.</p>
  {:else}
    <ul class="species-list">
      {#each groupSpecies as s (s.ebird_code)}
        <li class="species-row">
          <span>
            <strong>{s.common_name}</strong>
            <em>{s.scientific_name}</em>
          </span>
          <button class="btn-remove" onclick={() => removeSpecies(s.ebird_code)}>Remove</button>
        </li>
      {/each}
    </ul>
  {/if}

  <div class="search-section">
    <input
      type="text"
      placeholder="Search species to add..."
      bind:value={searchQuery}
      oninput={onSearchInput}
    />
    {#if searchResults.length > 0}
      <ul class="search-results">
        {#each searchResults as s (s.ebird_code)}
          <li class="search-row">
            <span>
              <strong>{s.common_name}</strong>
              <em>{s.scientific_name}</em>
            </span>
            <button class="btn-add" onclick={() => addSpecies(s.ebird_code)}>Add</button>
          </li>
        {/each}
      </ul>
    {/if}
  </div>
</div>

<style>
  .group-detail {
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
  }
  .actions {
    display: flex;
    gap: 0.75rem;
  }
  .btn-practice {
    background: var(--accent);
    color: #fff;
    border: none;
    border-radius: 10px;
    padding: 0.625rem 1.25rem;
    font-size: 0.875rem;
    font-weight: 600;
    cursor: pointer;
    font-family: inherit;
  }
  .species-list, .search-results {
    list-style: none;
    padding: 0;
    margin: 0;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }
  .species-row, .search-row {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: 10px;
    padding: 0.75rem 1rem;
    display: flex;
    justify-content: space-between;
    align-items: center;
    box-shadow: var(--shadow);
  }
  .species-row span, .search-row span {
    display: flex;
    flex-direction: column;
    gap: 0.125rem;
  }
  .species-row strong, .search-row strong {
    font-size: 0.9375rem;
    color: var(--text);
  }
  .species-row em, .search-row em {
    font-size: 0.8125rem;
    color: var(--text-muted);
    font-style: italic;
  }
  .btn-remove {
    background: transparent;
    border: 1px solid var(--border);
    color: var(--text-muted);
    border-radius: 6px;
    padding: 0.25rem 0.5rem;
    font-size: 0.75rem;
    cursor: pointer;
    font-family: inherit;
  }
  .btn-add {
    background: var(--surface);
    border: 1px solid var(--accent);
    color: var(--accent);
    border-radius: 6px;
    padding: 0.25rem 0.625rem;
    font-size: 0.75rem;
    font-weight: 600;
    cursor: pointer;
    font-family: inherit;
  }
  .search-section {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }
  .search-section input {
    background: var(--surface);
    border: 1px solid var(--border);
    color: var(--text);
    border-radius: 8px;
    padding: 0.5rem 0.75rem;
    font-size: 0.9375rem;
    font-family: inherit;
    width: 100%;
    box-sizing: border-box;
  }
  .status, .empty {
    text-align: center;
    color: var(--text-muted);
    padding: 2rem 0;
  }
</style>
