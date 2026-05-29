<script lang="ts">
  let {
    total,
    offset,
    limit,
    onPageChange,
  }: {
    total: number
    offset: number
    limit: number
    onPageChange: (offset: number) => void
  } = $props()

  const totalPages = $derived(Math.ceil(total / limit))
  const currentPage = $derived(Math.floor(offset / limit) + 1)

  function pages(): (number | '...')[] {
    if (totalPages <= 7) {
      return Array.from({ length: totalPages }, (_, i) => i + 1)
    }
    const result: (number | '...')[] = [1]
    if (currentPage > 3) result.push('...')
    for (let p = Math.max(2, currentPage - 1); p <= Math.min(totalPages - 1, currentPage + 1); p++) {
      result.push(p)
    }
    if (currentPage < totalPages - 2) result.push('...')
    result.push(totalPages)
    return result
  }

  function goToPage(page: number) {
    onPageChange((page - 1) * limit)
  }
</script>

{#if totalPages > 1}
  <div class="pagination" role="navigation" aria-label="Pagination">
    <button
      class="page-btn"
      disabled={currentPage === 1}
      onclick={() => goToPage(currentPage - 1)}
      aria-label="Previous page"
    >
      &lt;
    </button>

    {#each pages() as p}
      {#if p === '...'}
        <span class="ellipsis">…</span>
      {:else}
        <button
          class="page-btn"
          class:active={p === currentPage}
          onclick={() => goToPage(p as number)}
          aria-label="Page {p}"
          aria-current={p === currentPage ? 'page' : undefined}
        >
          {p}
        </button>
      {/if}
    {/each}

    <button
      class="page-btn"
      disabled={currentPage === totalPages}
      onclick={() => goToPage(currentPage + 1)}
      aria-label="Next page"
    >
      &gt;
    </button>
  </div>
{/if}

<style>
  .pagination {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 0.25rem;
    padding: 0.5rem 0;
  }

  .page-btn {
    background: var(--surface);
    border: 1px solid var(--border);
    color: var(--text-secondary);
    border-radius: 6px;
    padding: 0.3125rem 0.625rem;
    font-size: 0.875rem;
    cursor: pointer;
    font-family: inherit;
    min-width: 32px;
    text-align: center;
    box-shadow: var(--shadow);
  }

  .page-btn:hover:not(:disabled) {
    border-color: var(--accent);
    color: var(--accent);
  }

  .page-btn.active {
    background: var(--accent);
    border-color: var(--accent);
    color: #fff;
    font-weight: 600;
  }

  .page-btn:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .ellipsis {
    color: var(--text-muted);
    padding: 0 0.25rem;
    font-size: 0.875rem;
  }
</style>
