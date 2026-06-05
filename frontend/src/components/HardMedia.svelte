<script lang="ts">
  import type { StatsHardMedia } from '../types'

  let { media }: { media: StatsHardMedia[] } = $props()
</script>

<div class="hard-media">
  {#each media as item (item.media_id)}
    <div class="hard-media-row">
      <span>{item.common_name}</span>
      <span>{item.media_id}</span>
    </div>
  {/each}
</div>
