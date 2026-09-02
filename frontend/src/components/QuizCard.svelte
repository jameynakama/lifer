<script lang="ts">
  import type { BirdCard, Species } from '../types'
  import SpeciesTypeahead from './SpeciesTypeahead.svelte'
  import WavePlayer from './WavePlayer.svelte'

  let {
    card,
    species,
    onReveal,
  }: {
    card: BirdCard
    species: Species[]
    onReveal: (selected: Species | null) => void
  } = $props()

  let selected: Species | null = $state(null)
  let player: WavePlayer | undefined = $state()

  function reveal(s: Species | null) {
    player?.stop()
    onReveal(s)
  }
</script>

<div class="quiz-card">
  {#if card.lane === 'audio'}
    <div class="audio-card card">
      <WavePlayer url={card.media_url} peaks={card.peaks} bind:this={player} />
    </div>
  {:else}
    <div class="photo-card card">
      <div class="photo-wrapper">
        <img src={card.media_url} alt="Identify this bird" class="quiz-photo" />
      </div>
      <p class="prompt">What bird is this?</p>
    </div>
  {/if}
  <SpeciesTypeahead
    {species}
    onSelect={(s) => {
      selected = s
    }}
    onSubmit={() => {
      if (selected) reveal(selected)
    }}
  />
  <div class="actions">
    <button
      class="btn-reveal btn-primary"
      onclick={() => reveal(selected)}
      disabled={selected === null}
    >
      Reveal answer
    </button>
    <button class="btn-skip" onclick={() => reveal(null)}> I don't know </button>
  </div>
</div>

<style>
  .quiz-card {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }
  .audio-card {
    border-radius: var(--radius-lg);
    padding: 1rem;
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
  .quiz-photo {
    width: 100%;
    height: 100%;
    object-fit: cover;
    object-position: center top;
    display: block;
  }
  .prompt {
    padding: 0.625rem 1rem;
    font-size: 0.8125rem;
    color: var(--text-muted);
    font-style: italic;
  }
  .actions {
    display: grid;
    grid-template-columns: 1fr auto;
    gap: 0.5rem;
  }
  .btn-reveal {
    padding: 0.8125rem;
    font-size: 0.9375rem;
  }
  .btn-reveal:disabled {
    opacity: 0.35;
  }
  .btn-skip {
    background: var(--surface);
    border: 1px solid var(--border);
    color: var(--text-muted);
    border-radius: 10px;
    padding: 0.8125rem 1rem;
    font-size: 0.875rem;
    cursor: pointer;
    font-family: inherit;
    white-space: nowrap;
    box-shadow: var(--shadow);
  }
</style>
