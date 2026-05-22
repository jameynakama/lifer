<script lang="ts">
  import type { BirdCard } from '../types'
  let { card, onReveal }: { card: BirdCard, onReveal: () => void } = $props()
  let guess: string = $state('')
</script>

<div class="quiz-card">
  <div class="audio-wrapper">
    <audio controls src={card.recording_path}>Your browser does not support audio.</audio>
  </div>
  <input
    bind:value={guess}
    type="text"
    placeholder="Type species name"
    class="guess-input"
  />
  <button class="btn-reveal" onclick={onReveal}>Reveal answer</button>
</div>

<style>
  .quiz-card {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }
  .audio-wrapper {
    background: var(--surface);
    border-radius: 8px;
    padding: 0.25rem;
    box-shadow: var(--shadow);
  }
  audio {
    width: 100%;
    display: block;
  }
  .guess-input {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: 8px;
    padding: 0.6875rem 0.875rem;
    width: 100%;
    font-size: 0.875rem;
    color: var(--text);
    font-family: inherit;
    outline: none;
  }
  .guess-input:focus {
    border-color: var(--accent);
  }
  .btn-reveal {
    background: var(--surface);
    border: 1px solid var(--border);
    color: var(--text-secondary);
    border-radius: 10px;
    padding: 0.75rem;
    width: 100%;
    font-size: 0.875rem;
    font-weight: 600;
    cursor: pointer;
    font-family: inherit;
    box-shadow: var(--shadow);
  }
</style>
