<script lang="ts">
  import type { StatsFamily } from '../types'
  import InfoTip from './InfoTip.svelte'

  let { families }: { families: StatsFamily[] } = $props()

  const pct = (f: StatsFamily) => Math.round((f.correct / f.attempts) * 100)
</script>

<section class="card panel">
  <h3 class="panel-title">
    By family<InfoTip
      text="Answer accuracy from your whole review history, grouped by family, worst first. Small attempt counts swing wildly — 100% on three tries is luck, not mastery."
    />
  </h3>
  {#if families.length === 0}
    <p class="muted">No family stats yet — they build as you review.</p>
  {:else}
    <ul class="list-reset">
      {#each families as f (f.family)}
        {@const p = pct(f)}
        <li class="row">
          <span>{f.family}</span>
          <span class="pct" class:low={p < 50}>{p}%</span>
        </li>
      {/each}
    </ul>
  {/if}
</section>

<style>
  .row {
    display: flex;
    justify-content: space-between;
    padding: 0.3125rem 0;
    font-size: 0.875rem;
    color: var(--text);
  }
  .pct {
    font-weight: 700;
    color: var(--success);
  }
  .pct.low {
    color: var(--error);
  }
</style>
