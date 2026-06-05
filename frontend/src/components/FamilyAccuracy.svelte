<script lang="ts">
  import type { StatsFamily } from '../types'

  let { families }: { families: StatsFamily[] } = $props()
</script>

<div class="family-accuracy">
  {#each families as fam (fam.family)}
    <div class="family-row">{fam.family}</div>
  {/each}
</div>
