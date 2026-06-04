<script lang="ts">
  import { goto } from '$app/navigation'
  import type { Deck, DecksResponse, PresetDeck } from '../types'
  import DashboardStats from '$components/DashboardStats.svelte'
  import DeckList from '$components/DeckList.svelte'
  import InstallPrompt from '$components/InstallPrompt.svelte'
  import PresetDeckList from '$components/PresetDeckList.svelte'

  let decks: Deck[] = $state([])
  let nextDueAt: string | null = $state(null)
  let loading = $state(true)
  let presetDecks: PresetDeck[] = $state([])
  let presetsLoading = $state(true)
  let cloning: Set<number> = $state(new Set())

  $effect(() => {
    fetch('/api/v1/decks')
      .then(async (res) => {
        if (res.ok) {
          const data: DecksResponse = await res.json()
          decks = data.decks
          nextDueAt = data.next_due_at
        }
      })
      .finally(() => { loading = false })
  })

  $effect(() => {
    fetch('/api/v1/decks/presets')
      .then(async (res) => { if (res.ok) presetDecks = await res.json() })
      .finally(() => { presetsLoading = false })
  })

  async function cloneDeck(id: number) {
    cloning = new Set([...cloning, id])
    try {
      const res = await fetch(`/api/v1/decks/${id}/clone`, { method: 'POST' })
      if (res.ok) {
        const created = await res.json()
        goto(`/decks/${created.id}`)
      }
    } finally {
      cloning = new Set([...cloning].filter((c) => c !== id))
    }
  }

  const audioDue = $derived(decks.reduce((sum, d) => sum + d.audio_due, 0))
  const imageDue = $derived(decks.reduce((sum, d) => sum + d.image_due, 0))

  function startPractice(deck: Deck, lane: 'audio' | 'image') {
    goto(`/decks/${deck.id}/quiz?lane=${lane}`)
  }
</script>

<div class="dashboard">
  <InstallPrompt />
  {#if loading}
    <p class="status">Loading...</p>
  {:else if decks.length === 0}
    <p class="empty">No decks yet. <a href="/decks">Create one</a> or clone a Starter Deck below to get started.</p>
    {#if !presetsLoading && presetDecks.length > 0}
      <h2 class="section-heading">Starter Decks</h2>
      <p class="section-subheading">Clone one and start practicing right away</p>
      <PresetDeckList {presetDecks} {cloning} onClone={cloneDeck} />
    {/if}
  {:else}
    <DashboardStats {audioDue} {imageDue} {nextDueAt} />
    <DeckList {decks} onPractice={startPractice} />
  {/if}
</div>

<style>
  .dashboard {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }
  .status, .empty {
    text-align: center;
    color: var(--text-muted);
    padding: 2rem 0;
  }
  .empty a {
    color: var(--accent);
  }
  .section-heading {
    font-size: 1rem;
    font-weight: 700;
    color: var(--text);
    margin: 0;
  }
  .section-subheading {
    font-size: 0.8125rem;
    color: var(--text-muted);
    margin: -0.5rem 0 0;
  }
</style>
