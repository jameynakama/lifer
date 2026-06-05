<script lang="ts">
  import { page } from '$app/state'
  import { createQuery } from '@tanstack/svelte-query'
  import { statsQueryOptions } from '$lib/queries'
  import type { Stat } from '../../types'
  import StatsBar from '$components/StatsBar.svelte'
  import ProgressBreakdown from '$components/ProgressBreakdown.svelte'
  import EarVsEye from '$components/EarVsEye.svelte'
  import ConfusionPairs from '$components/ConfusionPairs.svelte'
  import FamilyAccuracy from '$components/FamilyAccuracy.svelte'
  import FadingSoonest from '$components/FadingSoonest.svelte'
  import HardMedia from '$components/HardMedia.svelte'
  import DangerZone from '$components/DangerZone.svelte'

  let lane = $derived(
    page.url.searchParams.get('lane') === 'audio'
      ? ('audio' as const)
      : page.url.searchParams.get('lane') === 'image'
        ? ('image' as const)
        : ('' as const),
  )

  const statsQuery = createQuery(() => statsQueryOptions(lane))
  const stats = $derived(statsQuery.data)

  const accuracy = $derived(
    stats && stats.totals.attempts > 0
      ? Math.round((stats.totals.correct / stats.totals.attempts) * 100)
      : null,
  )
  const totalsRow: Stat[] = $derived(
    stats
      ? [
          {
            label: 'Birds',
            value: stats.totals.species,
            tip: 'Distinct species with at least one card.',
          },
          {
            label: 'Known',
            value: `${stats.totals.known}/${stats.totals.cards}`,
            highlight: true,
            tip: 'Cards graduated to long-term review, out of all your cards. A scheduler state, not a score.',
          },
          {
            label: 'Accuracy',
            value: accuracy === null ? '—' : `${accuracy}%`,
            tip: 'Correct answers across your whole logged review history.',
          },
          {
            label: 'This week',
            value: stats.totals.reviews_last_7d,
            tip: 'Reviews answered in the last 7 days.',
          },
        ]
      : [],
  )

  const tabs = [
    { href: '/stats', label: 'Combined', lane: '' },
    { href: '/stats?lane=audio', label: '🔊 Audio', lane: 'audio' },
    { href: '/stats?lane=image', label: '👁 Image', lane: 'image' },
  ]
</script>

<div class="stats-page">
  <h1>Your Progress</h1>

  <nav class="tabs" aria-label="Stats lane">
    {#each tabs as tab (tab.lane)}
      <a
        href={tab.href}
        class:active={lane === tab.lane}
        aria-current={lane === tab.lane ? 'page' : undefined}>{tab.label}</a
      >
    {/each}
  </nav>

  {#if statsQuery.isPending}
    <p class="status">Loading...</p>
  {:else if statsQuery.isError || !stats}
    <p class="status">Couldn't load stats. Try again in a bit.</p>
  {:else}
    <StatsBar variant="grid" stats={totalsRow} />
    <ProgressBreakdown progress={stats.progress} />
    {#if stats.lanes}
      <EarVsEye lanes={stats.lanes} />
    {/if}
    <ConfusionPairs confusions={stats.confusions} />
    <div class="pair">
      <FamilyAccuracy families={stats.families} />
      <FadingSoonest fading={stats.fading} remember={stats.remember} />
    </div>
    <HardMedia media={stats.hard_media} />
  {/if}
  <DangerZone />
</div>

<style>
  .stats-page {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }
  h1 {
    font-size: 1.25rem;
    font-weight: 700;
    color: var(--text);
  }
  .tabs {
    display: flex;
    gap: 0.25rem;
    border-bottom: 1px solid var(--border);
  }
  .tabs a {
    padding: 0.5rem 0.875rem;
    font-size: 0.875rem;
    color: var(--text-muted);
    text-decoration: none;
    border-bottom: 2px solid transparent;
  }
  .tabs a.active {
    color: var(--text);
    border-bottom-color: var(--accent);
  }
  .pair {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 0.625rem;
  }
  @media (max-width: 639px) {
    .pair {
      grid-template-columns: 1fr;
    }
  }
</style>
