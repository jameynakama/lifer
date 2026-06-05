<script lang="ts">
  import { goto } from '$app/navigation'
  import { page } from '$app/state'
  import { apiGet, apiPost } from '$lib/api'
  import type { BirdCard, Species } from '../../../../types'
  import QuizSession from '$components/QuizSession.svelte'

  let deckId = $derived(page.params.id)
  let lane: 'audio' | 'image' = $derived(
    page.url.searchParams.get('lane') === 'image' ? 'image' : 'audio'
  )

  let card = $state<BirdCard | null>(null)
  let deckSpecies: Species[] = $state([])
  let done = $state(false)
  let reviewed = $state(0)
  let loading = $state(true)
  let error = $state('')
  // Pins the session: cards FSRS re-dues mid-session (short learning steps)
  // must not repeat within the same quiz run.
  let sessionStart = $state(new Date().toISOString())

  async function loadDeckSpecies() {
    try {
      deckSpecies = await apiGet(`/api/v1/decks/${deckId}/species`)
    } catch {
      // non-fatal -- typeahead will be empty but quiz still works
    }
  }

  async function fetchNext() {
    loading = true
    error = ''
    try {
      const next = await apiGet<BirdCard | undefined>(
        `/api/v1/decks/${deckId}/next?lane=${lane}&due_before=${encodeURIComponent(sessionStart)}`
      )
      if (!next) {
        done = true
        card = null
        return
      }
      card = next
    } catch {
      error = 'Failed to load next card.'
    } finally {
      loading = false
    }
  }

  async function onAdvance(correct: boolean) {
    if (!card) return
    const rating = correct ? 3 : 1
    try {
      await apiPost(`/api/v1/decks/${deckId}/rate`, {
        ebird_code: card.ebird_code,
        lane: card.lane,
        rating,
      })
    } catch {
      // non-fatal
    }
    reviewed += 1
    await fetchNext()
  }

  const total = $derived(reviewed + (card?.due_remaining ?? 0))
  const stats = $derived([
    { label: 'Reviewed', value: total > 0 ? `${reviewed}/${total}` : reviewed },
    { label: 'Lane', value: lane === 'audio' ? '🔊 Audio' : '👁 Image' },
  ])

  $effect(() => {
    if (deckId) {
      reviewed = 0
      done = false
      card = null
      deckSpecies = []
      sessionStart = new Date().toISOString()
      loadDeckSpecies()
      fetchNext()
    }
  })
</script>

{#key deckId}
  <QuizSession {card} species={deckSpecies} {stats} {loading} {error} {done} {onAdvance} onRetry={fetchNext}>
    {#snippet doneScreen()}
      <div class="done">
        <p class="done-icon">🎉</p>
        <p class="done-title">All caught up!</p>
        {#if reviewed > 0}
          <p class="done-sub">{reviewed} {reviewed === 1 ? 'card' : 'cards'} reviewed this session.</p>
        {/if}
        <p class="done-sub">Come back later when more cards are due.</p>
        <button class="btn-primary" onclick={() => goto(`/decks/${deckId}`)}>Back to deck</button>
      </div>
    {/snippet}
  </QuizSession>
{/key}

<style>
  .done button {
    padding: 0.75rem 1.5rem;
    font-size: 0.9375rem;
  }
</style>
