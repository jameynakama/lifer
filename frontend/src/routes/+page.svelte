<script lang="ts">
  import { goto } from '$app/navigation'
  import type { Deck, DecksResponse } from '../types'
  import DashboardStats from '$components/DashboardStats.svelte'
  import DeckList from '$components/DeckList.svelte'
  import InstallPrompt from '$components/InstallPrompt.svelte'

  let decks: Deck[] = $state([])
  let nextDueAt: string | null = $state(null)
  let loading = $state(true)

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
    <p class="empty">No decks yet. <a href="/decks">Create one</a> to get started.</p>
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
</style>
