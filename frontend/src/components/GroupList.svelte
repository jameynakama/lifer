<script lang="ts">
  import type { Group } from '../types'

  let {
    groups,
    onPractice,
  }: {
    groups: Group[];
    onPractice: (group: Group, lane: 'audio' | 'image') => void;
  } = $props()
</script>

<div class="group-list">
  {#each groups as group}
    <div class="group-card">
      <div class="group-info">
        <span class="group-name">{group.name}</span>
        {#if group.is_preset}<span class="preset-badge">Preset</span>{/if}
      </div>
      <div class="group-actions">
        {#if group.audio_due > 0}
          <button class="btn-lane" onclick={() => onPractice(group, 'audio')}>
            🔊 Audio · {group.audio_due}
          </button>
        {/if}
        {#if group.image_due > 0}
          <button class="btn-lane" onclick={() => onPractice(group, 'image')}>
            👁 Image · {group.image_due}
          </button>
        {/if}
        {#if group.audio_due === 0 && group.image_due === 0}
          <span class="all-done">All done</span>
        {/if}
      </div>
    </div>
  {/each}
</div>

<style>
  .group-list {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }
  .group-card {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: 10px;
    padding: 0.875rem 1rem;
    display: flex;
    justify-content: space-between;
    align-items: center;
    box-shadow: var(--shadow);
  }
  .group-info {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }
  .group-name {
    font-size: 0.9375rem;
    font-weight: 600;
    color: var(--text);
  }
  .preset-badge {
    font-size: 0.625rem;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--text-muted);
    background: var(--border);
    border-radius: 4px;
    padding: 0.125rem 0.375rem;
  }
  .group-actions {
    display: flex;
    gap: 0.5rem;
  }
  .btn-lane {
    background: var(--surface);
    border: 1px solid var(--border);
    color: var(--text-secondary);
    border-radius: 8px;
    padding: 0.375rem 0.625rem;
    font-size: 0.75rem;
    font-weight: 600;
    cursor: pointer;
    font-family: inherit;
    box-shadow: var(--shadow);
  }
  .all-done {
    font-size: 0.75rem;
    color: var(--text-muted);
  }
</style>
