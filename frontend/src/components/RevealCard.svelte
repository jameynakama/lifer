<script lang="ts">
  import type { BirdCard, Species } from '../types'

  let { card, correct, guessed, onNext }: {
    card: BirdCard
    correct: boolean
    guessed: Species | null
    onNext: () => void
  } = $props()

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') onNext()
  }
</script>

<svelte:window onkeydown={handleKeydown} />

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

  <div class="photo-card">
    <div class="photo-wrapper">
      <img src={card.photo_url} alt={card.common_name} class="photo" />
    </div>
    <div class="species-info">
      <p class="common-name">{card.common_name}</p>
      <p class="scientific-name">{card.scientific_name}</p>
    </div>
  </div>

  <button class="btn-next" onclick={onNext}>Next</button>
</div>

<style>
  .reveal-card {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }
  .result-banner {
    border-radius: 10px;
    padding: 0.75rem 1rem;
    font-size: 0.9375rem;
    font-weight: 600;
  }
  .result-banner.correct {
    background: rgba(20, 83, 45, 0.15);
    color: #14532d;
    border: 1px solid rgba(20, 83, 45, 0.3);
  }
  .result-banner.incorrect {
    background: rgba(127, 29, 29, 0.15);
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
  .photo-card {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: 12px;
    overflow: hidden;
    box-shadow: var(--shadow);
  }
  .photo-wrapper {
    width: 100%;
    aspect-ratio: 4 / 3;
    overflow: hidden;
  }
  .photo {
    width: 100%;
    height: 100%;
    object-fit: cover;
    object-position: center top;
    display: block;
  }
  .species-info {
    padding: 0.875rem 1rem;
  }
  .common-name {
    color: var(--text);
    font-size: 1.0625rem;
    font-weight: 700;
    line-height: 1.2;
  }
  .scientific-name {
    color: var(--text-muted);
    font-size: 0.8125rem;
    font-style: italic;
    margin-top: 0.25rem;
  }
  .btn-next {
    background: var(--accent);
    color: #fff;
    border: none;
    border-radius: 10px;
    padding: 0.8125rem;
    font-size: 0.9375rem;
    font-weight: 600;
    cursor: pointer;
    font-family: inherit;
    width: 100%;
  }
  .btn-next:active {
    opacity: 0.85;
  }
</style>
