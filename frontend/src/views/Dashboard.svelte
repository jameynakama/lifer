<script lang="ts">
  import type { Group } from '../types'
  import { session } from '../stores/session'
  import { view } from '../stores/view'
  import StatsBar from '../components/StatsBar.svelte'
  import GroupList from '../components/GroupList.svelte'

  const MOCK_GROUPS: Group[] = [
    { id: '1', name: 'Pacific Northwest', is_preset: true, audio_due: 8, image_due: 5 },
    { id: '2', name: 'My Warblers', is_preset: false, audio_due: 3, image_due: 0 },
  ]

  const groups = MOCK_GROUPS
  const totalDue = groups.reduce((sum, g) => sum + g.audio_due + g.image_due, 0)

  const stats = [
    { label: 'Due today', value: totalDue },
    { label: 'Day streak', value: 5 },
    { label: 'Species', value: 47 },
  ]

  function startPractice(group: Group, lane: 'audio' | 'image') {
    $session = { groupId: group.id, lane }
    $view = 'quiz'
  }
</script>

<div class="dashboard">
  <StatsBar {stats} />
  <GroupList {groups} onPractice={startPractice} />
</div>

<style>
  .dashboard {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }
</style>
