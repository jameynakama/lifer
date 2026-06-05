<script lang="ts">
  import type { Snippet } from 'svelte'
  import type { BirdCard, Species, Stat } from '../types'
  import QuizCard from './QuizCard.svelte'
  import RevealCard from './RevealCard.svelte'
  import StatsBar from './StatsBar.svelte'

  let {
    card,
    species,
    stats,
    showStats = true,
    loading,
    error,
    done,
    onAdvance,
    onRetry,
    doneScreen,
  }: {
    card: BirdCard | null
    species: Species[]
    stats: Stat[]
    showStats?: boolean
    loading: boolean
    error: string
    done: boolean
    /** Called when the user advances past a reveal, with whether the guess was correct, and what was guessed (null = I don't know). */
    onAdvance: (correct: boolean, guessed: Species | null) => void | Promise<void>
    onRetry: () => void
    doneScreen: Snippet
  } = $props()

  let revealed = $state(false)
  let guessed: Species | null = $state(null)
  let correct = $state(false)

  function onReveal(selected: Species | null) {
    guessed = selected
    correct = selected !== null && card !== null && selected.ebird_code === card.ebird_code
    revealed = true
  }

  async function onNext() {
    // Reset only after the advance completes so the reveal stays visible
    // while the next card (and any rating POST) is in flight.
    await onAdvance(correct, guessed)
    revealed = false
    guessed = null
    correct = false
  }
</script>

<div class="quiz">
  {#if showStats}
    <StatsBar {stats} />
  {/if}

  {#if loading}
    <p class="status">Loading...</p>
  {:else if error}
    <p class="status error">{error} <button onclick={onRetry}>Retry</button></p>
  {:else if done}
    {@render doneScreen()}
  {:else if card}
    {#if revealed}
      <RevealCard {card} {correct} {guessed} {onNext} />
    {:else}
      {#key card.ebird_code}
        <QuizCard {card} {species} {onReveal} />
      {/key}
    {/if}
  {:else}
    <p class="status error">Something went wrong. <button onclick={onRetry}>Retry</button></p>
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
</style>
