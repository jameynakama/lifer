<script lang="ts">
  import type { Stat } from '../types'
  import StatsBar from './StatsBar.svelte'

  let {
    audioDue,
    imageDue,
    nextDueAt,
  }: {
    audioDue: number
    imageDue: number
    nextDueAt: string | null
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

  const stats: Stat[] = $derived([
    { label: 'DUE TODAY', value: totalDue },
    { label: 'AUDIO DUE', value: audioDue },
    { label: 'IMAGE DUE', value: imageDue },
    { label: 'NEXT DUE IN', value: countdown, highlight: totalDue > 0 },
  ])
</script>

<StatsBar variant="grid" {stats} />
