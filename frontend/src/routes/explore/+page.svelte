<script lang="ts">
  import { createQuery } from '@tanstack/svelte-query'
  import SpeciesRow from '$components/SpeciesRow.svelte'
  import PaginationBar from '$components/PaginationBar.svelte'

  const defaultLimit = 20

  let q = $state('')
  let offset = $state(0)
  let debounceTimer: ReturnType<typeof setTimeout> | null = null

  function onSearchInput() {
    if (debounceTimer) clearTimeout(debounceTimer)
    debounceTimer = setTimeout(() => {
      offset = 0
    }, 300)
  }

  const speciesQuery = createQuery(() => ({
    queryKey: ['species', { q, limit: defaultLimit, offset }],
    queryFn: () => {
      const params = new URLSearchParams({ limit: String(defaultLimit), offset: String(offset) })
      if (q) params.set('q', q)
      return fetch(`/api/v1/species?${params}`).then((r) => r.json())
    },
  }))

  function onPageChange(newOffset: number) {
    offset = newOffset
  }
</script>

<div class="explore">
  <input
    type="text"
    placeholder="Search species…"
    bind:value={q}
    oninput={onSearchInput}
  />

  {#if speciesQuery.isPending}
    <p class="status">Loading…</p>
  {:else if speciesQuery.isError}
    <p class="status error">Couldn't load species.</p>
  {:else}
    <ul class="species-list">
      {#each (speciesQuery.data?.results ?? []) as s (s.ebird_code)}
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

    {#if !q}
      <PaginationBar
        total={speciesQuery.data?.count ?? 0}
        {offset}
        limit={defaultLimit}
        {onPageChange}
      />
    {/if}
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
