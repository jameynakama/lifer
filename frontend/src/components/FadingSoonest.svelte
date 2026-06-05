<script lang="ts">
  import type { StatsFading } from '../types'
  import InfoTip from './InfoTip.svelte'

  let {
    fading,
    remember,
  }: {
    fading: StatsFading[]
    remember: { now: number; in_a_week: number; in_a_month: number }
  } = $props()
</script>

<section class="card panel">
  <h3 class="panel-title">
    Fading soonest<InfoTip
      text="Known birds whose memory the model predicts is decaying fastest. The next ones you'd likely miss without a refresher."
    />
  </h3>
  <p class="remember muted">
    You'd remember ~{remember.now} now · ~{remember.in_a_week} in a week · ~{remember.in_a_month} in a
    month
  </p>
  {#if fading.length === 0}
    <p class="muted">Nothing is fading — no graduated cards yet.</p>
  {:else}
    <ul class="list-reset">
      {#each fading as f (`${f.ebird_code}:${f.lane}`)}
        <li class="row">
          <span>{f.lane === 'audio' ? '🔊' : '👁'} {f.common_name}</span>
          <span class="muted"
            >{Math.min(100, Math.round(f.retrievability * 100))}% · due {f.due_in_days === 0
              ? 'now'
              : `in ${f.due_in_days}d`}</span
          >
        </li>
      {/each}
    </ul>
  {/if}
</section>

<style>
  .remember {
    font-size: 0.8125rem;
    margin-bottom: 0.5rem;
  }
  .row {
    display: flex;
    justify-content: space-between;
    gap: 0.75rem;
    padding: 0.3125rem 0;
    font-size: 0.875rem;
    color: var(--text);
  }
</style>
