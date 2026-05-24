<script lang="ts">
  import { goto } from '$app/navigation'
  import { page } from '$app/state'
  import type { BirdCard } from '../../../../types'
  import QuizCard from '$components/QuizCard.svelte'
  import ImageQuizCard from '$components/ImageQuizCard.svelte'
  import RevealCard from '$components/RevealCard.svelte'
  import StatsBar from '$components/StatsBar.svelte'

  let groupId = $derived(page.params.id)
  let lane: 'audio' | 'image' = $derived(
    page.url.searchParams.get('lane') === 'image' ? 'image' : 'audio'
  )

  let card: BirdCard | null = $state(null)
  let revealed = $state(false)
  let done = $state(false)
  let reviewed = $state(0)
  let loading = $state(true)
  let error = $state('')

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

  async function onRate(rating: number) {
    if (!card) return
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
    await fetchNext()
  }

  function onReveal() {
    revealed = true
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
      <p>All done for now!</p>
      <button onclick={() => goto(`/groups/${groupId}`)}>Back to group</button>
    </div>
  {:else if card}
    {#if revealed}
      <RevealCard {card} {onRate} />
    {:else if lane === 'audio'}
      <QuizCard {card} {onReveal} />
    {:else}
      <ImageQuizCard {card} {onReveal} />
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
  .done p {
    color: var(--text);
    font-size: 1rem;
    font-weight: 600;
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
