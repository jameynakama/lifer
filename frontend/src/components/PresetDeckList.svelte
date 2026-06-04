<script lang="ts">
  import type { PresetDeck } from '../types'

  interface Props {
    presetDecks: PresetDeck[]
    cloning: Set<number>
    onClone: (id: number) => void
  }
  let { presetDecks, cloning, onClone }: Props = $props()
</script>

<ul class="preset-list">
  {#each presetDecks as preset (preset.id)}
    <li class="preset-row">
      <div class="preset-info">
        <span class="preset-name">{preset.name}</span>
        {#if preset.description}
          <span class="preset-description">{preset.description}</span>
        {/if}
        <span class="species-count">{preset.species_count} species</span>
      </div>
      <button
        class="btn-clone"
        disabled={cloning.has(preset.id)}
        onclick={() => onClone(preset.id)}
      >{cloning.has(preset.id) ? 'Cloning…' : 'Clone'}</button>
    </li>
  {/each}
</ul>

<style>
  .preset-list {
    list-style: none;
    padding: 0;
    margin: 0;
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }
  .preset-row {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: 10px;
    padding: 0.875rem 1rem;
    display: flex;
    align-items: center;
    gap: 0.75rem;
    box-shadow:
      -1px 2px 0 0 var(--surface),
      -1px 2px 0 1px var(--border),
      -3px 5px 0 0 var(--surface),
      -3px 5px 0 1px var(--border),
      var(--shadow);
  }
  .preset-info {
    display: flex;
    flex-direction: column;
    gap: 0.125rem;
    flex: 1;
    min-width: 0;
  }
  .preset-name {
    color: var(--text);
    font-weight: 600;
    font-size: 0.9375rem;
  }
  .preset-description {
    font-size: 0.8125rem;
    color: var(--text-muted);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .species-count {
    font-size: 0.75rem;
    color: var(--text-muted);
  }
  .btn-clone {
    flex-shrink: 0;
    background: var(--surface);
    border: 1px solid var(--accent);
    color: var(--accent);
    border-radius: 8px;
    padding: 0.375rem 0.75rem;
    font-size: 0.8125rem;
    font-weight: 600;
    cursor: pointer;
    font-family: inherit;
  }
  .btn-clone:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
</style>
