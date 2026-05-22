<script lang="ts">
  import type { Stat } from '../types'
  let { stats = [] }: { stats: Stat[] } = $props()
</script>

<div class="stats-bar">
  {#each stats as stat}
    <div class="stat">
      <span class="value">{stat.value}</span>
      <span class="label">{stat.label}</span>
    </div>
  {/each}
</div>

<style>
  .stats-bar {
    display: flex;
    gap: 0.5rem;
  }
  .stat {
    background: var(--surface);
    border-radius: 8px;
    padding: 0.625rem 0.875rem;
    flex: 1;
    text-align: center;
    box-shadow: var(--shadow);
    display: flex;
    flex-direction: column;
    align-items: center;
  }
  .value {
    color: var(--text);
    font-size: 1.25rem;
    font-weight: 700;
    line-height: 1;
  }
  .label {
    color: var(--text-muted);
    font-size: 0.5625rem;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    margin-top: 0.25rem;
  }
</style>
