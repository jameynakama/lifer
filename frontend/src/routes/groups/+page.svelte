<script lang="ts">
  import type { Group } from '../../types'

  let groups: Group[] = $state([])
  let loading = $state(true)
  let newName = $state('')
  let creating = $state(false)

  async function loadGroups() {
    const res = await fetch('/api/v1/groups')
    if (res.ok) groups = await res.json()
    loading = false
  }

  async function createGroup() {
    if (!newName.trim()) return
    creating = true
    try {
      const res = await fetch('/api/v1/groups', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: newName.trim() }),
      })
      if (res.ok) {
        const created = await res.json()
        groups = [...groups, { ...created, audio_due: 0, image_due: 0 }]
        newName = ''
      }
    } finally {
      creating = false
    }
  }

  async function deleteGroup(id: number) {
    const res = await fetch(`/api/v1/groups/${id}`, { method: 'DELETE' })
    if (res.ok) {
      groups = groups.filter((g) => g.id !== id)
    }
  }

  loadGroups()
</script>

<div class="groups-page">
  <h1>Groups</h1>

  <form class="create-form" onsubmit={(e) => { e.preventDefault(); createGroup() }}>
    <input
      type="text"
      bind:value={newName}
      placeholder="Group name"
      disabled={creating}
    />
    <button type="submit" disabled={creating || !newName.trim()}>Create</button>
  </form>

  {#if loading}
    <p class="status">Loading...</p>
  {:else if groups.length === 0}
    <p class="empty">No groups yet. Create your first one above.</p>
  {:else}
    <ul class="group-list">
      {#each groups as group (group.id)}
        <li class="group-row">
          <a href="/groups/{group.id}">{group.name}</a>
          <button class="btn-delete" onclick={() => deleteGroup(group.id as unknown as number)}>Delete</button>
        </li>
      {/each}
    </ul>
  {/if}
</div>

<style>
  .groups-page {
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
  }
  h1 {
    font-size: 1.25rem;
    font-weight: 700;
    color: var(--text);
    margin: 0;
  }
  .create-form {
    display: flex;
    gap: 0.5rem;
  }
  .create-form input {
    flex: 1;
    background: var(--surface);
    border: 1px solid var(--border);
    color: var(--text);
    border-radius: 8px;
    padding: 0.5rem 0.75rem;
    font-size: 0.9375rem;
    font-family: inherit;
  }
  .create-form button {
    background: var(--accent);
    color: #fff;
    border: none;
    border-radius: 8px;
    padding: 0.5rem 1rem;
    font-size: 0.875rem;
    font-weight: 600;
    cursor: pointer;
    font-family: inherit;
  }
  .create-form button:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
  .group-list {
    list-style: none;
    padding: 0;
    margin: 0;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }
  .group-row {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: 10px;
    padding: 0.875rem 1rem;
    display: flex;
    justify-content: space-between;
    align-items: center;
    box-shadow: var(--shadow);
  }
  .group-row a {
    color: var(--text);
    font-weight: 600;
    text-decoration: none;
    font-size: 0.9375rem;
  }
  .btn-delete {
    background: transparent;
    border: 1px solid var(--border);
    color: var(--text-muted);
    border-radius: 6px;
    padding: 0.25rem 0.5rem;
    font-size: 0.75rem;
    cursor: pointer;
    font-family: inherit;
  }
  .status, .empty {
    text-align: center;
    color: var(--text-muted);
    padding: 2rem 0;
  }
</style>
