<script lang="ts">
  import type { BirdCard, Species } from '../types'
  import SpeciesTypeahead from './SpeciesTypeahead.svelte'

  let { card, species, onReveal }: {
    card: BirdCard
    species: Species[]
    onReveal: (selected: Species | null) => void
  } = $props()

  let selected: Species | null = $state(null)
</script>

<div class="quiz-card">
  <div class="image-wrapper">
    <img src={card.media_url} alt="Identify this bird" class="quiz-photo" />
  </div>
  <SpeciesTypeahead {species} onSelect={(s) => { selected = s }} />
  <div class="actions">
    <button
      class="btn-reveal"
      onclick={() => onReveal(selected)}
      disabled={selected === null}
    >
      Reveal answer
    </button>
    <button class="btn-skip" onclick={() => onReveal(null)}>
      I don't know
    </button>
  </div>
</div>

<style>
  .quiz-card {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }
  .image-wrapper {
    background: var(--surface);
    border-radius: 8px;
    overflow: hidden;
    box-shadow: var(--shadow);
  }
  .quiz-photo {
    width: 100%;
    display: block;
    max-height: 280px;
    object-fit: cover;
  }
  .actions {
    display: grid;
    grid-template-columns: 1fr auto;
    gap: 0.5rem;
  }
  .btn-reveal {
    background: var(--surface);
    border: 1px solid var(--border);
    color: var(--text-secondary);
    border-radius: 10px;
    padding: 0.75rem;
    font-size: 0.875rem;
    font-weight: 600;
    cursor: pointer;
    font-family: inherit;
    box-shadow: var(--shadow);
  }
  .btn-reveal:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }
  .btn-skip {
    background: transparent;
    border: 1px solid var(--border);
    color: var(--text-muted);
    border-radius: 10px;
    padding: 0.75rem 1rem;
    font-size: 0.8125rem;
    cursor: pointer;
    font-family: inherit;
    white-space: nowrap;
  }
</style>
