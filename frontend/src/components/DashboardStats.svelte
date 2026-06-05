<script lang="ts">
  let {
    audioDue,
    imageDue,
    nextDueAt,
  }: {
    audioDue: number;
    imageDue: number;
    nextDueAt: string | null;
  } = $props()

  const totalDue = $derived(audioDue + imageDue)

  function formatCountdown(target: Date): string {
    const ms = target.getTime() - Date.now()
    if (ms <= 0) return 'Now'
    const totalMinutes = Math.floor(ms / 60_000)
    const hours = Math.floor(totalMinutes / 60)
    const minutes = totalMinutes % 60
    if (hours > 0) return `${hours}h ${minutes}m`
    if (minutes > 0) return `${minutes}m`
    return '<1m'
  }

  let countdown = $state<string>('--')
  let interval: ReturnType<typeof setInterval> | null = null

  $effect(() => {
    if (interval) {
      clearInterval(interval)
      interval = null
    }

    if (totalDue > 0) {
      countdown = 'Now'
      return
    }

    if (!nextDueAt) {
      countdown = '--'
      return
    }

    const target = new Date(nextDueAt)
    countdown = formatCountdown(target)
    interval = setInterval(() => {
      countdown = formatCountdown(target)
    }, 30_000)

    return () => {
      if (interval) {
        clearInterval(interval)
        interval = null
      }
    }
  })
</script>

<div class="stats-bar">
  <div class="stat card">
    <span class="value">{totalDue}</span>
    <span class="label">DUE TODAY</span>
  </div>
  <div class="stat card">
    <span class="value">{audioDue}</span>
    <span class="label">AUDIO DUE</span>
  </div>
  <div class="stat card">
    <span class="value">{imageDue}</span>
    <span class="label">IMAGE DUE</span>
  </div>
  <div class="stat card">
    <span class="value" class:now={totalDue > 0}>{countdown}</span>
    <span class="label">NEXT DUE IN</span>
  </div>
</div>

<style>
  .stats-bar {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 0.625rem;
  }
  .stat {
    padding: 0.875rem 1rem;
    display: flex;
    flex-direction: column;
  }
  .value {
    color: var(--text);
    font-size: 1.5rem;
    font-weight: 700;
    line-height: 1;
  }
  .value.now {
    color: var(--accent);
  }
  .label {
    color: var(--text-muted);
    font-size: 0.5625rem;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    margin-top: 0.3rem;
  }

  @media (max-width: 639px) {
    .stats-bar {
      grid-template-columns: repeat(2, 1fr);
    }
  }
</style>
