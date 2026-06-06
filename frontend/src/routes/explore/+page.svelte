<script lang="ts">
  import { SvelteSet } from 'svelte/reactivity'
  import { createQuery, useQueryClient } from '@tanstack/svelte-query'
  import { apiGet, apiPost } from '$lib/api'
  import { queryKeys } from '$lib/queries'
  import type { SpeciesListItem } from '../../types'
  import SpeciesRow from '$components/SpeciesRow.svelte'
  import PaginationBar from '$components/PaginationBar.svelte'
  import BulkAddBar from '$components/BulkAddBar.svelte'
  import DeckPickerPopover from '$components/DeckPickerPopover.svelte'

  const defaultLimit = 20

  const queryClient = useQueryClient()

  let q = $state('')
  let offset = $state(0)
  const selectedCodes = new SvelteSet<string>()
  let showDeckPicker = $state(false)

  const allSpeciesQuery = createQuery(() => ({
    queryKey: queryKeys.speciesAll,
    queryFn: (): Promise<SpeciesListItem[]> => apiGet('/api/v1/species/all'),
    staleTime: Infinity,
  }))

  const allSpecies = $derived(allSpeciesQuery.data ?? [])

  const pinnedSpecies = $derived(allSpecies.filter((s) => selectedCodes.has(s.ebird_code)))

  const filtered = $derived(
    q.length < 2
      ? allSpecies
      : allSpecies.filter((s) => {
          const lq = q.toLowerCase()
          return (
            s.common_name.toLowerCase().includes(lq) || s.scientific_name.toLowerCase().includes(lq)
          )
        }),
  )

  const filteredNonPinned = $derived(filtered.filter((s) => !selectedCodes.has(s.ebird_code)))

  const displayed = $derived(filteredNonPinned.slice(offset, offset + defaultLimit))

  function onPageChange(newOffset: number) {
    offset = newOffset
  }

  function toggleSelected(code: string) {
    if (selectedCodes.has(code)) selectedCodes.delete(code)
    else selectedCodes.add(code)
  }

  function clearSelection() {
    selectedCodes.clear()
  }

  async function bulkAddToDeck(deckId: number) {
    showDeckPicker = false
    const codes = [...selectedCodes]
    await apiPost(`/api/v1/decks/${deckId}/species/bulk`, { species_codes: codes }).catch(() => {})
    queryClient.invalidateQueries({ queryKey: queryKeys.decks })
    clearSelection()
  }
</script>

<div class="explore">
  <input
    type="text"
    placeholder="Search species…"
    bind:value={q}
    oninput={() => {
      offset = 0
    }}
  />

  {#if allSpeciesQuery.isPending}
    <p class="status">Loading…</p>
  {:else if allSpeciesQuery.isError}
    <p class="status error">Couldn't load species.</p>
  {:else}
    {#if q.length >= 2 && filteredNonPinned.length > 0}
      <div class="list-header">
        <button
          class="btn-select-all"
          onclick={() => {
            for (const sp of filteredNonPinned) selectedCodes.add(sp.ebird_code)
          }}>Select all ({filteredNonPinned.length})</button
        >
      </div>
    {/if}
    <ul class="species-list">
      {#each pinnedSpecies as s (s.ebird_code)}
        <li class="species-item pinned">
          <input
            type="checkbox"
            checked
            onchange={() => toggleSelected(s.ebird_code)}
            aria-label="Select {s.common_name}"
          />
          <SpeciesRow
            ebird_code={s.ebird_code}
            common_name={s.common_name}
            scientific_name={s.scientific_name}
            image_url={s.image_url}
          />
        </li>
      {/each}
      {#each displayed as s (s.ebird_code)}
        <li class="species-item">
          <input
            type="checkbox"
            checked={selectedCodes.has(s.ebird_code)}
            onchange={() => toggleSelected(s.ebird_code)}
            aria-label="Select {s.common_name}"
          />
          <SpeciesRow
            ebird_code={s.ebird_code}
            common_name={s.common_name}
            scientific_name={s.scientific_name}
            image_url={s.image_url}
          />
        </li>
      {/each}
    </ul>

    <PaginationBar total={filteredNonPinned.length} {offset} limit={defaultLimit} {onPageChange} />
  {/if}

  <BulkAddBar
    selectedCount={selectedCodes.size}
    onAdd={() => {
      showDeckPicker = true
    }}
    onClear={clearSelection}
  />

  {#if showDeckPicker}
    <DeckPickerPopover
      onPick={bulkAddToDeck}
      onClose={() => {
        showDeckPicker = false
      }}
    />
  {/if}
</div>

<style>
  .explore {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  input[type='text'] {
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

  input[type='text']:focus {
    outline: none;
    border-color: var(--accent);
  }

  .species-list {
    list-style: none;
    padding: 0;
    margin: 0;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .species-item {
    display: flex;
    align-items: center;
    gap: 0.625rem;
  }

  .species-item :global(.species-row-wrapper) {
    flex: 1;
    min-width: 0;
  }

  .species-item input[type='checkbox'] {
    flex-shrink: 0;
    accent-color: var(--accent);
    width: 1rem;
    height: 1rem;
    cursor: pointer;
  }

  .species-item.pinned :global(.species-row) {
    border-color: color-mix(in srgb, var(--accent) 50%, var(--border));
  }

  .list-header {
    display: flex;
    justify-content: flex-end;
  }
  .btn-select-all {
    background: transparent;
    border: none;
    color: var(--accent);
    font-size: 0.8125rem;
    font-weight: 600;
    cursor: pointer;
    font-family: inherit;
    padding: 0.25rem 0;
  }
  .status.error {
    color: var(--error);
  }
</style>
