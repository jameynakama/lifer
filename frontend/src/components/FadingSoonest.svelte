<script lang="ts">
  import type { StatsFading } from '../types'

  let {
    fading,
    remember,
  }: {
    fading: StatsFading[]
    remember: { now: number; in_a_week: number; in_a_month: number }
  } = $props()
</script>

<div class="fading-soonest">
  <p>
    You'd remember ~{remember.now} now · ~{remember.in_a_week} in a week · ~{remember.in_a_month} in a
    month
  </p>
  {#each fading as item (item.ebird_code + item.lane)}
    <div class="fading-row">{item.common_name}</div>
  {/each}
</div>
