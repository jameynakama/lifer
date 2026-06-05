<script lang="ts">
  import type { StatsProgress } from '../types'

  let { progress }: { progress: StatsProgress } = $props()
</script>

<div class="progress-breakdown">
  <!-- full markup in Task 10 -->
  {progress.not_seen}
</div>
