<script lang="ts">
  import { page } from '$app/state'

  interface DeckInfo {
    id: number
    name: string
    owner_name: string
    owner_email: string
  }

  interface SpeciesItem {
    ebird_code: string
    common_name: string
    scientific_name: string
  }

  let deck: DeckInfo | null = $state(null)
  let species: SpeciesItem[] = $state([])
  let loading = $state(true)
  let error = $state('')

  const deckID = $derived(page.params.id)

  $effect(() => {
    loading = true
    error = ''
    fetch(`/api/v1/admin/decks/${deckID}/species`)
      .then((r) => {
        if (!r.ok) throw new Error(`Failed to load: ${r.status}`)
        return r.json()
      })
      .then((data) => {
        deck = data.deck
        species = data.species ?? []
      })
      .catch((e: Error) => {
        error = e.message
      })
      .finally(() => {
        loading = false
      })
  })
</script>

<a href="/admin/decks" class="back">← Back to decks</a>

{#if loading}
  <p>Loading...</p>
{:else if error}
  <p class="error">{error}</p>
{:else if deck}
  <div class="deck-header">
    <h2>{deck.name}</h2>
    {#if deck.owner_email}
      <span class="owner">{deck.owner_name} &lt;{deck.owner_email}&gt;</span>
    {:else}
      <span class="badge">Preset</span>
    {/if}
  </div>

  <p class="count">{species.length} species</p>

  {#if species.length === 0}
    <p class="empty">No species in this deck.</p>
  {:else}
    <ul>
      {#each species as sp (sp.ebird_code)}
        <li>
          <span class="name">{sp.common_name}</span>
          <span class="muted">{sp.ebird_code}</span>
        </li>
      {/each}
    </ul>
  {/if}
{/if}

<style>
  .back {
    display: inline-block;
    color: var(--text-muted);
    text-decoration: none;
    font-size: 0.875rem;
    margin-bottom: 1rem;
  }
  .back:hover { color: var(--text); }
  .deck-header {
    display: flex;
    align-items: baseline;
    gap: 0.75rem;
    margin-bottom: 0.25rem;
  }
  h2 {
    font-size: 1.125rem;
    font-weight: 700;
    color: var(--text);
    margin: 0;
  }
  .owner {
    color: var(--text-muted);
    font-size: 0.875rem;
  }
  .badge {
    font-size: 0.75rem;
    font-weight: 600;
    color: var(--accent);
    border: 1px solid var(--accent);
    border-radius: 4px;
    padding: 0.1rem 0.4rem;
  }
  .count {
    color: var(--text-muted);
    font-size: 0.875rem;
    margin: 0.25rem 0 0.75rem;
  }
  ul {
    list-style: none;
    padding: 0;
    margin: 0;
  }
  li {
    display: flex;
    align-items: baseline;
    gap: 0.5rem;
    padding: 0.35rem 0;
    border-bottom: 1px solid var(--border);
  }
  .name {
    color: var(--text);
    font-size: 0.9375rem;
  }
  .muted {
    color: var(--text-muted);
    font-size: 0.8125rem;
  }
  .empty {
    color: var(--text-muted);
  }
  .error {
    color: var(--danger);
  }
</style>
