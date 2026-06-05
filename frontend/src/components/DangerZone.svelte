<script lang="ts">
  import { useQueryClient } from '@tanstack/svelte-query'
  import { apiPost } from '$lib/api'
  import { queryKeys } from '$lib/queries'

  type Scope = 'schedule' | 'everything'
  type ResetResponse = { cards_deleted: number; reviews_deleted: number; cards_seeded: number }

  const rows: { scope: Scope; label: string; copy: string }[] = [
    {
      scope: 'schedule',
      label: 'Reset schedule',
      copy: 'Forget all learning progress. Every species starts fresh; your review history and stats survive.',
    },
    {
      scope: 'everything',
      label: 'Reset everything',
      copy: 'Delete all learning progress and review history. Clean slate.',
    },
  ]

  const queryClient = useQueryClient()

  let armed: Scope | null = $state(null)
  let confirmText = $state('')
  let busy = $state(false)
  let message = $state('')
  let failed = $state(false)

  function toggle(scope: Scope) {
    armed = armed === scope ? null : scope
    confirmText = ''
    message = ''
    failed = false
  }

  async function fire(scope: Scope) {
    busy = true
    failed = false
    try {
      const res = await apiPost<ResetResponse>('/api/v1/reset', { scope })
      const deleted =
        scope === 'everything'
          ? `Deleted ${res.cards_deleted.toLocaleString()} cards and ${res.reviews_deleted.toLocaleString()} reviews.`
          : `Deleted ${res.cards_deleted.toLocaleString()} cards.`
      message = `${deleted} ${res.cards_seeded.toLocaleString()} fresh cards ready.`
      armed = null
      confirmText = ''
      // Card deletion changes due counts app-wide; review_log deletion
      // changes every stats panel. Mark both stale so observers refetch.
      queryClient.invalidateQueries({ queryKey: queryKeys.decks })
      queryClient.invalidateQueries({ queryKey: queryKeys.statsAll })
    } catch {
      failed = true
    } finally {
      busy = false
    }
  }
</script>

<section class="card panel danger-zone">
  <h2 class="panel-title">Danger zone</h2>
  {#each rows as row (row.scope)}
    <div class="row">
      <div class="row-text">
        <p class="row-label">{row.label}</p>
        <p class="row-copy">{row.copy}</p>
      </div>
      <button class="btn-danger-ghost arm" onclick={() => toggle(row.scope)}>
        {row.label}
      </button>
    </div>
    {#if armed === row.scope}
      <div class="confirm-strip">
        <input
          type="text"
          aria-label="Type RESET to confirm"
          placeholder="Type RESET to confirm"
          bind:value={confirmText}
        />
        <button
          class="btn-confirm"
          disabled={confirmText !== 'RESET' || busy}
          onclick={() => fire(row.scope)}
        >
          Confirm
        </button>
      </div>
    {/if}
  {/each}
  {#if message}
    <p class="result">{message}</p>
  {/if}
  {#if failed}
    <p class="result failed">Reset failed. Try again.</p>
  {/if}
</section>

<style>
  .danger-zone {
    border-color: var(--danger);
  }
  .danger-zone :global(.panel-title) {
    color: var(--danger);
  }
  .row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.75rem;
    padding: 0.5rem 0;
  }
  .row-label {
    font-size: 0.875rem;
    font-weight: 600;
    color: var(--text);
  }
  .row-copy {
    font-size: 0.8125rem;
    color: var(--text-muted);
  }
  .arm {
    flex-shrink: 0;
    padding: 0.375rem 0.75rem;
    font-size: 0.8125rem;
  }
  .confirm-strip {
    display: flex;
    gap: 0.5rem;
    padding: 0.25rem 0 0.625rem;
  }
  .confirm-strip input {
    flex: 1;
    min-width: 0;
    padding: 0.375rem 0.625rem;
    font-family: inherit;
    font-size: 0.8125rem;
    background: var(--bg);
    color: var(--text);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
  }
  .btn-confirm {
    padding: 0.375rem 0.875rem;
    font-family: inherit;
    font-size: 0.8125rem;
    font-weight: 600;
    color: #fff;
    background: var(--danger);
    border: none;
    border-radius: var(--radius-sm);
    cursor: pointer;
  }
  .btn-confirm:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
  .result {
    font-size: 0.8125rem;
    color: var(--text-muted);
    padding-top: 0.375rem;
  }
  .result.failed {
    color: var(--danger);
  }
  @media (max-width: 639px) {
    .row {
      flex-direction: column;
      align-items: flex-start;
    }
  }
</style>
