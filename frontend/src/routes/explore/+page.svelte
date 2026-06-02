<script lang="ts">
  import { createQuery } from '@tanstack/svelte-query'
  import type { SpeciesListItem } from '../../types'
  import SpeciesRow from '$components/SpeciesRow.svelte'
  import PaginationBar from '$components/PaginationBar.svelte'

  const defaultLimit = 20

  let q = $state('')
  let offset = $state(0)

  const allSpeciesQuery = createQuery(() => ({
    queryKey: ['species', 'all'],
    queryFn: (): Promise<SpeciesListItem[]> =>
      fetch('/api/v1/species/all').then((r) => r.json()),
    staleTime: Infinity,
  }))

  const allSpecies = $derived(allSpeciesQuery.data ?? [])

  const filtered = $derived(
    q.length < 2
      ? allSpecies
      : allSpecies.filter((s) => {
          const lq = q.toLowerCase()
          return (
            s.common_name.toLowerCase().includes(lq) ||
            s.scientific_name.toLowerCase().includes(lq)
          )
        })
  )

  const displayed = $derived(filtered.slice(offset, offset + defaultLimit))

  function onPageChange(newOffset: number) {
    offset = newOffset
  }
</script>

<div class="explore">
  <input
    type="text"
    placeholder="Search species…"
    bind:value={q}
    oninput={() => { offset = 0 }}
  />

  {#if allSpeciesQuery.isPending}
    <p class="status">Loading…</p>
  {:else if allSpeciesQuery.isError}
    <p class="status error">Couldn't load species.</p>
  {:else}
    <ul class="species-list">
      {#each displayed as s (s.ebird_code)}
        <li>
          <SpeciesRow
            ebird_code={s.ebird_code}
            common_name={s.common_name}
            scientific_name={s.scientific_name}
            image_url={s.image_url ?? null}
          />
        </li>
      {/each}
    </ul>

    <PaginationBar
      total={filtered.length}
      {offset}
      limit={defaultLimit}
      {onPageChange}
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

  .status {
    text-align: center;
    color: var(--text-muted);
    padding: 2rem 0;
  }
</style>
