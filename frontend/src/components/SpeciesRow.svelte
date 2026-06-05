<script lang="ts">
  import DeckDropdown from './DeckDropdown.svelte'

  let {
    ebird_code,
    common_name,
    scientific_name,
    image_url,
  }: {
    ebird_code: string
    common_name: string
    scientific_name: string
    image_url: string | null
  } = $props()

  let dropdownOpen = $state(false)

  function toggleDropdown(e: MouseEvent) {
    e.preventDefault()
    e.stopPropagation()
    dropdownOpen = !dropdownOpen
  }

  function closeDropdown() {
    dropdownOpen = false
  }
</script>

<svelte:window onclick={closeDropdown} />

<div class="species-row-wrapper">
  <a href="/explore/{ebird_code}" class="species-row card">
    {#if image_url}
      <img class="thumbnail" src={image_url} alt={common_name} loading="lazy" />
    {:else}
      <div class="thumbnail thumbnail-placeholder"></div>
    {/if}
    <div class="names">
      <span class="common-name">{common_name}</span>
      <span class="scientific-name">{scientific_name}</span>
    </div>
  </a>
  <div class="deck-btn-wrapper">
    <button class="deck-btn" onclick={toggleDropdown} aria-label="Add to deck">
      + Deck
    </button>
    {#if dropdownOpen}
      <DeckDropdown {ebird_code} onClose={closeDropdown} />
    {/if}
  </div>
</div>

<style>
  .species-row-wrapper {
    display: flex;
    align-items: center;
    gap: 0;
  }

  .species-row {
    flex: 1;
    min-width: 0;
    border-right: none;
    border-radius: 10px 0 0 10px;
    padding: 0.625rem 1rem;
    display: flex;
    align-items: center;
    gap: 0.75rem;
    text-decoration: none;
  }

  .species-row:hover .common-name {
    color: var(--accent);
  }

  .thumbnail {
    width: 44px;
    height: 44px;
    border-radius: 8px;
    object-fit: cover;
    flex-shrink: 0;
  }

  .thumbnail-placeholder {
    background: var(--border);
  }

  .names {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 0.125rem;
  }

  .common-name {
    font-size: 0.9375rem;
    font-weight: 600;
    color: var(--text);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .scientific-name {
    font-size: 0.8125rem;
    color: var(--text-muted);
    font-style: italic;
  }

  .deck-btn-wrapper {
    position: relative;
    flex-shrink: 0;
  }

  .deck-btn {
    background: var(--surface);
    border: 1px solid var(--border);
    border-left: none;
    border-radius: 0 10px 10px 0;
    color: var(--text-secondary);
    padding: 0.625rem 0.75rem;
    font-size: 0.75rem;
    font-weight: 500;
    cursor: pointer;
    font-family: inherit;
    white-space: nowrap;
    box-shadow: var(--shadow);
  }

  .deck-btn:hover {
    color: var(--accent);
  }
</style>
