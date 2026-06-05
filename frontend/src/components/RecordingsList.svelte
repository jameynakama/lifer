<script lang="ts">
  import type { SpeciesRecording } from '../types'
  import WavePlayer from './WavePlayer.svelte'

  let { recordings }: { recordings: SpeciesRecording[] } = $props()
</script>

{#if recordings.length > 0}
  <div class="recordings-list">
    {#each recordings as rec (rec.xeno_canto_id)}
      <div class="recording-row">
        <div class="recording-meta">{rec.type} · {rec.quality}{#if rec.credit} · {rec.credit}{/if}</div>
        <WavePlayer url={rec.file_path} />
      </div>
    {/each}
  </div>
{:else}
  <p class="empty">No recordings available.</p>
{/if}

<style>
  .recordings-list {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .recording-row {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: 8px;
    padding: 0.5rem 0.75rem;
  }

  .recording-meta {
    font-size: 0.6875rem;
    color: var(--text-muted);
    text-transform: uppercase;
    letter-spacing: 0.05em;
    margin-bottom: 0.375rem;
  }

  .empty {
    color: var(--text-muted);
    font-size: 0.875rem;
    margin: 0;
  }
</style>
