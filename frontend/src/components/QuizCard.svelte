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
  <div class="audio-card">
    <audio controls src={card.media_url}>Your browser does not support audio.</audio>
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
  .audio-card {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: 12px;
    padding: 1rem;
    box-shadow: var(--shadow);
  }
  audio {
    width: 100%;
    display: block;
  }
  .actions {
    display: grid;
    grid-template-columns: 1fr auto;
    gap: 0.5rem;
  }
  .btn-reveal {
    background: var(--accent);
    color: #fff;
    border: none;
    border-radius: 10px;
    padding: 0.8125rem;
    font-size: 0.9375rem;
    font-weight: 600;
    cursor: pointer;
    font-family: inherit;
  }
  .btn-reveal:disabled {
    opacity: 0.35;
    cursor: not-allowed;
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
