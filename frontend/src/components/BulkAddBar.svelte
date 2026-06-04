<script lang="ts">
  interface Props {
    selectedCount: number
    onAdd: () => void
    onClear: () => void
  }
  let { selectedCount, onAdd, onClear }: Props = $props()
</script>

{#if selectedCount > 0}
  <div class="bulk-bar" role="toolbar">
    <span class="count">{selectedCount} selected</span>
    <div class="actions">
      <button class="btn-clear" onclick={onClear}>Clear</button>
      <button class="btn-add" onclick={onAdd}>Add {selectedCount} species</button>
    </div>
  </div>
{/if}

<style>
  .bulk-bar {
    position: sticky;
    bottom: 1rem;
    background: var(--surface);
    border: 1px solid var(--accent);
    border-radius: 10px;
    padding: 0.625rem 1rem;
    display: flex;
    align-items: center;
    justify-content: space-between;
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.15);
    z-index: 10;
  }
  .count {
    font-size: 0.875rem;
    color: var(--text-muted);
    font-weight: 600;
  }
  .actions {
    display: flex;
    gap: 0.5rem;
  }
  .btn-clear {
    background: transparent;
    border: 1px solid var(--border);
    color: var(--text-muted);
    border-radius: 6px;
    padding: 0.375rem 0.625rem;
    font-size: 0.8125rem;
    cursor: pointer;
    font-family: inherit;
  }
  .btn-add {
    background: var(--accent);
    border: none;
    color: #fff;
    border-radius: 6px;
    padding: 0.375rem 0.875rem;
    font-size: 0.8125rem;
    font-weight: 600;
    cursor: pointer;
    font-family: inherit;
  }
</style>
