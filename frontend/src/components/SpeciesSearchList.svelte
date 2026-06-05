<script lang="ts">
  import type { SpeciesListItem } from '../types'

  interface Props {
    results: SpeciesListItem[]
    pinned: SpeciesListItem[]
    addedCodes: Set<string>
    selectedCodes: Set<string>
    onAdd: (code: string) => void
    onToggle: (code: string) => void
    onSelectAll?: (codes: string[]) => void
  }
  let { results, pinned, addedCodes, selectedCodes, onAdd, onToggle, onSelectAll }: Props = $props()

  const allRows = $derived([
    ...pinned.map((s) => ({ ...s, isPinned: true })),
    ...results.map((s) => ({ ...s, isPinned: false })),
  ])

  const selectableCodes = $derived(
    results.filter((s) => !addedCodes.has(s.ebird_code)).map((s) => s.ebird_code),
  )
</script>

{#if allRows.length > 0}
  {#if onSelectAll && selectableCodes.length > 0}
    <div class="list-header">
      <button class="btn-select-all" onclick={() => onSelectAll(selectableCodes)}>
        Select all ({selectableCodes.length})
      </button>
    </div>
  {/if}
  <ul class="search-list list-reset">
    {#each allRows as s (s.ebird_code)}
      <li class="search-row card" class:pinned={s.isPinned}>
        {#if !addedCodes.has(s.ebird_code)}
          <input
            type="checkbox"
            checked={s.isPinned || selectedCodes.has(s.ebird_code)}
            onchange={() => onToggle(s.ebird_code)}
            aria-label="Select {s.common_name}"
          />
        {/if}
        <span class="names">
          <strong>{s.common_name}</strong>
          <em>{s.scientific_name}</em>
        </span>
        {#if addedCodes.has(s.ebird_code)}
          <span class="added-indicator">Added</span>
        {:else}
          <button class="btn-add" onclick={() => onAdd(s.ebird_code)}>Add</button>
        {/if}
      </li>
    {/each}
  </ul>
{/if}

<style>
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
  .search-list {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .search-row {
    padding: 0.75rem 1rem;
    display: flex;
    align-items: center;
    gap: 0.625rem;
  }

  .search-row.pinned {
    border-color: color-mix(in srgb, var(--accent) 50%, var(--border));
  }

  .search-row input[type='checkbox'] {
    flex-shrink: 0;
    accent-color: var(--accent);
    width: 1rem;
    height: 1rem;
    cursor: pointer;
  }

  .names {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 0.125rem;
  }

  .names strong {
    font-size: 0.9375rem;
    color: var(--text);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .names em {
    font-size: 0.8125rem;
    color: var(--text-muted);
    font-style: italic;
  }

  .btn-add {
    flex-shrink: 0;
    margin-left: auto;
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
    flex-shrink: 0;
    margin-left: auto;
    font-size: 0.75rem;
    color: var(--text-muted);
    padding: 0.25rem 0.5rem;
  }
</style>
