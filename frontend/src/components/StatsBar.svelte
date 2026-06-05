<script lang="ts">
  import type { Stat } from '../types'
  import InfoTip from './InfoTip.svelte'

  let {
    stats = [],
    variant = 'inline',
  }: {
    stats: Stat[]
    /** inline: compact quiz-header bar; grid: dashboard stat tiles. */
    variant?: 'inline' | 'grid'
  } = $props()
</script>

<div class="stats-bar {variant}">
  {#each stats as stat (stat.label)}
    <div class="stat" class:card={variant === 'grid'}>
      <span class="value" class:now={stat.highlight ?? false}>{stat.value}</span>
      <span class="label">
        {stat.label}{#if stat.tip}<InfoTip text={stat.tip} />{/if}
      </span>
    </div>
  {/each}
</div>

<style>
  .value {
    color: var(--text);
    font-weight: 700;
    line-height: 1;
  }
  .value.now {
    color: var(--accent);
  }
  .label {
    color: var(--text-muted);
    font-size: 0.5625rem;
    text-transform: uppercase;
    letter-spacing: 0.06em;
  }

  /* inline: compact quiz-header bar */
  .stats-bar.inline {
    display: flex;
    gap: 0.5rem;
  }
  .inline .stat {
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
  .inline .value {
    font-size: 1.25rem;
  }
  .inline .label {
    margin-top: 0.25rem;
  }

  /* grid: dashboard stat tiles */
  .stats-bar.grid {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 0.625rem;
  }
  .grid .stat {
    padding: 0.875rem 1rem;
    display: flex;
    flex-direction: column;
  }
  .grid .value {
    font-size: 1.5rem;
  }
  .grid .label {
    margin-top: 0.3rem;
  }

  @media (max-width: 639px) {
    .stats-bar.grid {
      grid-template-columns: repeat(2, 1fr);
    }
  }
</style>
