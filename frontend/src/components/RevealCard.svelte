<script lang="ts">
  import type { BirdCard, Species } from '../types'
  import WavePlayer from './WavePlayer.svelte'

  let {
    card,
    correct,
    guessed,
    onNext,
  }: {
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

  {#if card.lane === 'audio'}
    <div class="audio-card card">
      {#if card.recording_type}
        <span class="recording-type">{card.recording_type}</span>
      {/if}
      <WavePlayer url={card.media_url} />
      {#if card.recording_credit}
        <p class="credit">{card.recording_credit}</p>
      {/if}
    </div>
  {/if}

  <div class="photo-card card">
    <div class="photo-wrapper">
      <img src={card.photo_url} alt={card.common_name} class="photo" />
    </div>
    <div class="species-info">
      <p class="common-name">{card.common_name}</p>
      <p class="scientific-name">{card.scientific_name}</p>
      {#if card.photo_credit}
        <p class="credit">{card.photo_credit}</p>
      {/if}
    </div>
  </div>

  <button class="btn-next btn-primary" onclick={onNext}>Next</button>
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
    animation: popIn 280ms ease-out;
  }
  .result-banner.incorrect {
    background: rgba(127, 29, 29, 0.15);
    color: #7f1d1d;
    border: 1px solid rgba(127, 29, 29, 0.3);
    animation: shake 340ms ease-out;
  }
  :global([data-theme='dark']) .result-banner.correct {
    background: rgba(20, 83, 45, 0.25);
    color: var(--success);
    border-color: rgba(20, 83, 45, 0.6);
  }
  :global([data-theme='dark']) .result-banner.incorrect {
    background: rgba(127, 29, 29, 0.25);
    color: var(--error);
    border-color: rgba(127, 29, 29, 0.6);
  }
  .audio-card {
    border-radius: var(--radius-lg);
    padding: 1rem;
    position: relative;
  }
  .recording-type {
    display: inline-block;
    font-size: 0.6875rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--text-muted);
    background: var(--bg);
    border: 1px solid var(--border);
    border-radius: 4px;
    padding: 0.125rem 0.375rem;
    margin-bottom: 0.5rem;
  }
  .photo-card {
    border-radius: var(--radius-lg);
    overflow: hidden;
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
    animation: photoReveal 350ms ease-out;
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
  .credit {
    color: var(--text-muted);
    font-size: 0.6875rem;
    margin-top: 0.375rem;
  }
  .btn-next {
    padding: 0.8125rem;
    font-size: 0.9375rem;
    width: 100%;
  }
  .btn-next:active {
    opacity: 0.85;
  }

  @keyframes popIn {
    0% {
      transform: scale(0.88);
    }
    60% {
      transform: scale(1.05);
    }
    100% {
      transform: scale(1);
    }
  }

  @keyframes shake {
    0% {
      transform: translateX(0);
    }
    20% {
      transform: translateX(-8px);
    }
    40% {
      transform: translateX(7px);
    }
    60% {
      transform: translateX(-5px);
    }
    80% {
      transform: translateX(4px);
    }
    100% {
      transform: translateX(0);
    }
  }

  @keyframes photoReveal {
    from {
      transform: scale(1.07);
      opacity: 0.6;
    }
    to {
      transform: scale(1);
      opacity: 1;
    }
  }
</style>
