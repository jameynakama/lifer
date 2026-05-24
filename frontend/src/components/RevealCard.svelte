<script lang="ts">
  import type { BirdCard, Species } from '../types'

  let { card, correct, guessed, onRate }: {
    card: BirdCard
    correct: boolean
    guessed: Species | null
    onRate: (rating: number) => void
  } = $props()

  const suggestedRating = $derived(correct ? 3 : 1)

  const ratings = [
    { label: 'Again', value: 1 },
    { label: 'Hard', value: 2 },
    { label: 'Good', value: 3 },
    { label: 'Easy', value: 4 },
  ]
</script>

<div class="reveal-card">
  <div class="result-banner" class:correct class:incorrect={!correct}>
    {#if correct}
      <span>✓ {card.common_name}</span>
    {:else if guessed}
      <span>✗ You guessed: {guessed.common_name}</span>
    {:else}
      <span>✗ You didn't know</span>
    {/if}
  </div>

  <img src={card.photo_url} alt={card.common_name} class="photo" />
  <div class="species">
    <p class="common-name">{card.common_name}</p>
    <p class="scientific-name">{card.scientific_name}</p>
  </div>
  <p class="how-well">How well did you know it?</p>
  <div class="ratings">
    {#each ratings as rating}
      <button
        class="rating-{rating.label.toLowerCase()}"
        class:suggested={rating.value === suggestedRating}
        onclick={() => onRate(rating.value)}
      >
        {rating.label}
      </button>
    {/each}
  </div>
</div>

<style>
  .reveal-card {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }
  .result-banner {
    border-radius: 8px;
    padding: 0.625rem 0.875rem;
    font-size: 0.875rem;
    font-weight: 600;
  }
  .result-banner.correct {
    background: rgba(20, 83, 45, 0.12);
    color: #14532d;
    border: 1px solid rgba(20, 83, 45, 0.3);
  }
  .result-banner.incorrect {
    background: rgba(127, 29, 29, 0.12);
    color: #7f1d1d;
    border: 1px solid rgba(127, 29, 29, 0.3);
  }
  :global([data-theme="dark"]) .result-banner.correct {
    background: rgba(20, 83, 45, 0.25);
    color: #4ade80;
    border-color: rgba(20, 83, 45, 0.6);
  }
  :global([data-theme="dark"]) .result-banner.incorrect {
    background: rgba(127, 29, 29, 0.25);
    color: #f87171;
    border-color: rgba(127, 29, 29, 0.6);
  }
  .photo {
    width: 100%;
    border-radius: 10px;
    max-height: 200px;
    object-fit: cover;
  }
  .common-name {
    color: var(--text);
    font-size: 1rem;
    font-weight: 700;
  }
  .scientific-name {
    color: var(--text-muted);
    font-size: 0.8125rem;
    font-style: italic;
  }
  .how-well {
    color: var(--text-muted);
    font-size: 0.6875rem;
    text-transform: uppercase;
    letter-spacing: 0.06em;
  }
  .ratings {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 0.5rem;
  }
  .ratings button {
    border: none;
    border-radius: 8px;
    padding: 0.625rem 0.25rem;
    font-size: 0.75rem;
    font-weight: 600;
    cursor: pointer;
    font-family: inherit;
    color: #fff;
  }
  .rating-again { background: #7f1d1d; }
  .rating-hard  { background: #78350f; }
  .rating-good  { background: #14532d; }
  .rating-easy  { background: #1e3a8a; }
  .ratings button.suggested {
    outline: 3px solid rgba(255, 255, 255, 0.85);
    outline-offset: 2px;
  }
</style>
