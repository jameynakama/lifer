<script lang="ts">
  import { goto } from '$app/navigation'
  import { page } from '$app/state'
  import { apiGet } from '$lib/api'
  import type { BirdCard, Species, Stat } from '../../../../types'
  import QuizSession from '$components/QuizSession.svelte'

  let deckId = $derived(page.params.id)
  let lane: 'audio' | 'image' = $derived(
    page.url.searchParams.get('lane') === 'image' ? 'image' : 'audio'
  )

  let cards: BirdCard[] = $state([])
  let index = $state(0)
  let done = $state(false)
  let noMedia = $state(false)
  let loading = $state(true)
  let error = $state('')

  let card: BirdCard | null = $derived(cards[index] ?? null)
  let species: Species[] = $derived(
    cards.map((c) => ({
      ebird_code: c.ebird_code,
      common_name: c.common_name,
      scientific_name: c.scientific_name,
    }))
  )

  function shuffle(arr: BirdCard[]): BirdCard[] {
    const a = [...arr]
    for (let i = a.length - 1; i > 0; i--) {
      const j = Math.floor(Math.random() * (i + 1))
      ;[a[i], a[j]] = [a[j], a[i]]
    }
    return a
  }

  async function loadCards() {
    loading = true
    error = ''
    noMedia = false
    done = false
    cards = []
    index = 0
    try {
      const data = await apiGet<BirdCard[]>(`/api/v1/decks/${deckId}/practice?lane=${lane}`)
      if (data.length === 0) {
        noMedia = true
        return
      }
      cards = shuffle(data)
      index = 0
    } catch {
      error = 'Failed to load practice cards.'
    } finally {
      loading = false
    }
  }

  function onAdvance() {
    if (index >= cards.length - 1) {
      done = true
      return
    }
    index += 1
  }

  function practiceAgain() {
    cards = shuffle(cards)
    index = 0
    done = false
  }

  const stats: Stat[] = $derived([
    { label: 'Practiced', value: `${index + 1} / ${cards.length}` },
    { label: 'Lane', value: lane === 'audio' ? '🔊 Audio' : '👁 Image' },
  ])

  $effect(() => {
    if (deckId) loadCards()
  })
</script>

{#key deckId}
  <QuizSession
    {card}
    {species}
    {stats}
    showStats={!done && !noMedia && !error && cards.length > 0}
    {loading}
    {error}
    done={done || noMedia}
    {onAdvance}
    onRetry={loadCards}
  >
    {#snippet doneScreen()}
      {#if noMedia}
        <div class="done">
          <p class="done-title">No species with media in this deck.</p>
          <button onclick={() => goto(`/decks/${deckId}`)}>Back to Deck</button>
        </div>
      {:else}
        <div class="done">
          <p class="done-icon">🎉</p>
          <p class="done-title">All done!</p>
          <p class="done-sub">{cards.length} species practiced.</p>
          <button class="btn-primary" onclick={practiceAgain}>Practice Again</button>
          <button class="btn-secondary" onclick={() => goto(`/decks/${deckId}`)}>Back to Deck</button>
        </div>
      {/if}
    {/snippet}
  </QuizSession>
{/key}

<style>
  .btn-primary {
    padding: 0.75rem 1.5rem;
    font-size: 0.9375rem;
  }
  .btn-secondary {
    background: transparent;
    border: 1px solid var(--border);
    color: var(--text-muted);
    border-radius: 10px;
    padding: 0.75rem 1.5rem;
    font-size: 0.9375rem;
    font-weight: 600;
    cursor: pointer;
    font-family: inherit;
  }
</style>
