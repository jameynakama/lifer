<script lang="ts">
  import { goto } from '$app/navigation'
  import { page } from '$app/state'
  import type { BirdCard, Species, Stat } from '../../../../types'
  import QuizCard from '$components/QuizCard.svelte'
  import ImageQuizCard from '$components/ImageQuizCard.svelte'
  import RevealCard from '$components/RevealCard.svelte'
  import StatsBar from '$components/StatsBar.svelte'

  let deckId = $derived(page.params.id)
  let lane: 'audio' | 'image' = $derived(
    page.url.searchParams.get('lane') === 'image' ? 'image' : 'audio'
  )

  let cards: BirdCard[] = $state([])
  let index = $state(0)
  let revealed = $state(false)
  let done = $state(false)
  let noMedia = $state(false)
  let loading = $state(true)
  let error = $state('')
  let guessed: Species | null = $state(null)
  let correct = $state(false)

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
      const res = await fetch(`/api/v1/decks/${deckId}/practice?lane=${lane}`)
      if (!res.ok) throw new Error(`Server error ${res.status}`)
      const data: BirdCard[] = await res.json()
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

  function onReveal(selected: Species | null) {
    guessed = selected
    correct = selected !== null && card !== null && selected.ebird_code === card.ebird_code
    revealed = true
  }

  function onNext() {
    if (index >= cards.length - 1) {
      done = true
      return
    }
    index += 1
    revealed = false
    guessed = null
    correct = false
  }

  function practiceAgain() {
    cards = shuffle(cards)
    index = 0
    revealed = false
    guessed = null
    correct = false
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

<div class="quiz">
  {#if !done && !noMedia && !error && cards.length > 0}
    <StatsBar {stats} />
  {/if}

  {#if loading}
    <p class="status">Loading...</p>
  {:else if error}
    <p class="status error">{error} <button onclick={loadCards}>Retry</button></p>
  {:else if noMedia}
    <div class="done">
      <p class="done-title">No species with media in this deck.</p>
      <button onclick={() => goto(`/decks/${deckId}`)}>Back to Deck</button>
    </div>
  {:else if done}
    <div class="done">
      <p class="done-icon">🎉</p>
      <p class="done-title">All done!</p>
      <p class="done-sub">{cards.length} species practiced.</p>
      <button class="btn-primary" onclick={practiceAgain}>Practice Again</button>
      <button class="btn-secondary" onclick={() => goto(`/decks/${deckId}`)}>Back to Deck</button>
    </div>
  {:else if card}
    {#if revealed}
      <RevealCard {card} {correct} {guessed} {onNext} />
    {:else if lane === 'audio'}
      {#key card.ebird_code}
        <QuizCard {card} {species} {onReveal} />
      {/key}
    {:else}
      {#key card.ebird_code}
        <ImageQuizCard {card} {species} {onReveal} />
      {/key}
    {/if}
  {:else}
    <p class="status error">Something went wrong. <button onclick={loadCards}>Retry</button></p>
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
  .btn-primary {
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
