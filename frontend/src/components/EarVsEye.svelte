<script lang="ts">
  import type { StatsLanes } from '../types'

  let { lanes }: { lanes: StatsLanes } = $props()
</script>

<div class="ear-vs-eye">
  <h2>Ear vs Eye</h2>
  <!-- full markup in Task 10 -->
  {lanes.audio.cards}
</div>
