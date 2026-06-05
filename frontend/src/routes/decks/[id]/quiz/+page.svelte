<script lang="ts">
  import { goto } from '$app/navigation'
  import { page } from '$app/state'
  import { apiGet, apiPost } from '$lib/api'
  import type { BirdCard, Species } from '../../../../types'
  import QuizCard from '$components/QuizCard.svelte'
  import ImageQuizCard from '$components/ImageQuizCard.svelte'
  import RevealCard from '$components/RevealCard.svelte'
  import StatsBar from '$components/StatsBar.svelte'

  let deckId = $derived(page.params.id)
  let lane: 'audio' | 'image' = $derived(
    page.url.searchParams.get('lane') === 'image' ? 'image' : 'audio'
  )

  let card = $state<BirdCard | null>(null)
  let deckSpecies: Species[] = $state([])
  let revealed = $state(false)
  let done = $state(false)
  let reviewed = $state(0)
  let loading = $state(true)
  let error = $state('')
  let guessed: Species | null = $state(null)
  let correct = $state(false)

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
      const next = await apiGet<BirdCard | undefined>(`/api/v1/decks/${deckId}/next?lane=${lane}`)
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

  function onReveal(selected: Species | null) {
    guessed = selected
    correct = selected !== null && card !== null && selected.ebird_code === card.ebird_code
    revealed = true
  }

  async function onNext() {
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
    revealed = false
    guessed = null
    correct = false
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
      revealed = false
      card = null
      guessed = null
      correct = false
      deckSpecies = []
      loadDeckSpecies()
      fetchNext()
    }
  })
</script>

<div class="quiz">
  <StatsBar {stats} />

  {#if loading}
    <p class="status">Loading...</p>
  {:else if error}
    <p class="status error">{error}</p>
  {:else if done}
    <div class="done">
      <p class="done-icon">🎉</p>
      <p class="done-title">All caught up!</p>
      {#if reviewed > 0}
        <p class="done-sub">{reviewed} {reviewed === 1 ? 'card' : 'cards'} reviewed this session.</p>
      {/if}
      <p class="done-sub">Come back later when more cards are due.</p>
      <button class="btn-primary" onclick={() => goto(`/decks/${deckId}`)}>Back to deck</button>
    </div>
  {:else if card}
    {#if revealed}
      <RevealCard {card} {correct} {guessed} {onNext} />
    {:else if lane === 'audio'}
      {#key card.ebird_code}
        <QuizCard {card} species={deckSpecies} {onReveal} />
      {/key}
    {:else}
      {#key card.ebird_code}
        <ImageQuizCard {card} species={deckSpecies} {onReveal} />
      {/key}
    {/if}
  {:else}
    <p class="status error">Something went wrong. <button onclick={fetchNext}>Retry</button></p>
  {/if}
</div>

<style>
  .quiz {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }
  .error {
    color: var(--error);
  }
  .done {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 1rem;
    padding: 2rem 0;
  }
  .done-icon {
    font-size: 2.5rem;
    line-height: 1;
  }
  .done-title {
    color: var(--text);
    font-size: 1.125rem;
    font-weight: 700;
  }
  .done-sub {
    color: var(--text-muted);
    font-size: 0.875rem;
    text-align: center;
  }
  .done button {
    padding: 0.75rem 1.5rem;
    font-size: 0.9375rem;
  }
</style>
