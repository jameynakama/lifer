<script lang="ts">
  import { goto } from '$app/navigation'
  import { useQueryClient } from '@tanstack/svelte-query'
  import { apiPost } from '$lib/api'
  import { createDecksQuery, createPresetsQuery, queryKeys } from '$lib/queries'
  import type { Deck } from '../types'
  import DashboardStats from '$components/DashboardStats.svelte'
  import DeckList from '$components/DeckList.svelte'
  import InstallPrompt from '$components/InstallPrompt.svelte'
  import PresetDeckList from '$components/PresetDeckList.svelte'

  const queryClient = useQueryClient()
  const decksQuery = createDecksQuery()
  const presetsQuery = createPresetsQuery()

  let cloning: Set<number> = $state(new Set())

  const decks = $derived(decksQuery.data?.decks ?? [])
  const nextDueAt = $derived(decksQuery.data?.next_due_at ?? null)
  const loading = $derived(decksQuery.isPending)
  const presetDecks = $derived(presetsQuery.data ?? [])
  const presetsLoading = $derived(presetsQuery.isPending)

  async function cloneDeck(id: number) {
    cloning = new Set([...cloning, id])
    try {
      const created = await apiPost<{ id: number }>(`/api/v1/decks/${id}/clone`)
      queryClient.invalidateQueries({ queryKey: queryKeys.decks })
      goto(`/decks/${created.id}`)
    } catch {
      // leave the dashboard as-is; the clone button simply re-enables
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
    <p class="status">No decks yet. <a href="/decks">Create one</a> or clone a Starter Deck below to get started.</p>
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
  .status a {
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
