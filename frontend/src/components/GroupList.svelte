<script lang="ts">
  import { type Group } from '../types'
  let { groups, onPractice }: { groups: Group[], onPractice: (g: Group) => void } = $props()
</script>

<ul class="group-list">
  {#each groups as group}
    <li>
      <div class="info">
        <span class="name">{group.name}</span>
        <span class="due">{group.due_count} due</span>
      </div>
      <button onclick={() => onPractice(group)}>Practice</button>
    </li>
  {/each}
</ul>

<style>
  .group-list {
    list-style: none;
    padding: 0;
    margin: 0;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }
  li {
    background: var(--surface);
    border-radius: 10px;
    padding: 0.75rem 0.875rem;
    display: flex;
    justify-content: space-between;
    align-items: center;
    box-shadow: var(--shadow);
  }
  .info {
    display: flex;
    flex-direction: column;
    gap: 0.125rem;
  }
  .name {
    color: var(--text);
    font-size: 0.875rem;
    font-weight: 600;
  }
  .due {
    color: var(--text-muted);
    font-size: 0.6875rem;
  }
  button {
    background: var(--accent-hover);
    color: #fff;
    border: none;
    border-radius: 6px;
    padding: 0.375rem 0.75rem;
    font-size: 0.75rem;
    font-weight: 600;
    cursor: pointer;
    font-family: inherit;
  }
</style>
