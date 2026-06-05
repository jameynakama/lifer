<script lang="ts">
  import { apiGet, apiPatch } from '$lib/api'
  import type { User } from '../../../types'

  let users: User[] = $state([])
  let loading = $state(true)
  let error = $state('')

  async function loadUsers() {
    try {
      users = await apiGet<User[]>('/api/v1/admin/users')
    } catch (e) {
      error = (e as Error).message
    } finally {
      loading = false
    }
  }

  async function toggleAdmin(user: User) {
    const next = !user.is_admin
    try {
      await apiPatch(`/api/v1/admin/users/${user.id}`, { is_admin: next })
      users = users.map((u) => (u.id === user.id ? { ...u, is_admin: next } : u))
    } catch {
      // leave toggle unchanged
    }
  }

  loadUsers()
</script>

<div class="admin-users">
  <h1>Users</h1>

  {#if loading}
    <p>Loading...</p>
  {:else if error}
    <p class="error">{error}</p>
  {:else if users.length === 0}
    <p>No users.</p>
  {:else}
    <ul>
      {#each users as user (user.id)}
        <li>
          <div class="user-info">
            <span class="name">{user.name}</span>
            <span class="email muted">{user.email}</span>
          </div>
          <button
            class="admin-toggle"
            class:active={user.is_admin}
            onclick={() => toggleAdmin(user)}
          >
            {user.is_admin ? 'Admin' : 'Not admin'}
          </button>
        </li>
      {/each}
    </ul>
  {/if}
</div>

<style>
  ul {
    list-style: none;
    padding: 0;
  }
  li {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0.5rem 0;
    border-bottom: 1px solid var(--border);
  }
  .user-info {
    display: flex;
    flex-direction: column;
    gap: 0.15rem;
  }
  .name {
    font-weight: 500;
  }
  .muted {
    color: var(--text-muted);
    font-size: 0.875rem;
  }
  .admin-toggle {
    padding: 0.25rem 0.75rem;
    border: 1px solid var(--border);
    border-radius: 4px;
    background: transparent;
    color: var(--text-muted);
    cursor: pointer;
    font-size: 0.875rem;
  }
  .admin-toggle.active {
    background: var(--accent, #2563eb);
    color: #fff;
    border-color: transparent;
  }
  .error {
    color: var(--danger);
  }
</style>
