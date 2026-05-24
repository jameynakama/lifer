<script lang="ts">
  import { goto } from '$app/navigation'
  import { page } from '$app/state'
  import type { BirdCard, Species } from '../../../../types'
  import QuizCard from '$components/QuizCard.svelte'
  import ImageQuizCard from '$components/ImageQuizCard.svelte'
  import RevealCard from '$components/RevealCard.svelte'
  import StatsBar from '$components/StatsBar.svelte'

  let groupId = $derived(page.params.id)
  let lane: 'audio' | 'image' = $derived(
    page.url.searchParams.get('lane') === 'image' ? 'image' : 'audio'
  )

  let card: BirdCard | null = $state(null)
  let groupSpecies: Species[] = $state([])
  let revealed = $state(false)
  let done = $state(false)
  let reviewed = $state(0)
  let loading = $state(true)
  let error = $state('')
  let guessed: Species | null = $state(null)
  let correct = $state(false)

  async function loadGroupSpecies() {
    try {
      const res = await fetch(`/api/v1/groups/${groupId}/species`)
      if (res.ok) groupSpecies = await res.json()
    } catch {
      // non-fatal -- typeahead will be empty but quiz still works
    }
  }

  async function fetchNext() {
    loading = true
    error = ''
    try {
      const res = await fetch(`/api/v1/groups/${groupId}/next?lane=${lane}`)
      if (res.status === 204) {
        done = true
        card = null
        return
      }
      if (!res.ok) throw new Error(`Server error ${res.status}`)
      card = await res.json()
    } catch {
      error = 'Failed to load next card.'
    } finally {
      loading = false
    }
  }

  function onReveal(selected: Species | null) {
    guessed = selected
    correct = selected !== null && card !== null && selected.id === card.species_id
    revealed = true
  }

  async function onNext() {
    if (!card) return
    const rating = correct ? 3 : 1
    try {
      await fetch(`/api/v1/groups/${groupId}/rate`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ species_id: card.species_id, lane: card.lane, rating }),
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

  const stats = $derived([
    { label: 'Reviewed', value: reviewed },
    { label: 'Lane', value: lane === 'audio' ? '🔊 Audio' : '👁 Image' },
  ])

  $effect(() => {
    if (groupId) {
      reviewed = 0
      done = false
      revealed = false
      card = null
      guessed = null
      correct = false
      groupSpecies = []
      loadGroupSpecies()
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
      <button onclick={() => goto(`/groups/${groupId}`)}>Back to group</button>
    </div>
  {:else if card}
    {#if revealed}
      <RevealCard {card} {correct} {guessed} {onNext} />
    {:else if lane === 'audio'}
      {#key card.species_id}
        <QuizCard {card} species={groupSpecies} {onReveal} />
      {/key}
    {:else}
      {#key card.species_id}
        <ImageQuizCard {card} species={groupSpecies} {onReveal} />
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
  .status {
    text-align: center;
    color: var(--text-muted);
    padding: 2rem 0;
  }
  .error {
    color: #b91c1c;
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
    background: var(--accent);
    color: #fff;
    border: none;
    border-radius: 10px;
    padding: 0.75rem 1.5rem;
    font-size: 0.9375rem;
    font-weight: 600;
    cursor: pointer;
    font-family: inherit;
  }
</style>
