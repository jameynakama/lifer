<script lang="ts">
  import { goto } from '$app/navigation'
  import { page } from '$app/state'
  import { createQuery, useQueryClient } from '@tanstack/svelte-query'
  import { apiDelete, apiGet, apiPatch, apiPost, apiPut } from '$lib/api'
  import { queryKeys } from '$lib/queries'
  import type { Deck, DeckSpecies, SpeciesListItem } from '../../../types'
  import BulkAddBar from '$components/BulkAddBar.svelte'
  import SpeciesSearchList from '$components/SpeciesSearchList.svelte'

  let deckId = $derived(page.params.id)
  let deckName = $state('')
  let deckDescription = $state('')
  let editingName = $state(false)
  let editingDescription = $state(false)
  let nameInput = $state('')
  let descriptionInput = $state('')
  let deckSpecies: DeckSpecies[] = $state([])
  let audioDue = $state(0)
  let imageDue = $state(0)
  let searchQuery = $state('')
  let loading = $state(true)

  let addedCodes = $derived(new Set(deckSpecies.map((s) => s.ebird_code)))
  let togglingCodes: Set<string> = $state(new Set())
  let selectedSearchCodes: Set<string> = $state(new Set())

  const allSpeciesQuery = createQuery(() => ({
    queryKey: queryKeys.speciesAll,
    queryFn: (): Promise<SpeciesListItem[]> => apiGet('/api/v1/species/all'),
    staleTime: Infinity,
  }))

  const queryClient = useQueryClient()
  const invalidateDecks = () => queryClient.invalidateQueries({ queryKey: queryKeys.decks })

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

  const pinnedSearchSpecies = $derived(
    (allSpeciesQuery.data ?? []).filter((s) => selectedSearchCodes.has(s.ebird_code))
  )

  const unpinnedSearchResults = $derived(
    searchResults.filter((s) => !selectedSearchCodes.has(s.ebird_code))
  )

  async function loadDeck() {
    try {
      const [d, species] = await Promise.all([
        apiGet<Deck>(`/api/v1/decks/${deckId}`),
        apiGet<DeckSpecies[]>(`/api/v1/decks/${deckId}/species`),
      ])
      deckName = d.name ?? ''
      deckDescription = d.description ?? ''
      audioDue = d.audio_due ?? 0
      imageDue = d.image_due ?? 0
      deckSpecies = species
    } catch {
      // network error
    } finally {
      loading = false
    }
  }

  function startEditing() {
    nameInput = deckName
    editingName = true
  }

  async function saveName() {
    const trimmed = nameInput.trim()
    if (!trimmed || trimmed === deckName) { editingName = false; return }
    try {
      await apiPatch(`/api/v1/decks/${deckId}`, { name: trimmed, description: deckDescription })
      deckName = trimmed
      invalidateDecks()
    } catch {
      // keep the old name
    }
    editingName = false
  }

  function handleNameKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') saveName()
    else if (e.key === 'Escape') editingName = false
  }

  function startEditingDescription() {
    descriptionInput = deckDescription
    editingDescription = true
  }

  async function saveDescription() {
    const trimmed = descriptionInput.trim()
    if (trimmed === deckDescription) { editingDescription = false; return }
    try {
      await apiPatch(`/api/v1/decks/${deckId}`, { name: deckName, description: trimmed })
      deckDescription = trimmed
    } catch {
      // keep the old description
    }
    editingDescription = false
  }

  function handleDescriptionKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') saveDescription()
    else if (e.key === 'Escape') editingDescription = false
  }

  function toggleSearchSelected(code: string) {
    const next = new Set(selectedSearchCodes)
    if (next.has(code)) next.delete(code)
    else next.add(code)
    selectedSearchCodes = next
  }

  async function bulkAddToThisDeck() {
    const codes = [...selectedSearchCodes]
    try {
      await apiPost(`/api/v1/decks/${deckId}/species/bulk`, { species_codes: codes })
    } catch {
      return
    }
    const newSpecies = codes
      .filter((c) => !addedCodes.has(c))
      .map((c) => {
        const s = (allSpeciesQuery.data ?? []).find((sp) => sp.ebird_code === c)
        return s
          ? { ebird_code: s.ebird_code, common_name: s.common_name, scientific_name: s.scientific_name, audio_enabled: true, image_enabled: true }
          : null
      })
      .filter((s): s is DeckSpecies => s !== null)
    deckSpecies = [...deckSpecies, ...newSpecies]
    selectedSearchCodes = new Set()
    invalidateDecks()
  }

  async function addSpecies(ebirdCode: string) {
    try {
      await apiPost(`/api/v1/decks/${deckId}/species`, { ebird_code: ebirdCode })
    } catch {
      return
    }
    {
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

  async function deleteDeck() {
    if (!confirm(`Delete "${deckName}"? This cannot be undone.`)) return
    try {
      await apiDelete(`/api/v1/decks/${deckId}`)
    } catch {
      return
    }
    invalidateDecks()
    goto('/decks')
  }

  async function removeSpecies(ebirdCode: string) {
    try {
      await apiDelete(`/api/v1/decks/${deckId}/species/${ebirdCode}`)
    } catch {
      return
    }
    deckSpecies = deckSpecies.filter((s) => s.ebird_code !== ebirdCode)
    invalidateDecks()
  }

  async function toggleLane(species: DeckSpecies, lane: 'audio' | 'image') {
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
      const updated = await apiPut<{ audio_enabled: boolean; image_enabled: boolean }>(
        `/api/v1/species/${species.ebird_code}/preferences`,
        next
      )
      deckSpecies = deckSpecies.map((s) =>
        s.ebird_code === species.ebird_code
          ? { ...s, audio_enabled: updated.audio_enabled, image_enabled: updated.image_enabled }
          : s
      )
    } catch {
      deckSpecies = deckSpecies.map((s) =>
        s.ebird_code === species.ebird_code ? { ...s, ...prev } : s
      )
    } finally {
      togglingCodes = new Set([...togglingCodes].filter((c) => c !== species.ebird_code))
    }
  }

  $effect(() => {
    if (deckId) loadDeck()
  })
</script>

<div class="deck-detail">
  <div class="deck-header">
    {#if editingName}
      <!-- svelte-ignore a11y_autofocus -->
      <input
        class="name-input"
        bind:value={nameInput}
        onkeydown={handleNameKeydown}
        onblur={saveName}
        autofocus
      />
    {:else}
      <h1 class="deck-name">{deckName}</h1>
      <button class="btn-rename" onclick={startEditing} aria-label="Rename deck">✎</button>
    {/if}
  </div>

  <div class="description-row">
    {#if editingDescription}
      <!-- svelte-ignore a11y_autofocus -->
      <input
        class="description-input"
        bind:value={descriptionInput}
        onkeydown={handleDescriptionKeydown}
        onblur={saveDescription}
        placeholder="Add a description..."
        autofocus
      />
    {:else}
      <span class="deck-description" class:placeholder={!deckDescription}>{deckDescription || 'Add a description...'}</span>
      <button class="btn-rename" onclick={startEditingDescription} aria-label="Edit description">✎</button>
    {/if}
  </div>

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
    <button class="btn-delete-deck" onclick={deleteDeck}>Delete deck</button>
  </div>

  {#if loading}
    <p class="status">Loading...</p>
  {:else if deckSpecies.length === 0}
    <p class="empty">No species yet. Search below to add some.</p>
  {:else}
    <p class="lane-legend">♪ audio &nbsp; ◉ image -- click to include or exclude that media type from your study queue</p>
    <ul class="species-list">
      {#each deckSpecies as s (s.ebird_code)}
        <li class="species-row">
          <a href="/explore/{s.ebird_code}" class="species-names">
            <strong>{s.common_name}</strong>
            <em>{s.scientific_name}</em>
          </a>
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
    <SpeciesSearchList
      results={unpinnedSearchResults}
      pinned={pinnedSearchSpecies}
      {addedCodes}
      selectedCodes={selectedSearchCodes}
      onAdd={addSpecies}
      onToggle={toggleSearchSelected}
      onSelectAll={(codes) => { selectedSearchCodes = new Set([...selectedSearchCodes, ...codes]) }}
    />
  </div>

  <BulkAddBar
    selectedCount={selectedSearchCodes.size}
    onAdd={bulkAddToThisDeck}
    onClear={() => { selectedSearchCodes = new Set() }}
  />
</div>

<style>
  .deck-detail {
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
  }
  .deck-header {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }
  .deck-name {
    font-size: 1.25rem;
    font-weight: 700;
    color: var(--text);
    margin: 0;
  }
  .btn-rename {
    background: transparent;
    border: none;
    color: var(--text-muted);
    font-size: 1rem;
    cursor: pointer;
    padding: 0.125rem 0.25rem;
    line-height: 1;
  }
  .btn-rename:hover {
    color: var(--text);
  }
  .name-input {
    font-size: 1.25rem;
    font-weight: 700;
    color: var(--text);
    background: var(--surface);
    border: 1px solid var(--accent);
    border-radius: 6px;
    padding: 0.125rem 0.375rem;
    font-family: inherit;
    flex: 1;
  }
  .description-row {
    display: flex;
    align-items: center;
    gap: 0.25rem;
    margin-top: -0.5rem;
  }
  .deck-description {
    font-size: 0.875rem;
    color: var(--text-muted);
  }
  .deck-description.placeholder {
    font-style: italic;
    opacity: 0.6;
  }
  .description-input {
    flex: 1;
    font-size: 0.875rem;
    color: var(--text);
    background: var(--surface);
    border: 1px solid var(--accent);
    border-radius: 6px;
    padding: 0.125rem 0.375rem;
    font-family: inherit;
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
  .btn-delete-deck {
    background: transparent;
    border: 1px solid var(--border);
    color: var(--text-muted);
    border-radius: 10px;
    padding: 0.625rem 1.25rem;
    font-size: 0.875rem;
    font-weight: 600;
    cursor: pointer;
    font-family: inherit;
    margin-left: auto;
  }
  .btn-delete-deck:hover {
    border-color: #ef4444;
    color: #ef4444;
  }
  .species-list {
    list-style: none;
    padding: 0;
    margin: 0;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }
  .species-row {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: 10px;
    padding: 0.75rem 1rem;
    display: flex;
    align-items: center;
    gap: 0.625rem;
    box-shadow: var(--shadow);
  }
  .species-names {
    display: flex;
    flex-direction: column;
    gap: 0.125rem;
    text-decoration: none;
  }
  .species-row strong {
    font-size: 0.9375rem;
    color: var(--text);
  }
  .species-row em {
    font-size: 0.8125rem;
    color: var(--text-muted);
    font-style: italic;
  }
  .row-actions {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin-left: auto;
    flex-shrink: 0;
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
  .btn-remove:hover {
    border-color: #ef4444;
    color: #ef4444;
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
  .lane-legend {
    font-size: .9rem;
    color: var(--text-muted);
    margin: 0;
  }
  .status, .empty {
    text-align: center;
    color: var(--text-muted);
    padding: 2rem 0;
  }
</style>
