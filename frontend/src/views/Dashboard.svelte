<script lang="ts">
  import type { Group } from '../types'
  import { session } from '../stores/session'
  import { view } from '../stores/view'
  import StatsBar from '../components/StatsBar.svelte'
  import GroupList from '../components/GroupList.svelte'

  const MOCK_GROUPS: Group[] = [
    { id: '1', name: 'Pacific Northwest', is_preset: true, due_count: 8 },
    { id: '2', name: 'My Warblers', is_preset: false, due_count: 3 },
  ]

  const groups = MOCK_GROUPS
  const topGroup = groups[0]

  const stats = [
    { label: 'Due today', value: groups.reduce((sum, g) => sum + g.due_count, 0) },
    { label: 'Day streak', value: 5 },
    { label: 'Species', value: 47 },
  ]

  function startPractice(group: Group) {
    $session = { groupId: group.id }
    $view = 'quiz'
  }
</script>

<div>
  <StatsBar {stats} />
  <button onclick={() => startPractice(topGroup)}>Start Practice</button>
  <p>{topGroup.name} · {topGroup.due_count} due</p>
  <GroupList {groups} onPractice={startPractice} />
</div>
