<script lang="ts">
  import { createQuery } from '@tanstack/svelte-query'
  import { goto } from '$app/navigation'
  import { page } from '$app/state'
  import DeckDropdown from '$components/DeckDropdown.svelte'
  import RecordingsList from '$components/RecordingsList.svelte'
  import PhotoGrid from '$components/PhotoGrid.svelte'

  let ebirdCode = $derived(page.params.ebird_code)
  let dropdownOpen = $state(false)

  const detailQuery = createQuery(() => ({
    queryKey: ['species', ebirdCode],
    queryFn: async () => {
      const res = await fetch(`/api/v1/species/${ebirdCode}`)
      if (res.status === 404) throw { status: 404 }
      if (!res.ok) throw new Error('server error')
      return res.json()
    },
  }))

  $effect(() => {
    if (detailQuery.isError) {
      goto('/explore')
    }
  })

  function toggleDropdown(e: MouseEvent) {
    e.stopPropagation()
    dropdownOpen = !dropdownOpen
  }
</script>

<svelte:window onclick={() => { dropdownOpen = false }} />

<div class="detail">
  {#if detailQuery.isPending}
    <p class="status">Loading…</p>
  {:else if detailQuery.isError}
    <p class="status">Couldn't load species.</p>
  {:else if detailQuery.data}
    {@const sp = detailQuery.data}

    <div class="species-header">
      <div>
        <h1 class="common-name">{sp.common_name}</h1>
        <p class="scientific-name">{sp.scientific_name}</p>
      </div>
      <div class="deck-btn-wrapper">
        <button class="deck-btn" onclick={toggleDropdown} aria-label="Add to deck">
          + Deck
        </button>
        {#if dropdownOpen}
          <DeckDropdown ebird_code={sp.ebird_code} onClose={() => { dropdownOpen = false }} />
        {/if}
      </div>
    </div>

    <div class="section">
      <div class="section-label">Recordings</div>
      <RecordingsList recordings={sp.recordings} />
    </div>

    <div class="section">
      <div class="section-label">Photos</div>
      <PhotoGrid images={sp.images} />
    </div>
  {/if}
</div>

<style>
  .detail {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  .species-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    gap: 0.75rem;
  }

  .common-name {
    font-size: 1rem;
    font-weight: 700;
    color: var(--text);
    margin: 0;
  }

  .scientific-name {
    font-size: 0.8125rem;
    font-style: italic;
    color: var(--text-muted);
    margin: 0.125rem 0 0;
  }

  .deck-btn-wrapper {
    position: relative;
    flex-shrink: 0;
  }

  .deck-btn {
    background: var(--surface);
    border: 1px solid var(--border);
    color: var(--text-secondary);
    border-radius: 6px;
    padding: 0.25rem 0.5rem;
    font-size: 0.75rem;
    font-weight: 500;
    cursor: pointer;
    font-family: inherit;
    white-space: nowrap;
    box-shadow: var(--shadow);
  }

  .deck-btn:hover {
    border-color: var(--accent);
    color: var(--accent);
  }

  .section {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: 10px;
    padding: 0.75rem 1rem;
    box-shadow: var(--shadow);
  }

  .section-label {
    font-size: 0.6875rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-secondary);
    margin-bottom: 0.5rem;
  }

  .status {
    text-align: center;
    color: var(--text-muted);
    padding: 2rem 0;
  }
</style>
