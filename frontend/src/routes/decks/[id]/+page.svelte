<script lang="ts">
  import { goto } from '$app/navigation'
  import { page } from '$app/state'
  import { createQuery } from '@tanstack/svelte-query'
  import type { SpeciesListItem } from '../../../types'

  interface Species {
    ebird_code: string
    common_name: string
    scientific_name: string
    audio_enabled: boolean
    image_enabled: boolean
  }

  let deckId = $derived(page.params.id)
  let deckSpecies: Species[] = $state([])
  let audioDue = $state(0)
  let imageDue = $state(0)
  let searchQuery = $state('')
  let loading = $state(true)

  let addedCodes = $derived(new Set(deckSpecies.map((s) => s.ebird_code)))
  let togglingCodes: Set<string> = $state(new Set())

  const allSpeciesQuery = createQuery(() => ({
    queryKey: ['species', 'all'],
    queryFn: (): Promise<SpeciesListItem[]> =>
      fetch('/api/v1/species/all').then((r) => r.json()),
    staleTime: Infinity,
  }))

  const searchResults = $derived(
    searchQuery.length < 2
      ? []
      : (allSpeciesQuery.data ?? []).filter((s) => {
          const lq = searchQuery.toLowerCase()
          return (
            s.common_name.toLowerCase().includes(lq) ||
            s.scientific_name.toLowerCase().includes(lq)
          )
        })
  )

  async function loadDeck() {
    try {
      const [deckRes, speciesRes] = await Promise.all([
        fetch(`/api/v1/decks/${deckId}`),
        fetch(`/api/v1/decks/${deckId}/species`),
      ])
      if (deckRes.ok) {
        const d = await deckRes.json()
        audioDue = d.audio_due ?? 0
        imageDue = d.image_due ?? 0
      }
      if (speciesRes.ok) deckSpecies = await speciesRes.json()
    } catch {
      // network error
    } finally {
      loading = false
    }
  }

  async function addSpecies(ebirdCode: string) {
    const res = await fetch(`/api/v1/decks/${deckId}/species`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ ebird_code: ebirdCode }),
    })
    if (res.ok) {
      const added = searchResults.find((s) => s.ebird_code === ebirdCode)
      if (added) deckSpecies = [...deckSpecies, {
        ebird_code: added.ebird_code,
        common_name: added.common_name,
        scientific_name: added.scientific_name,
        audio_enabled: true,
        image_enabled: true,
      }]
    }
  }

  async function removeSpecies(ebirdCode: string) {
    const res = await fetch(`/api/v1/decks/${deckId}/species/${ebirdCode}`, {
      method: 'DELETE',
    })
    if (res.ok) {
      deckSpecies = deckSpecies.filter((s) => s.ebird_code !== ebirdCode)
    }
  }

  async function toggleLane(species: Species, lane: 'audio' | 'image') {
    if (togglingCodes.has(species.ebird_code)) return
    togglingCodes = new Set([...togglingCodes, species.ebird_code])

    const prev = { audio_enabled: species.audio_enabled, image_enabled: species.image_enabled }
    const next =
      lane === 'audio'
        ? { ...prev, audio_enabled: !prev.audio_enabled }
        : { ...prev, image_enabled: !prev.image_enabled }

    deckSpecies = deckSpecies.map((s) =>
      s.ebird_code === species.ebird_code ? { ...s, ...next } : s
    )

    try {
      const res = await fetch(`/api/v1/species/${species.ebird_code}/preferences`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(next),
      })

      if (res.ok) {
        const updated = await res.json()
        deckSpecies = deckSpecies.map((s) =>
          s.ebird_code === species.ebird_code
            ? { ...s, audio_enabled: updated.audio_enabled, image_enabled: updated.image_enabled }
            : s
        )
      } else {
        deckSpecies = deckSpecies.map((s) =>
          s.ebird_code === species.ebird_code ? { ...s, ...prev } : s
        )
      }
    } finally {
      togglingCodes = new Set([...togglingCodes].filter((c) => c !== species.ebird_code))
    }
  }

  $effect(() => {
    if (deckId) loadDeck()
  })
</script>

<div class="deck-detail">
  <div class="actions">
    <button
      class="btn-study"
      onclick={() => goto(`/decks/${deckId}/quiz?lane=audio`)}
      disabled={audioDue === 0}
    >
      Study Audio{#if audioDue > 0}<span class="due-badge">{audioDue}</span>{/if}
    </button>
    <button
      class="btn-study"
      onclick={() => goto(`/decks/${deckId}/quiz?lane=image`)}
      disabled={imageDue === 0}
    >
      Study Image{#if imageDue > 0}<span class="due-badge">{imageDue}</span>{/if}
    </button>
  </div>
  <div class="actions">
    <button class="btn-practice-outline" onclick={() => goto(`/decks/${deckId}/practice?lane=audio`)}>
      Practice Audio
    </button>
    <button class="btn-practice-outline" onclick={() => goto(`/decks/${deckId}/practice?lane=image`)}>
      Practice Image
    </button>
  </div>

  {#if loading}
    <p class="status">Loading...</p>
  {:else if deckSpecies.length === 0}
    <p class="empty">No species yet. Search below to add some.</p>
  {:else}
    <ul class="species-list">
      {#each deckSpecies as s (s.ebird_code)}
        <li class="species-row">
          <span class="species-names">
            <strong>{s.common_name}</strong>
            <em>{s.scientific_name}</em>
          </span>
          <div class="row-actions">
            <div class="lane-toggles">
              <button
                class="lane-toggle"
                class:active={s.audio_enabled}
                aria-label="Toggle audio"
                disabled={togglingCodes.has(s.ebird_code)}
                onclick={() => toggleLane(s, 'audio')}
              >♪</button>
              <button
                class="lane-toggle"
                class:active={s.image_enabled}
                aria-label="Toggle image"
                disabled={togglingCodes.has(s.ebird_code)}
                onclick={() => toggleLane(s, 'image')}
              >◉</button>
            </div>
            <button class="btn-remove" onclick={() => removeSpecies(s.ebird_code)}>Remove</button>
          </div>
        </li>
      {/each}
    </ul>
  {/if}

  <div class="search-section">
    <input
      type="text"
      placeholder="Search species to add..."
      bind:value={searchQuery}
    />
    {#if searchResults.length > 0}
      <ul class="search-results">
        {#each searchResults as s (s.ebird_code)}
          <li class="search-row">
            <span>
              <strong>{s.common_name}</strong>
              <em>{s.scientific_name}</em>
            </span>
            {#if addedCodes.has(s.ebird_code)}
              <span class="added-indicator">Added</span>
            {:else}
              <button class="btn-add" onclick={() => addSpecies(s.ebird_code)}>Add</button>
            {/if}
          </li>
        {/each}
      </ul>
    {/if}
  </div>
</div>

<style>
  .deck-detail {
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
  }
  .actions {
    display: flex;
    gap: 0.75rem;
  }
  .btn-study {
    background: var(--accent);
    color: #fff;
    border: none;
    border-radius: 10px;
    padding: 0.625rem 1.25rem;
    font-size: 0.875rem;
    font-weight: 600;
    cursor: pointer;
    font-family: inherit;
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }
  .btn-study:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }
  .due-badge {
    background: rgba(255, 255, 255, 0.25);
    border-radius: 999px;
    padding: 0.1rem 0.45rem;
    font-size: 0.75rem;
    font-weight: 700;
    line-height: 1.4;
  }
  .btn-practice-outline {
    background: transparent;
    border: 1px solid var(--accent);
    color: var(--accent);
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
  .species-names {
    display: flex;
    flex-direction: column;
    gap: 0.125rem;
  }
  .search-row span {
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
  .row-actions {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }
  .lane-toggles {
    display: flex;
    gap: 0.25rem;
  }
  .lane-toggle {
    background: transparent;
    border: 1px solid var(--border);
    color: var(--text-muted);
    border-radius: 6px;
    padding: 0.25rem 0.4rem;
    font-size: 0.875rem;
    cursor: pointer;
    font-family: inherit;
    line-height: 1;
  }
  .lane-toggle.active {
    background: var(--accent);
    border-color: var(--accent);
    color: #fff;
  }
  .lane-toggle:disabled {
    opacity: 0.4;
    cursor: not-allowed;
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
  .added-indicator {
    font-size: 0.75rem;
    color: var(--text-muted);
    padding: 0.25rem 0.5rem;
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
