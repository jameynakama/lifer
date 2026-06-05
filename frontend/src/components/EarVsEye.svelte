<script lang="ts">
  import type { StatsLanes } from '../types'

  let { lanes }: { lanes: StatsLanes } = $props()

  const pct = (known: number, cards: number) => (cards > 0 ? Math.round((known / cards) * 100) : 0)
  const audioPct = $derived(pct(lanes.audio.known, lanes.audio.cards))
  const imagePct = $derived(pct(lanes.image.known, lanes.image.cards))
</script>

<section class="card panel">
  <h3 class="panel-title">Ear vs Eye</h3>
  <div class="vs">
    <div class="side">
      <span class="lane-label">🔊 Audio</span>
      <span class="pct">{audioPct}%</span>
      <div class="bar" aria-hidden="true">
        <span style="width: {audioPct}%"></span>
      </div>
      <span class="muted">{lanes.audio.known}/{lanes.audio.cards} known</span>
    </div>
    <div class="side">
      <span class="lane-label">👁 Image</span>
      <span class="pct">{imagePct}%</span>
      <div class="bar image" aria-hidden="true">
        <span style="width: {imagePct}%"></span>
      </div>
      <span class="muted">{lanes.image.known}/{lanes.image.cards} known</span>
    </div>
  </div>
  {#if lanes.gaps.length > 0}
    <p class="gaps">
      Biggest gaps:
      {#each lanes.gaps.slice(0, 5) as gap, i (gap.ebird_code)}
        {i > 0 ? ' · ' : ''}{gap.common_name}
        ({gap.known_lane === 'audio' ? '🔊' : '👁'} ✓ / {gap.weak_lane === 'audio' ? '🔊' : '👁'} ✗)
      {/each}
    </p>
  {/if}
</section>

<style>
  .vs {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 1rem;
  }
  .side {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
  }
  .lane-label {
    font-size: 0.8125rem;
    color: var(--text);
  }
  .pct {
    font-size: 1.375rem;
    font-weight: 700;
    color: var(--text);
  }
  .bar {
    height: 8px;
    border-radius: 4px;
    background: var(--bg);
    overflow: hidden;
  }
  .bar span {
    display: block;
    height: 100%;
    background: var(--accent);
  }
  .bar.image span {
    background: var(--success);
  }
  .gaps {
    margin-top: 0.75rem;
    font-size: 0.8125rem;
    color: var(--text-muted);
  }
</style>
