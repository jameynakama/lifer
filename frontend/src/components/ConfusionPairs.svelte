<script lang="ts">
  import type { StatsConfusion } from '../types'

  let { confusions }: { confusions: StatsConfusion[] } = $props()
</script>

<div class="confusion-pairs">
  {#if confusions.length === 0}
    <p>No confusions yet — flawless, or just getting started.</p>
  {:else}
    {#each confusions as pair (pair.actual.ebird_code + pair.guessed.ebird_code)}
      <div class="pair">
        <span>{pair.actual.common_name}</span>
        <span>{pair.guessed.common_name}</span>
      </div>
    {/each}
  {/if}
</div>
