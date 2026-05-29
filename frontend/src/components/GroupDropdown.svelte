<script lang="ts">
  import { createQuery, createMutation, useQueryClient } from '@tanstack/svelte-query'

  let {
    ebird_code,
    onClose,
  }: {
    ebird_code: string
    onClose: () => void
  } = $props()

  const queryClient = useQueryClient()

  const groupsQuery = createQuery(() => ({
    queryKey: ['groups'],
    queryFn: () => fetch('/api/v1/groups').then((r) => r.json()),
  }))

  const membershipQuery = createQuery(() => ({
    queryKey: ['species', ebird_code, 'groups'],
    queryFn: () =>
      fetch(`/api/v1/species/${ebird_code}/groups`)
        .then((r) => r.json())
        .then((d) => (d.group_ids as number[]) ?? []),
  }))

  const addMutation = createMutation(() => ({
    mutationFn: (groupId: number) =>
      fetch(`/api/v1/groups/${groupId}/species`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ebird_code }),
      }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['species', ebird_code, 'groups'] }),
  }))

  const removeMutation = createMutation(() => ({
    mutationFn: (groupId: number) =>
      fetch(`/api/v1/groups/${groupId}/species/${ebird_code}`, { method: 'DELETE' }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['species', ebird_code, 'groups'] }),
  }))

  const createGroupMutation = createMutation(() => ({
    mutationFn: async (name: string) => {
      const res = await fetch('/api/v1/groups', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name }),
      })
      const group = await res.json()
      await fetch(`/api/v1/groups/${group.id}/species`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ebird_code }),
      })
      return group
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['groups'] })
      queryClient.invalidateQueries({ queryKey: ['species', ebird_code, 'groups'] })
      newGroupName = ''
    },
  }))

  let newGroupName = $state('')
  let mutationError = $state('')

  function toggle(groupId: number, currentlyIn: boolean) {
    mutationError = ''
    if (currentlyIn) {
      removeMutation.mutate(groupId, {
        onError: () => { mutationError = 'Failed. Try again.' },
      })
    } else {
      addMutation.mutate(groupId, {
        onError: () => { mutationError = 'Failed. Try again.' },
      })
    }
  }

  function createGroup() {
    if (!newGroupName.trim()) return
    mutationError = ''
    createGroupMutation.mutate(newGroupName.trim(), {
      onError: () => { mutationError = 'Failed. Try again.' },
    })
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') createGroup()
    if (e.key === 'Escape') onClose()
  }
</script>

<div class="dropdown" role="dialog" aria-label="Add to group">
  <div class="create-section">
    <input
      class="create-input"
      type="text"
      placeholder="New group name…"
      bind:value={newGroupName}
      onkeydown={handleKeydown}
    />
    <button
      class="create-btn"
      onclick={createGroup}
      disabled={!newGroupName.trim() || createGroupMutation.isPending}
    >
      {createGroupMutation.isPending ? 'Creating…' : '+ Create group'}
    </button>
  </div>

  <div class="groups-list">
    {#if groupsQuery.isPending || membershipQuery.isPending}
      <p class="loading-msg">Loading…</p>
    {:else if !groupsQuery.data || groupsQuery.data.length === 0}
      <p class="loading-msg">No groups yet.</p>
    {:else}
      {#each groupsQuery.data as group (group.id)}
        {@const isMember = (membershipQuery.data ?? []).includes(group.id)}
        <label class="group-item">
          <input
            type="checkbox"
            checked={isMember}
            onchange={() => toggle(group.id, isMember)}
          />
          {group.name}
        </label>
      {/each}
    {/if}
  </div>

  {#if mutationError}
    <p class="error-msg">{mutationError}</p>
  {/if}
</div>

<style>
  .dropdown {
    position: absolute;
    top: calc(100% + 4px);
    right: 0;
    width: 220px;
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: 10px;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
    z-index: 100;
    overflow: hidden;
  }

  .create-section {
    padding: 0.625rem 0.75rem;
    border-bottom: 1px solid var(--border);
    display: flex;
    flex-direction: column;
    gap: 0.375rem;
  }

  .create-input {
    background: var(--bg);
    border: 1px solid var(--border);
    color: var(--text);
    border-radius: 6px;
    padding: 0.375rem 0.5rem;
    font-size: 0.8125rem;
    font-family: inherit;
    width: 100%;
    box-sizing: border-box;
  }

  .create-btn {
    background: var(--accent);
    color: #fff;
    border: none;
    border-radius: 6px;
    padding: 0.3125rem 0.625rem;
    font-size: 0.75rem;
    font-weight: 600;
    cursor: pointer;
    font-family: inherit;
    width: 100%;
  }

  .create-btn:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .groups-list {
    max-height: 200px;
    overflow-y: auto;
    padding: 0.375rem 0;
  }

  .group-item {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.5rem 0.75rem;
    cursor: pointer;
    font-size: 0.875rem;
    color: var(--text);
  }

  .group-item:hover {
    background: var(--bg);
  }

  .group-item input[type='checkbox'] {
    accent-color: var(--accent);
    flex-shrink: 0;
  }

  .loading-msg {
    padding: 0.5rem 0.75rem;
    font-size: 0.8125rem;
    color: var(--text-muted);
    margin: 0;
  }

  .error-msg {
    padding: 0.25rem 0.75rem 0.5rem;
    font-size: 0.75rem;
    color: #ef4444;
    margin: 0;
  }
</style>
