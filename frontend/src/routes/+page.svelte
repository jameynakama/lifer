<script lang="ts">
  import { goto } from '$app/navigation'
  import type { Group } from '../types'
  import StatsBar from '$components/StatsBar.svelte'
  import GroupList from '$components/GroupList.svelte'

  let groups: Group[] = $state([])
  let loading = $state(true)

  $effect(() => {
    fetch('/api/v1/groups')
      .then(async (res) => {
        if (res.ok) groups = await res.json()
      })
      .finally(() => { loading = false })
  })

  const totalDue = $derived(groups.reduce((sum, g) => sum + g.audio_due + g.image_due, 0))

  const stats = $derived([
    { label: 'Due today', value: totalDue },
  ])

  function startPractice(group: Group, lane: 'audio' | 'image') {
    goto(`/groups/${group.id}/quiz?lane=${lane}`)
  }
</script>

<div class="dashboard">
  {#if loading}
    <p class="status">Loading...</p>
  {:else if groups.length === 0}
    <p class="empty">No groups yet. <a href="/groups">Create one</a> to get started.</p>
  {:else}
    <StatsBar {stats} />
    <GroupList {groups} onPractice={startPractice} />
  {/if}
</div>

<style>
  .dashboard {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }
  .status, .empty {
    text-align: center;
    color: var(--text-muted);
    padding: 2rem 0;
  }
  .empty a {
    color: var(--accent);
  }
</style>
