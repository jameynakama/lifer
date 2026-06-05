<script lang="ts">
  import type { StatsProgress } from '../types'

  let { progress }: { progress: StatsProgress } = $props()

  const segments = $derived(
    (
      [
        { key: 'known', label: 'Known', count: progress.known, color: 'var(--success)' },
        { key: 'learning', label: 'Learning', count: progress.learning, color: 'var(--accent)' },
        {
          key: 'relearning',
          label: 'Relearning',
          count: progress.relearning,
          color: 'var(--error)',
        },
        { key: 'not_seen', label: 'Not seen', count: progress.not_seen, color: 'var(--border)' },
      ] as const
    ).filter((s) => s.count > 0),
  )
  const total = $derived(segments.reduce((n, s) => n + s.count, 0))
</script>

<section class="card panel">
  <h3 class="panel-title">Progress</h3>
  {#if total === 0}
    <p class="muted">No cards yet — add a deck to get started.</p>
  {:else}
    <div class="bar" role="img" aria-label="Progress breakdown">
      {#each segments as s (s.key)}
        <span style="width: {(s.count / total) * 100}%; background: {s.color}"></span>
      {/each}
    </div>
    <ul class="legend list-reset">
      {#each segments as s (s.key)}
        <li><span class="dot" style="background: {s.color}"></span>{s.label} ({s.count})</li>
      {/each}
    </ul>
  {/if}
</section>

<style>
  .bar {
    display: flex;
    height: 12px;
    border-radius: 6px;
    overflow: hidden;
    background: var(--bg);
  }
  .legend {
    display: flex;
    flex-wrap: wrap;
    gap: 0.375rem 1rem;
    margin-top: 0.625rem;
    font-size: 0.8125rem;
    color: var(--text);
  }
  .legend li {
    display: flex;
    align-items: center;
    gap: 0.375rem;
  }
  .dot {
    width: 9px;
    height: 9px;
    border-radius: 50%;
    flex-shrink: 0;
  }
</style>
