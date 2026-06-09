<script lang="ts">
  import type { StatsLanes } from '../types'
  import InfoTip from './InfoTip.svelte'

  let { lanes }: { lanes: StatsLanes } = $props()

  const pct = (banked: number, cards: number) =>
    cards > 0 ? Math.round((banked / cards) * 100) : 0
  const audioPct = $derived(pct(lanes.audio.banked, lanes.audio.cards))
  const imagePct = $derived(pct(lanes.image.banked, lanes.image.cards))
</script>

<section class="card panel">
  <h3 class="panel-title">
    Ear vs Eye<InfoTip
      text="Each bar shows how much of that lane is 'banked' — birds with stability of at least a week, the point where you're genuinely retaining them. In the gaps list, ✓ means that lane is banked and ✗ means it isn't yet."
    />
  </h3>
  <div class="vs">
    <div class="side">
      <span class="lane-label">🔊 Audio</span>
      <span class="pct">{audioPct}%</span>
      <div class="bar" aria-hidden="true">
        <span style="width: {audioPct}%"></span>
      </div>
      <span class="muted">{lanes.audio.banked}/{lanes.audio.cards} banked</span>
    </div>
    <div class="side">
      <span class="lane-label">👁 Image</span>
      <span class="pct">{imagePct}%</span>
      <div class="bar image" aria-hidden="true">
        <span style="width: {imagePct}%"></span>
      </div>
      <span class="muted">{lanes.image.banked}/{lanes.image.cards} banked</span>
    </div>
  </div>
  {#if lanes.gaps.length > 0}
    <p class="gaps">
      Biggest gaps:
      {#each lanes.gaps.slice(0, 5) as gap, i (gap.ebird_code)}
        {i > 0 ? ' · ' : ''}<a class="bird-link" href="/explore/{gap.ebird_code}"
          >{gap.common_name}</a
        >
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
    background: var(--tier-juvenile);
  }
  .gaps {
    margin-top: 0.75rem;
    font-size: 0.8125rem;
    color: var(--text-muted);
  }
</style>
