<script lang="ts">
  import type { StatsConfusion } from '../types'

  let { confusions }: { confusions: StatsConfusion[] } = $props()
</script>

<section class="card panel">
  <h3 class="panel-title">Confusion pairs</h3>
  {#if confusions.length === 0}
    <p class="muted">No confusions yet — flawless, or just getting started.</p>
  {:else}
    <ul class="list-reset">
      {#each confusions as pair (`${pair.actual.ebird_code}:${pair.guessed.ebird_code}`)}
        <li class="row">
          <span class="names">
            <b>{pair.actual.common_name}</b>
            <span class="muted"> — you said </span>
            {pair.guessed.common_name}
          </span>
          <span class="count">{pair.count}×</span>
        </li>
      {/each}
    </ul>
  {/if}
</section>

<style>
  .row {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
    gap: 0.75rem;
    padding: 0.3125rem 0;
    font-size: 0.875rem;
    color: var(--text);
  }
  .count {
    color: var(--error);
    font-weight: 700;
    flex-shrink: 0;
  }
</style>
