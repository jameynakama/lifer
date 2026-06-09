<script lang="ts">
  import type { StatsConfusion } from '../types'
  import InfoTip from './InfoTip.svelte'

  let { confusions }: { confusions: StatsConfusion[] } = $props()
</script>

<section class="card panel">
  <h3 class="panel-title">
    Confusion pairs<InfoTip
      text="Specific wrong answers, counted: the bird shown vs the bird you named. Your personal lookalikes and soundalikes."
    />
  </h3>
  {#if confusions.length === 0}
    <p class="muted">No confusions yet — flawless, or just getting started.</p>
  {:else}
    <ul class="list-reset">
      {#each confusions as pair (`${pair.actual.ebird_code}:${pair.guessed.ebird_code}`)}
        <li class="row">
          <span class="names">
            <a class="bird-link" href="/explore/{pair.actual.ebird_code}"
              ><b>{pair.actual.common_name}</b></a
            >
            <span class="muted"> — you said </span>
            <a class="bird-link" href="/explore/{pair.guessed.ebird_code}"
              >{pair.guessed.common_name}</a
            >
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
