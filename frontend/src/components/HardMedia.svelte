<script lang="ts">
  import type { StatsHardMedia } from '../types'

  let { media }: { media: StatsHardMedia[] } = $props()
</script>

<section class="card panel">
  <h3 class="panel-title">Hard media (≥3 looks)</h3>
  {#if media.length === 0}
    <p class="muted">No repeat offenders yet — needs three looks at the same recording or photo.</p>
  {:else}
    <ul class="list-reset">
      {#each media as m (`${m.lane}:${m.media_id}`)}
        <li class="row">
          <span>
            {m.lane === 'audio' ? '🔊' : '👁'}
            {m.common_name}
            {#if m.media_url}
              <a
                href={m.media_url}
                target="_blank"
                rel="noopener noreferrer"
                aria-label={`Open ${m.lane === 'audio' ? 'recording' : 'photo'} ${m.media_id} (new tab)`}
                class="muted">({m.media_id})</a
              >
            {:else}
              <span class="muted">({m.media_id})</span>
            {/if}
          </span>
          <span class="score">{m.correct}/{m.attempts}</span>
        </li>
      {/each}
    </ul>
  {/if}
</section>

<style>
  .row {
    display: flex;
    justify-content: space-between;
    gap: 0.75rem;
    padding: 0.3125rem 0;
    font-size: 0.875rem;
    color: var(--text);
  }
  .score {
    color: var(--error);
    font-weight: 700;
    flex-shrink: 0;
  }
</style>
