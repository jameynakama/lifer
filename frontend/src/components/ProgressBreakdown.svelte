<script lang="ts">
  import { createQuery } from '@tanstack/svelte-query'
  import { apiGet } from '$lib/api'
  import InfoTip from './InfoTip.svelte'
  import type { StatsProgress, StatsTierBird } from '../types'

  let { progress, lane = '' }: { progress: StatsProgress; lane?: string } = $props()

  const TIERS = [
    { key: 'egg', label: 'Egg', color: 'var(--tier-egg)' },
    { key: 'nestling', label: 'Nestling', color: 'var(--tier-nestling)' },
    { key: 'fledgling', label: 'Fledgling', color: 'var(--tier-fledgling)' },
    { key: 'juvenile', label: 'Juvenile', color: 'var(--tier-juvenile)' },
    { key: 'immature', label: 'Immature', color: 'var(--tier-immature)' },
    { key: 'adult', label: 'Adult', color: 'var(--tier-adult)' },
  ] as const

  const segments = $derived(TIERS.map((t) => ({ ...t, count: progress[t.key] })))
  let expanded: string | null = $state(null)
  function toggle(key: string) {
    expanded = expanded === key ? null : key
  }

  const birdsQuery = createQuery(() => ({
    queryKey: ['stats', 'tier', expanded, lane],
    enabled: expanded !== null,
    queryFn: (): Promise<StatsTierBird[]> =>
      apiGet(`/api/v1/stats/tier/${expanded}${lane ? `?lane=${lane}` : ''}`),
  }))
</script>

<section class="card panel">
  <h3 class="panel-title">
    Life cycle<InfoTip
      text="A bird's-eye view of where each card sits in its learning journey: new, in progress, or deeply embedded."
    />
  </h3>
  <div class="bar" aria-hidden="true">
    {#each segments as s (s.key)}
      {#if s.count > 0}
        <span style="flex: {s.count}; background: {s.color}"></span>
      {/if}
    {/each}
  </div>
  <ul class="tiers list-reset">
    {#each segments as s (s.key)}
      <li>
        <button class="tier" class:empty={s.count === 0} onclick={() => toggle(s.key)}>
          <span class="swatch" style="background: {s.color}"></span>
          <span class="label">{s.label}</span>
          <span class="count">{s.count}</span>
        </button>
        {#if expanded === s.key}
          <div class="birds">
            {#if birdsQuery.isPending}
              <p class="muted">Loading…</p>
            {:else if birdsQuery.isError}
              <p class="muted">Couldn't load birds.</p>
            {:else if birdsQuery.data && birdsQuery.data.length > 0}
              <ul class="list-reset">
                {#each birdsQuery.data as b (b.ebird_code + b.lane)}
                  <li>
                    <a class="bird" href="/explore/{b.ebird_code}"
                      >{b.lane === 'audio' ? '🔊' : '👁'} {b.common_name}</a
                    >
                  </li>
                {/each}
              </ul>
            {:else}
              <p class="muted">No birds here yet.</p>
            {/if}
          </div>
        {/if}
      </li>
    {/each}
  </ul>
</section>

<style>
  .bar {
    display: flex;
    height: 12px;
    border-radius: 6px;
    overflow: hidden;
    gap: 1px;
  }
  .tiers {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
    margin-top: 0.75rem;
  }
  .tier {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    width: 100%;
    background: transparent;
    border: none;
    cursor: pointer;
    padding: 0.25rem;
    font-family: inherit;
    color: var(--text);
    text-align: left;
  }
  .tier.empty {
    opacity: 0.45;
  }
  .swatch {
    width: 12px;
    height: 12px;
    border-radius: 3px;
    flex-shrink: 0;
  }
  .label {
    flex: 1;
    font-size: 0.875rem;
  }
  .count {
    font-variant-numeric: tabular-nums;
    font-weight: 600;
  }
  .birds {
    padding: 0.25rem 0 0.5rem 1.5rem;
    font-size: 0.8125rem;
  }
  .birds li {
    padding: 0.1rem 0;
  }
  .bird {
    color: var(--text);
    text-decoration: none;
  }
  .bird:hover {
    text-decoration: underline;
    color: var(--accent);
  }
  .muted {
    color: var(--text-muted);
  }
</style>
